package data

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"mailflow/internals/rag"
	"mailflow/pkg/logging"
)

type DataUploadService interface {
	UploadFile(file multipart.File, header *multipart.FileHeader) (string, error)
	ListFiles() ([]FileInfo, error)
}

type dataUploadService struct {
	ragSystem *rag.RAGSystem // Add RAG system dependency
}

// NewDataUploadService creates a new DataUploadService.
// It now requires a rag.RAGSystem instance to perform indexing.
func NewDataUploadService(ragSystem *rag.RAGSystem) DataUploadService {
	return &dataUploadService{
		ragSystem: ragSystem,
	}
}

func (s *dataUploadService) UploadFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	defer file.Close()

	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create upload directory: %w", err)
	}

	filePath := filepath.Join(uploadDir, header.Filename)
	dst, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	var fileContentBuffer bytes.Buffer
	mw := io.MultiWriter(dst, &fileContentBuffer)

	if _, err := io.Copy(mw, file); err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	fileContent := fileContentBuffer.String()
	if fileContent == "" {
		logging.Info("Uploaded file '%s' is empty, skipping RAG indexing.", header.Filename)
		return fmt.Sprintf("File '%s' uploaded successfully to %s (empty content, not indexed)", header.Filename, filePath), nil
	}

	docID := fmt.Sprintf("uploaded-file-%s-%d", header.Filename, time.Now().UnixNano())
	doc := rag.Document{
		ID:        docID,
		Source:    header.Filename,
		Content:   fileContent,
		CreatedAt: time.Now(),
	}

	logging.Info("Attempting to index uploaded file '%s' into RAG system...", header.Filename)
	err = s.ragSystem.IndexDocument(context.TODO(), doc)
	if err != nil {
		logging.Error("Failed to index uploaded file '%s' into RAG system: %v", header.Filename, err)
		return "", fmt.Errorf("file '%s' uploaded, but failed to index into RAG: %w", header.Filename, err)
	}

	logging.Info("File '%s' uploaded and successfully indexed into RAG system.", header.Filename)
	return fmt.Sprintf("File '%s' uploaded and indexed successfully to %s", header.Filename, filePath), nil
}

func (s *dataUploadService) ListFiles() ([]FileInfo, error) {
	uploadDir := "./uploads"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		return []FileInfo{}, nil
	}

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				logging.Error("Failed to get file info for %s: %v", entry.Name(), err)
				continue
			}
			fileName := entry.Name()
			fileExtension := filepath.Ext(fileName)
			if len(fileExtension) > 0 && fileExtension[0] == '.' {
				fileExtension = fileExtension[1:]
			}
			files = append(files, FileInfo{
				Name:       fileName,
				Extension:  fileExtension,
				Size:       info.Size(),
				UploadTime: info.ModTime(), // Using ModTime as upload time for simplicity
			})
		}
	}
	return files, nil
}
