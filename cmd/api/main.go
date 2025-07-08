package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/zaynkorai/mailflow/internals/ai"
	"github.com/zaynkorai/mailflow/util/gmail"
)

func main() {
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	if googleAPIKey == "" {
		log.Fatal("Error: GOOGLE_API_KEY environment variable is NOT set. Please set it.")
	}

	ctx := context.Background()
	workflowApp, err := ai.NewWorkflow(ctx, googleAPIKey)
	if err != nil {
		log.Fatalf("Failed to set up the workflow: %v", err)
	}

	http.HandleFunc("/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		initialState := ai.GraphState{
			EmailsInfo:         []gmail.EmailInfo{},
			CurrentEmailInfo:   gmail.EmailInfo{},
			EmailCategory:      "",
			GeneratedEmail:     "",
			RAGQueries:         []string{},
			RetrievedDocuments: "",
			WriterMessages:     []string{},
			Sendable:           false,
			Trials:             0,
		}

		initialState.EmailsInfo = append(initialState.EmailsInfo, gmail.EmailInfo{
			ID: "mock_id_1", ThreadID: "mock_thread_1", MessageID: "mock_msg_1",
			Sender: "test@example.com", Subject: "Inquiry about product A",
			Body: "Hi team, I have a question about the features of product A. Could you clarify its compatibility?",
		})

		workflowCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		finalState, err := workflowApp.Graph.Execute(workflowCtx, initialState, 100) // Max 100 steps
		if err != nil {
			log.Printf("Workflow failed to run: %v", err)
			http.Error(w, fmt.Sprintf("Workflow failed: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{"final_state": finalState}); err != nil {
			log.Printf("Failed to encode response: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf("0.0.0.0:%s", port)
	log.Printf("Server starting on %s. Send GET requests to /start.", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
