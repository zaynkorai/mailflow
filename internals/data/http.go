package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

func MakeHTTPHandler(r *mux.Router, endpoints Endpoints) {
	r.HandleFunc("/upload", decodeUploadFileRequest(endpoints.UploadFileEndpoint)).Methods("POST")
	r.HandleFunc("/files", decodeListFilesRequest(endpoints.ListFilesEndpoint)).Methods("GET")
}

func decodeUploadFileRequest(endpoint func(ctx context.Context, request interface{}) (response interface{}, err error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(100 << 20) // 100 MB
		if err != nil {
			fmt.Printf("Error parsing multipart form: %v\n", err)
			encodeErrorResponse(context.Background(), err, w)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			fmt.Println("Error retrieving file from form:", err)
			encodeErrorResponse(context.Background(), err, w)
			return
		}

		req := UploadFileRequest{
			File:       file,
			FileHeader: header,
		}

		resp, err := endpoint(context.Background(), req)
		if err != nil {
			fmt.Printf("Error processing upload request: %v\n", err)
			encodeErrorResponse(context.Background(), err, w)
			return
		}
		fmt.Println("Upload successful, preparing response...")
		encodeResponse(w, resp)
	}
}

func decodeListFilesRequest(endpoint func(ctx context.Context, request interface{}) (response interface{}, err error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := ListFilesRequest{}

		resp, err := endpoint(context.Background(), req)
		if err != nil {
			fmt.Printf("Error processing list files request: %v\n", err)
			encodeErrorResponse(context.Background(), err, w)
			return
		}
		fmt.Println("List files successful, preparing response...")
		encodeResponse(w, resp)
	}
}

func encodeResponse(w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("Error marshaling successful response to JSON bytes: %v\n", err)
		http.Error(w, "Internal Server Error: Failed to encode response", http.StatusInternalServerError)
		return err
	}
	_, err = w.Write(jsonBytes)
	return err
}

func encodeErrorResponse(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	w.WriteHeader(http.StatusInternalServerError)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
