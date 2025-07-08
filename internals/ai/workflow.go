package ai

import (
	"context"
	"fmt"
)

type Workflow struct {
	Graph *Graph
}

func NewWorkflow(ctx context.Context, googleAPIKey string) (*Workflow, error) {
	nodesImpl, err := NewNodes(ctx, googleAPIKey)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize nodes implementation: %w", err)
	}

	graph := NewGraph(nodesImpl)

	graph.AddNode("LoadInboxEmails", nodesImpl.LoadNewEmails)
	graph.AddNode("IsEmailInboxEmpty", nodesImpl.IsEmailInboxEmpty)
	graph.AddNode("CategorizeEmail", nodesImpl.CategorizeEmail)
	graph.AddNode("ConstructRagQueries", nodesImpl.ConstructRAGQueries)
	graph.AddNode("RetrieveFromRag", nodesImpl.RetrieveFromRAG)
	graph.AddNode("EmailWriter", nodesImpl.WriteDraftEmail)
	graph.AddNode("EmailProofreader", nodesImpl.VerifyGeneratedEmail)
	graph.AddNode("SendEmail", nodesImpl.CreateDraftResponse)
	graph.AddNode("SkipUnrelatedEmail", nodesImpl.SkipUnrelatedEmail)

	graph.SetEntryPoint("LoadInboxEmails")

	graph.AddEdge("LoadInboxEmails", "IsEmailInboxEmpty")

	// check if there are emails to process (conditional routing)
	graph.AddConditionalEdges(
		"IsEmailInboxEmpty",
		nodesImpl.CheckNewEmails, // This node returns the routing decision
		map[string]string{
			"process": "CategorizeEmail",
			"empty":   GraphEnd, // Map to our custom END sentinel
		},
	)

	// route email based on category (conditional routing)
	graph.AddConditionalEdges(
		"CategorizeEmail",
		nodesImpl.RouteEmailBasedOnCategory, // This node returns the routing decision
		map[string]string{
			"product related":     "ConstructRagQueries",
			"not product related": "EmailWriter", // Feedback or Complaint
			"unrelated":           "SkipUnrelatedEmail",
		},
	)

	// pass constructed queries to RAG chain to retrieve information
	graph.AddEdge("ConstructRagQueries", "RetrieveFromRag")
	// give information to writer agent to create draft email
	graph.AddEdge("RetrieveFromRag", "EmailWriter")
	// proofread the generated draft email
	graph.AddEdge("EmailWriter", "EmailProofreader")

	// check if email is sendable or not, if not rewrite the email (conditional routing)
	graph.AddConditionalEdges(
		"EmailProofreader",
		nodesImpl.MustRewrite, // This node returns the routing decision
		map[string]string{
			"send":    "SendEmail",
			"rewrite": "EmailWriter",
			"stop":    "CategorizeEmail", // LangGraph example loops back to categorize for next email or stop
		},
	)

	// check if there are still emails to be processed (after sending/skipping)
	graph.AddEdge("SendEmail", "IsEmailInboxEmpty")
	graph.AddEdge("SkipUnrelatedEmail", "IsEmailInboxEmpty")

	return &Workflow{Graph: graph.Compile()}, nil
}
