package data

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type DataUploadService interface {
	UploadFile(file multipart.File, header *multipart.FileHeader) (string, error)
	ListFiles() ([]FileInfo, error)
}

type dataUploadService struct{}

func NewDataUploadService() DataUploadService {
	return &dataUploadService{}
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

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}

	return fmt.Sprintf("File '%s' uploaded successfully to %s", header.Filename, filePath), nil
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
			fileName := entry.Name()
			fileExtension := filepath.Ext(fileName)
			if len(fileExtension) > 0 && fileExtension[0] == '.' {
				fileExtension = fileExtension[1:]
			}
			files = append(files, FileInfo{Name: fileName, Extension: fileExtension})
		}
	}
	return files, nil
}
