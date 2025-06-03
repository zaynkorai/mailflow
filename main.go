package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/zaynkorai/mailflow/ai"
	"github.com/zaynkorai/mailflow/util/gmail"
)

func main() {

	ctx := context.Background()
	googleAPIKey := os.Getenv("GOOGLE_API_KEY")
	if googleAPIKey == "" {
		log.Fatal("GOOGLE_API_KEY environment variable not set. Please set it to your Gemini API key.")
	}

	maxIterations := 70 // Corresponds to recursion_limit

	workflowApp, err := ai.NewWorkflow(ctx, googleAPIKey)
	if err != nil {
		log.Fatalf("Failed to initialize workflow: %v", err)
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

	fmt.Println(color.GreenString("Starting workflow..."))

	finalState, err := workflowApp.Graph.Execute(ctx, initialState, maxIterations)
	if err != nil {
		log.Fatalf("Workflow execution failed: %v", err)
	}

	log.Printf("Workflow completed. Final state: %+v\n", finalState)
}
