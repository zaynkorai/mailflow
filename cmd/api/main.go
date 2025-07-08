package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"mailflow/internals/data"

	"github.com/gorilla/mux"
)

func main() {

	svc := data.NewDataUploadService()

	endpoints := data.NewEndpoints(svc)

	r := mux.NewRouter()
	data.MakeHTTPHandler(r, endpoints)

	serveWebBuild(r, "./web/dist")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	addr := fmt.Sprintf(":%s", port)
	log.Printf("File upload service starting on %s", addr)

	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func serveWebBuild(router *mux.Router, staticFilesPath string) {
	router.PathPrefix("/static/").Handler(http.FileServer(http.Dir(staticFilesPath)))
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(staticFilesPath)))
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Serving web for path: %s (NotFoundHandler)", r.URL.Path)
		http.ServeFile(w, r, filepath.Join(staticFilesPath, "index.html"))
	})
}
