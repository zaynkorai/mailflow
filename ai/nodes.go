package ai

import (
	"context"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/zaynkorai/mailflow/util/gmail"
)

type Nodes struct {
	Agents *Agents
}

func NewNodes(ctx context.Context, googleAPIKey string) (*Nodes, error) {
	agents, err := NewAgents(ctx, googleAPIKey)
	if err != nil {
		return nil, err
	}
	return &Nodes{
		Agents: agents,
	}, nil
}

func (n *Nodes) LoadNewEmails(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Loading new emails..."))
	gut, _ := gmail.NewGmailUtils()
	recentEmailsData, _ := gut.FetchUnansweredEmails(50)

	state.EmailsInfo = recentEmailsData
	return state, "", nil
}

func (n *Nodes) CheckNewEmails(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	if len(state.EmailsInfo) == 0 {
		fmt.Println(color.RedString("No new emails"))
		return state, "empty", nil
	} else {
		fmt.Println(color.GreenString("New emails to process"))
		return state, "process", nil
	}
}

func (n *Nodes) IsEmailInboxEmpty(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	// This node's primary purpose is to just mark a point in the graph,
	// The actual check and routing decision happens in CheckNewEmails.
	return state, "", nil
}

func (n *Nodes) CategorizeEmail(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Checking email category..."))
	if len(state.EmailsInfo) == 0 {
		return state, "", fmt.Errorf("error: No emails in state to categorize")
	}
	currentEmail := state.EmailsInfo[len(state.EmailsInfo)-1]
	result, err := n.Agents.CategorizeEmail(ctx, currentEmail.Body)
	if err != nil {
		return state, "", fmt.Errorf("error categorizing email: %w", err)
	}
	fmt.Println(color.MagentaString("Email category: %s", result.Category))
	state.EmailCategory = string(result.Category)
	state.CurrentEmailInfo = currentEmail
	return state, "", nil
}

func (n *Nodes) RouteEmailBasedOnCategory(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Routing email based on category..."))
	category := EmailCategory(state.EmailCategory)
	switch category {
	case ProductEnquiry:
		return state, "product related", nil
	case Unrelated:
		return state, "unrelated", nil
	default:
		return state, "not product related", nil
	}
}

func (n *Nodes) ConstructRAGQueries(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Designing RAG query..."))
	emailContent := state.CurrentEmailInfo.Body
	queryResult, err := n.Agents.DesignRAGQueries(ctx, emailContent)
	if err != nil {
		return state, "", fmt.Errorf("error designing RAG queries: %w", err)
	}
	state.RAGQueries = queryResult.Queries
	return state, "", nil
}

func (n *Nodes) RetrieveFromRAG(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Retrieving information from internal knowledge..."))
	finalAnswer := strings.Builder{}
	for _, query := range state.RAGQueries {
		ragResult, err := n.Agents.GenerateRAGAnswer(ctx, "", query)
		if err != nil {
			return state, "", fmt.Errorf("error generating RAG answer for query '%s': %w", query, err)
		}
		finalAnswer.WriteString(query)
		finalAnswer.WriteString("\n")
		finalAnswer.WriteString(ragResult)
		finalAnswer.WriteString("\n\n")
	}
	state.RetrievedDocuments = finalAnswer.String()
	return state, "", nil
}

func (n *Nodes) WriteDraftEmail(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Writing draft email..."))
	inputs := fmt.Sprintf(
		"# **EMAIL CATEGORY:** %s\n\n# **EMAIL CONTENT:**\n%s\n\n# **INFORMATION:**\n%s",
		state.EmailCategory,
		state.CurrentEmailInfo.Body,
		state.RetrievedDocuments,
	)
	if state.WriterMessages == nil {
		state.WriterMessages = []string{}
	}
	draftResult, err := n.Agents.EmailWriter(ctx, inputs, state.WriterMessages)
	if err != nil {
		return state, "", fmt.Errorf("error writing email draft: %w", err)
	}
	email := draftResult.Email
	state.Trials++
	state.WriterMessages = append(state.WriterMessages, fmt.Sprintf("**Draft %d:**\n%s", state.Trials, email))
	state.GeneratedEmail = email
	return state, "", nil
}

func (n *Nodes) VerifyGeneratedEmail(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Verifying generated email..."))
	review, err := n.Agents.EmailProofreader(ctx, state.CurrentEmailInfo.Body, state.GeneratedEmail)
	if err != nil {
		return state, "", fmt.Errorf("error verifying generated email: %w", err)
	}
	if state.WriterMessages == nil {
		state.WriterMessages = []string{}
	}
	state.WriterMessages = append(state.WriterMessages, fmt.Sprintf("**Proofreader Feedback:**\n%s", review.Feedback))
	state.Sendable = review.Send
	return state, "", nil
}

func (n *Nodes) MustRewrite(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	emailSendable := state.Sendable
	if emailSendable {
		fmt.Println(color.GreenString("Email is good, ready to be sent!!!"))
		if len(state.EmailsInfo) > 0 {
			state.EmailsInfo = state.EmailsInfo[:len(state.EmailsInfo)-1]
		}
		state.WriterMessages = []string{}
		return state, "send", nil
	} else if state.Trials >= 3 {
		fmt.Println(color.RedString("Email is not good, we reached max trials must stop!!!"))
		if len(state.EmailsInfo) > 0 {
			state.EmailsInfo = state.EmailsInfo[:len(state.EmailsInfo)-1]
		}
		state.WriterMessages = []string{}
		return state, "stop", nil
	} else {
		fmt.Println(color.RedString("Email is not good, must rewrite it..."))
		return state, "rewrite", nil
	}
}

func (n *Nodes) CreateDraftResponse(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Creating draft email..."))
	gut, _ := gmail.NewGmailUtils()

	_, err := gut.CreateDraftReply(state.CurrentEmailInfo, state.GeneratedEmail)
	if err != nil {
		return state, "", fmt.Errorf("error creating draft reply: %w", err)
	}
	state.RetrievedDocuments = ""
	state.Trials = 0
	return state, "", nil
}

func (n *Nodes) SendEmailResponse(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println(color.YellowString("Sending email..."))
	gut, _ := gmail.NewGmailUtils()
	_, err := gut.SendReply(state.CurrentEmailInfo, state.GeneratedEmail)
	if err != nil {
		return state, "", fmt.Errorf("error sending email: %w", err)
	}
	state.RetrievedDocuments = ""
	state.Trials = 0
	return state, "", nil
}

func (n *Nodes) SkipUnrelatedEmail(ctx context.Context, state *GraphState) (*GraphState, string, error) {
	fmt.Println("Skipping unrelated email...")
	if len(state.EmailsInfo) > 0 {
		state.EmailsInfo = state.EmailsInfo[:len(state.EmailsInfo)-1]
	}
	return state, "", nil
}
