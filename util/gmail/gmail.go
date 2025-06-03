package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

const (
	tokenFile       = "token.json"
	credentialsFile = "credentials.json"
)

var (
	SCOPES = []string{gmail.GmailModifyScope}
)

type EmailInfo struct {
	ID         string `json:"id"`
	ThreadID   string `json:"threadId"`
	MessageID  string `json:"messageId"`
	References string `json:"references"`
	Sender     string `json:"sender"`
	Subject    string `json:"subject"`
	Body       string `json:"body"`
}

type DraftInfo struct {
	DraftID   string
	ThreadID  string
	MessageID string
}

type GmailUtils struct {
	service *gmail.Service
	myEmail string // Stores the user's own email address for skipping self-sent emails.
}

func NewGmailUtils() (*GmailUtils, error) {
	ctx := context.Background()

	b, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, SCOPES...)
	if err != nil {
		return nil, fmt.Errorf("unable to parse client secret file to config: %v", err)
	}

	tok, err := getToken(config)
	if err != nil {
		return nil, fmt.Errorf("unable to get token: %v", err)
	}

	client := config.Client(ctx, tok)

	srv, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve Gmail client: %v", err)
	}

	myEmail := os.Getenv("MY_EMAIL")
	if myEmail == "" {
		log.Println("WARNING: MY_EMAIL environment variable not set. Self-sent emails may not be skipped.")
	}

	return &GmailUtils{service: srv, myEmail: myEmail}, nil
}

func getToken(config *oauth2.Config) (*oauth2.Token, error) {
	tok, err := tokenFromFile(tokenFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFile, tok)
	}
	return tok, nil
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(context.TODO(), authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}

// FetchUnansweredEmails fetches all emails included in unanswered threads.
// An "unanswered" thread is one that doesn't have a draft reply yet.
func (gut *GmailUtils) FetchUnansweredEmails(maxResults int64) ([]EmailInfo, error) {
	log.Printf("Fetching unanswered emails (maxResults: %d)...", maxResults)

	recentEmails, err := gut.FetchRecentEmails(maxResults)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent emails: %w", err)
	}
	if len(recentEmails) == 0 {
		log.Println("No recent emails found.")
		return []EmailInfo{}, nil
	}

	drafts, err := gut.FetchDraftReplies()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch draft replies: %w", err)
	}

	threadsWithDrafts := make(map[string]bool)
	for _, draft := range drafts {
		threadsWithDrafts[draft.ThreadID] = true
	}
	log.Printf("Found %d threads with existing drafts.", len(threadsWithDrafts))

	seenThreads := make(map[string]bool)
	var unansweredEmails []EmailInfo
	for _, email := range recentEmails {
		threadID := email.ThreadId
		// Check if the thread has already been processed or has a draft
		if !seenThreads[threadID] && !threadsWithDrafts[threadID] {
			seenThreads[threadID] = true // Mark thread as seen to avoid duplicates
			emailInfo, err := gut.GetEmailInfo(email.Id)
			if err != nil {
				log.Printf("Error getting email info for ID %s: %v", email.Id, err)
				continue
			}

			if gut.ShouldSkipEmail(emailInfo) {
				log.Printf("Skipping email from sender: %s (ID: %s)", emailInfo.Sender, emailInfo.ID)
				continue
			}
			unansweredEmails = append(unansweredEmails, emailInfo)
		}
	}
	log.Printf("Found %d unanswered emails.", len(unansweredEmails))
	return unansweredEmails, nil
}

func (gut *GmailUtils) FetchRecentEmails(maxResults int64) ([]*gmail.Message, error) {
	log.Printf("Fetching recent emails (maxResults: %d)...", maxResults)
	now := time.Now()

	delay := now.Add(-8 * time.Hour)

	afterTimestamp := delay.Unix()
	beforeTimestamp := now.Unix()

	query := fmt.Sprintf("after:%d before:%d", afterTimestamp, beforeTimestamp)
	log.Printf("Gmail query: %s", query)

	results, err := gut.service.Users.Messages.List("me").Q(query).MaxResults(maxResults).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve messages: %w", err)
	}

	messages := results.Messages
	if len(messages) == 0 {
		log.Println("No messages found for the query.")
	} else {
		log.Printf("Found %d recent messages.", len(messages))
	}
	return messages, nil
}

func (gut *GmailUtils) FetchDraftReplies() ([]DraftInfo, error) {
	log.Println("Fetching draft replies...")
	draftsResponse, err := gut.service.Users.Drafts.List("me").Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve drafts: %w", err)
	}

	var draftList []DraftInfo
	if draftsResponse.Drafts != nil {
		for _, draft := range draftsResponse.Drafts {
			if draft.Message != nil {
				draftList = append(draftList, DraftInfo{
					DraftID:   draft.Id,
					ThreadID:  draft.Message.ThreadId,
					MessageID: draft.Message.Id,
				})
			}
		}
	}
	log.Printf("Found %d draft replies.", len(draftList))
	return draftList, nil
}

func (gut *GmailUtils) CreateDraftReply(initialEmail EmailInfo, replyText string) (*gmail.Draft, error) {
	log.Printf("Creating draft reply for email ID: %s", initialEmail.ID)

	message, err := gut.createReplyMessage(initialEmail, replyText, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create reply message: %w", err)
	}

	draft, err := gut.service.Users.Drafts.Create("me", &gmail.Draft{Message: message}).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to create draft: %w", err)
	}
	log.Printf("Draft created with ID: %s", draft.Id)
	return draft, nil
}

func (gut *GmailUtils) SendReply(initialEmail EmailInfo, replyText string) (*gmail.Message, error) {
	log.Printf("Sending reply for email ID: %s", initialEmail.ID)

	message, err := gut.createReplyMessage(initialEmail, replyText, true) // true for sending
	if err != nil {
		return nil, fmt.Errorf("failed to create reply message: %w", err)
	}

	sentMessage, err := gut.service.Users.Messages.Send("me", message).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to send message: %w", err)
	}
	log.Printf("Reply sent with message ID: %s", sentMessage.Id)
	return sentMessage, nil
}

func (gut *GmailUtils) createReplyMessage(email EmailInfo, replyText string, send bool) (*gmail.Message, error) {

	msg, bodyContent, err := gut.createHTMLEmailMessage(email.Sender, email.Subject, replyText)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTML email message: %w", err)
	}

	if email.MessageID != "" {
		msg.Header["In-Reply-To"] = []string{email.MessageID}
		references := strings.TrimSpace(fmt.Sprintf("%s %s", email.References, email.MessageID))
		msg.Header["References"] = []string{references}

		if send {
			msg.Header["Message-ID"] = []string{fmt.Sprintf("<%s@gmail.com>", uuid.New().String())}
		}
	}

	var rawEmail strings.Builder
	for k, v := range msg.Header {
		for _, s := range v {
			rawEmail.WriteString(fmt.Sprintf("%s: %s\r\n", k, s))
		}
	}
	rawEmail.WriteString("\r\n")
	rawEmail.WriteString(bodyContent)

	raw := base64.URLEncoding.EncodeToString([]byte(rawEmail.String()))

	return &gmail.Message{
		Raw:      raw,
		ThreadId: email.ThreadID,
	}, nil
}

func (gut *GmailUtils) ShouldSkipEmail(emailInfo EmailInfo) bool {
	if gut.myEmail == "" {
		return false
	}
	return strings.Contains(emailInfo.Sender, gut.myEmail)
}

func (gut *GmailUtils) GetEmailInfo(msgID string) (EmailInfo, error) {
	log.Printf("Getting email info for message ID: %s", msgID)
	message, err := gut.service.Users.Messages.Get("me", msgID).Format("full").Do()
	if err != nil {
		return EmailInfo{}, fmt.Errorf("unable to retrieve message %s: %w", msgID, err)
	}

	payload := message.Payload
	if payload == nil {
		return EmailInfo{}, fmt.Errorf("message payload is nil for ID %s", msgID)
	}

	headers := make(map[string]string)
	for _, header := range payload.Headers {
		headers[strings.ToLower(header.Name)] = header.Value
	}

	body, err := gut.GetEmailBody(payload)
	if err != nil {
		log.Printf("Warning: Failed to get email body for ID %s: %v", msgID, err)
		body = ""
	}

	return EmailInfo{
		ID:         msgID,
		ThreadID:   message.ThreadId,
		MessageID:  headers["message-id"],
		References: headers["references"],
		Sender:     headers["from"],
		Subject:    headers["subject"],
		Body:       body,
	}, nil
}

func (gut *GmailUtils) GetEmailBody(payload *gmail.MessagePart) (string, error) {

	decodeData := func(data string) string {
		if data == "" {
			return ""
		}
		decoded, err := base64.URLEncoding.DecodeString(data)
		if err != nil {
			log.Printf("Error decoding base64 data: %v", err)
			return ""
		}
		return strings.TrimSpace(string(decoded))
	}

	var extractBody func(parts []*gmail.MessagePart) string
	extractBody = func(parts []*gmail.MessagePart) string {
		for _, part := range parts {
			mimeType := part.MimeType
			data := part.Body.Data
			if mimeType == "text/plain" {
				return decodeData(data)
			}
			if mimeType == "text/html" {
				htmlContent := decodeData(data)
				return gut.extractMainContentFromHTML(htmlContent)
			}
			if len(part.Parts) > 0 {
				result := extractBody(part.Parts)
				if result != "" {
					return result
				}
			}
		}
		return ""
	}

	var body string
	if len(payload.Parts) > 0 {
		body = extractBody(payload.Parts)
	} else {
		data := payload.Body.Data
		body = decodeData(data)
		if payload.MimeType == "text/html" {
			body = gut.extractMainContentFromHTML(body)
		}
	}

	return gut.cleanBodyText(body), nil
}

func (gut *GmailUtils) extractMainContentFromHTML(htmlContent string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML content: %v", err)
		return htmlContent // Return original content if parsing fails
	}
	doc.Find("script, style, head, meta, title").Remove()

	return strings.TrimSpace(doc.Text())
}

func (gut *GmailUtils) cleanBodyText(text string) string {

	reSpaces := regexp.MustCompile(`\s+`)
	cleanedText := reSpaces.ReplaceAllString(text, " ")
	return strings.TrimSpace(cleanedText)
}

func (gut *GmailUtils) createHTMLEmailMessage(recipient, subject, replyText string) (*mail.Message, string, error) {
	msg := &mail.Message{
		Header: make(mail.Header),
	}
	msg.Header["To"] = []string{recipient}
	if !strings.HasPrefix(subject, "Re: ") {
		msg.Header["Subject"] = []string{"Re: " + subject}
	} else {
		msg.Header["Subject"] = []string{subject}
	}
	msg.Header["MIME-Version"] = []string{"1.0"}

	var b strings.Builder
	mw := multipart.NewWriter(&b)
	boundary := mw.Boundary()
	msg.Header["Content-Type"] = []string{fmt.Sprintf(`multipart/alternative; boundary="%s"`, boundary)}

	htmlPartHeader := make(textproto.MIMEHeader)
	htmlPartHeader.Set("Content-Type", "text/html; charset=UTF-8")
	htmlPartHeader.Set("Content-Transfer-Encoding", "quoted-printable")
	htmlPart, err := mw.CreatePart(htmlPartHeader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create HTML part: %w", err)
	}

	htmlText := strings.ReplaceAll(replyText, "\n", "<br>")
	htmlContent := fmt.Sprintf(`
		<!DOCTYPE html>
		<html>
		<head>
			<meta charset="utf-8">
			<meta name="viewport" content="width=device-width, initial-scale=1.0">
		</head>
		<body>%s</body>
		</html>
		`, htmlText)
	qpWriter := quotedprintable.NewWriter(htmlPart)
	_, err = qpWriter.Write([]byte(htmlContent))
	if err != nil {
		return nil, "", fmt.Errorf("failed to write HTML content to quoted-printable writer: %w", err)
	}
	qpWriter.Close()

	mw.Close()

	return msg, b.String(), nil
}
