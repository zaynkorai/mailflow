package data

import (
	"context"
	"mime/multipart"
	"time"
)

type Endpoints struct {
	UploadFileEndpoint func(ctx context.Context, request interface{}) (response interface{}, err error)
	ListFilesEndpoint  func(ctx context.Context, request interface{}) (response interface{}, err error)
}

func NewEndpoints(s DataUploadService) Endpoints {
	return Endpoints{
		UploadFileEndpoint: MakeUploadFileEndpoint(s),
		ListFilesEndpoint:  MakeListFilesEndpoint(s),
	}
}

func MakeUploadFileEndpoint(s DataUploadService) func(ctx context.Context, request interface{}) (response interface{}, err error) {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(UploadFileRequest)
		msg, err := s.UploadFile(req.File, req.FileHeader)
		return UploadFileResponse{Message: msg, Err: err}, nil
	}
}

func MakeListFilesEndpoint(s DataUploadService) func(ctx context.Context, request interface{}) (response interface{}, err error) {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		files, err := s.ListFiles()
		if err != nil {
			return nil, err
		}
		return files, nil
	}
}

type UploadFileRequest struct {
	File       multipart.File
	FileHeader *multipart.FileHeader
}

type UploadFileResponse struct {
	Message string `json:"message"`
	Err     error  `json:"error,omitempty"`
}

func (r UploadFileResponse) Error() string {
	if r.Err == nil {
		return ""
	}
	return r.Err.Error()
}

type ListFilesRequest struct{}

type FileInfo struct {
	Name       string    `json:"name"`
	Extension  string    `json:"extension"`
	Size       int64     `json:"size"`
	UploadTime time.Time `json:"upload_time"`
}
