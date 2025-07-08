package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"mailflow/internals/config"
	"mailflow/internals/data"
	"mailflow/internals/llm"
	"mailflow/internals/rag"
	"mailflow/internals/rag/adapter"
	"mailflow/pkg/logging"

	"github.com/gorilla/mux"
)

func main() {
	logging.InitLogger()
	logging.Info("Starting Mailflow API service...")

	cfg, err := config.LoadConfig()
	if err != nil {
		logging.Fatal("Failed to load configuration: %v", err)
	}
	logging.Info("Configuration loaded successfully. Port: %d, Google API Key: %s (first 5 chars)", cfg.Port, cfg.GoogleAPIKey[:5])

	geminiEmbedder := llm.NewGeminiEmbedder(cfg.GoogleAPIKey)
	vectorStore := adapter.NewInMemoryVectorStore()
	chunker := rag.NewSimpleTextChunker(rag.DefaultChunkSize, rag.DefaultChunkOverlap)
	ragSystem := rag.NewRAGSystem(chunker, geminiEmbedder, vectorStore)
	logging.Info("RAG system initialized for API service.")

	dataSvc := data.NewDataUploadService(ragSystem)
	logging.Info("Data upload service initialized for API service.")

	endpoints := data.NewEndpoints(dataSvc)

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
