package aime

import (
	"encoding/json"
	"time"
)

type CreateReportRequest struct {
	Answers      json.RawMessage `json:"answers"`
	AttachReport bool            `json:"attachReport"`
	Email        string          `json:"email"`
	Public       bool            `json:"isPublic"`
}

type CreateRevisionRequest struct {
	Answers      json.RawMessage `json:"answers"`
	AttachReport bool            `json:"attachReport"`
	Email        string          `json:"email"`
	Password     string          `json:"password"`
	Public       bool            `json:"isPublic"`
}

type CreateRevisionResponse struct {
	ID       string `json:"id"`
	Password string `json:"password"`
	Version  int    `json:"version"`
}

type CreateIssueRequest struct {
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Field   []string `json:"field"`
	Content string   `json:"content"`
	Type    int      `json:"type"`
}

type CreateIssueResponse struct {
	ID       int    `json:"id"`
	Password string `json:"password"`
}

type CreateAnswerRequest struct {
	Content string `json:"content"`
}

type CreateAnswerResponse struct {
	ID int `json:"id"`
}

type RevisionInfo struct {
	Revision  int       `json:"revision"`
	CreatedAt time.Time `json:"createdAt"`
	Public    bool      `json:"isPublic"`
}

type IssueInfo struct {
	ID         int       `json:"id"`
	RevisionID int       `json:"revisionId"`
	CreatedAt  time.Time `json:"createdAt"`
	Name       string    `json:"name"`
	Type       int       `json:"type"`
}

type GetRevisionResponse struct {
	Answers json.RawMessage `json:"answers"`

	Revision  int       `json:"revision"`
	CreatedAt time.Time `json:"createdAt"`
	Public    bool      `json:"isPublic"`

	Revisions []RevisionInfo `json:"revisions"`
	Issues    []IssueInfo    `json:"issues"`
}

type UploadFileResponse struct {
	File string `json:"file"`
}

type Result struct {
	ID        string    `json:"id"`
	UpdatedAt time.Time `json:"date"`
	Title     string    `json:"title"`
	Authors   []string  `json:"authors"`
	Revisions int       `json:"revisions"`
	Issues    int       `json:"issues"`
}

type SearchResponse struct {
	Count   int      `json:"count"`
	Results []Result `json:"results"`
	Query   string   `json:"query"`
}

type Keyword struct {
	Keyword string `json:"keyword"`
	Count   int    `json:"count"`
}

type KeywordsResponse struct {
	Keywords []Keyword `json:"keywords"`
}

type KeywordListResponse []string

type KeywordResponse struct {
	Count   int      `json:"count"`
	Results []Result `json:"results"`
	Keyword string   `json:"keyword"`
}
