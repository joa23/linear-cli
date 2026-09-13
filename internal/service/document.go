package service

import (
	"fmt"
	"strings"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/pkg/linear"
	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/documents"
	"github.com/joa23/linear-cli/pkg/linear/identifiers"
)

// DocumentService handles document-related operations
type DocumentService struct {
	client    *linear.Client
	formatter *format.Formatter
}

// DocumentServiceInterface defines the operations for document management
type DocumentServiceInterface interface {
	List(params *DocumentListParams) (string, error)
	Get(ref string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Create(params *DocumentCreateParams) (string, error)
	Update(ref string, params *DocumentUpdateParams) (string, error)
	Delete(ref string) (string, error)
}

// DocumentListParams holds CLI-level parameters for listing documents.
// Project, Issue, and Team are human identifiers (name, key, TEC-123) and are resolved here.
type DocumentListParams struct {
	Project         string
	Issue           string
	Team            string
	Query           string
	IncludeArchived bool
	Limit           int
	Verbosity       format.Verbosity
	OutputType      format.OutputType
}

// DocumentCreateParams holds CLI-level parameters for creating a document.
// The parent is Issue, else Project, else Team. Team alongside Project only scopes
// project name resolution. Project and Issue together is an error.
type DocumentCreateParams struct {
	Title      string
	Content    string
	ContentSet bool // true when the caller supplied content (so "" is sent explicitly)
	Project    string
	Issue      string
	Team       string
	Icon       string
	Color      string
	Verbosity  format.Verbosity
	OutputType format.OutputType
}

// DocumentUpdateParams holds CLI-level parameters for updating a document. nil = unchanged.
type DocumentUpdateParams struct {
	Title      *string
	Content    *string
	Project    *string
	Issue      *string
	Team       *string
	Icon       *string
	Color      *string
	Verbosity  format.Verbosity
	OutputType format.OutputType
}

// NewDocumentService creates a new DocumentService
func NewDocumentService(client *linear.Client, formatter *format.Formatter) *DocumentService {
	return &DocumentService{
		client:    client,
		formatter: formatter,
	}
}

// List returns a formatted list of documents matching the params
func (s *DocumentService) List(params *DocumentListParams) (string, error) {
	projectID, issueID, teamID, err := s.resolveParents(params.Project, params.Issue, params.Team)
	if err != nil {
		return "", err
	}

	filter := &documents.ListFilter{
		ProjectID:       projectID,
		IssueID:         issueID,
		TeamID:          teamID,
		Query:           params.Query,
		IncludeArchived: params.IncludeArchived,
	}

	result, err := s.client.Documents.ListDocuments(filter, params.Limit)
	if err != nil {
		return "", err
	}

	page := &format.Pagination{HasNextPage: result.HasNextPage, EndCursor: result.EndCursor}
	return s.formatter.RenderDocumentList(result.Documents, params.Verbosity, params.OutputType, page), nil
}

// Get returns a formatted document. ref may be a UUID, a slug, or a Linear document URL.
func (s *DocumentService) Get(ref string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	doc, err := s.fetchDocument(ref)
	if err != nil {
		return "", err
	}
	return s.formatter.RenderDocument(doc, verbosity, outputType), nil
}

// Create creates a new document under exactly one parent
func (s *DocumentService) Create(params *DocumentCreateParams) (string, error) {
	if strings.TrimSpace(params.Title) == "" {
		return "", fmt.Errorf("--title is required")
	}
	if params.Project == "" && params.Issue == "" && params.Team == "" {
		return "", fmt.Errorf("a parent is required: --project, --issue, or --team")
	}

	projectID, issueID, teamID, err := s.resolveParent(params.Project, params.Issue, params.Team)
	if err != nil {
		return "", err
	}

	input := &documents.CreateInput{
		Title:     params.Title,
		ProjectID: projectID,
		IssueID:   issueID,
		TeamID:    teamID,
		Icon:      params.Icon,
		Color:     params.Color,
	}
	if params.ContentSet {
		content := params.Content
		input.Content = &content
	}

	doc, err := s.client.Documents.CreateDocument(input)
	if err != nil {
		return "", err
	}

	return s.formatter.RenderDocument(doc, params.Verbosity, params.OutputType), nil
}

// Update updates an existing document. ref may be a UUID, a slug, or a Linear document URL.
func (s *DocumentService) Update(ref string, params *DocumentUpdateParams) (string, error) {
	if params.Title == nil && params.Content == nil && params.Project == nil &&
		params.Issue == nil && params.Team == nil && params.Icon == nil && params.Color == nil {
		return "", fmt.Errorf("nothing to update: provide at least one of --title, --content, --content-file, --project, --issue, --team, --icon, --color")
	}
	if params.Title != nil && strings.TrimSpace(*params.Title) == "" {
		return "", fmt.Errorf("--title cannot be empty")
	}

	docID, err := s.resolveDocumentID(ref)
	if err != nil {
		return "", err
	}

	input := &documents.UpdateInput{
		Title:   params.Title,
		Content: params.Content,
		Icon:    params.Icon,
		Color:   params.Color,
	}

	// Reparenting: resolve the new parent and clear the other two so the document has one parent.
	if params.Project != nil || params.Issue != nil || params.Team != nil {
		projectID, issueID, teamID, err := s.resolveParent(deref(params.Project), deref(params.Issue), deref(params.Team))
		if err != nil {
			return "", err
		}
		input.ProjectID = &projectID
		input.IssueID = &issueID
		input.TeamID = &teamID
	}

	doc, err := s.client.Documents.UpdateDocument(docID, input)
	if err != nil {
		return "", err
	}

	return s.formatter.RenderDocument(doc, params.Verbosity, params.OutputType), nil
}

// Delete moves a document to trash and returns a confirmation message
func (s *DocumentService) Delete(ref string) (string, error) {
	doc, err := s.fetchDocument(ref)
	if err != nil {
		return "", err
	}

	if err := s.client.Documents.DeleteDocument(doc.ID); err != nil {
		return "", err
	}

	return fmt.Sprintf("Deleted document %q (%s). Linear moves it to trash; restore from the Linear UI.", doc.Title, doc.ID), nil
}

// fetchDocument loads a document by UUID, slug, or URL
func (s *DocumentService) fetchDocument(ref string) (*core.Document, error) {
	uuid, slug := parseDocumentRef(ref)
	if uuid != "" {
		return s.client.Documents.GetDocument(uuid)
	}
	if slug == "" {
		return nil, fmt.Errorf("invalid document reference: %s (expected UUID, slug, or Linear document URL)", ref)
	}
	doc, err := s.client.Documents.GetDocumentBySlug(slug)
	if err != nil {
		return nil, fmt.Errorf("document not found: %s", ref)
	}
	return doc, nil
}

// resolveDocumentID converts a UUID, slug, or URL into a document UUID
func (s *DocumentService) resolveDocumentID(ref string) (string, error) {
	if uuid, _ := parseDocumentRef(ref); uuid != "" {
		return uuid, nil
	}
	doc, err := s.fetchDocument(ref)
	if err != nil {
		return "", err
	}
	return doc.ID, nil
}

// parseDocumentRef splits a reference into a UUID or a slug.
// Accepts: a UUID, a bare slug, "<title-slug>-<slugId>", or a Linear document URL.
// The slug is the token after the last "-" in the last URL path segment.
func parseDocumentRef(ref string) (uuid string, slug string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ""
	}
	if identifiers.IsUUID(ref) {
		return ref, ""
	}

	token := ref
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		trimmed := strings.TrimRight(ref, "/")
		if i := strings.Index(trimmed, "?"); i >= 0 {
			trimmed = trimmed[:i]
		}
		if i := strings.Index(trimmed, "#"); i >= 0 {
			trimmed = trimmed[:i]
		}
		token = trimmed[strings.LastIndex(trimmed, "/")+1:]
	}

	if i := strings.LastIndex(token, "-"); i >= 0 {
		token = token[i+1:]
	}
	return "", token
}

// resolveParent picks exactly one parent and resolves it to a UUID.
// Precedence: issue, then project, then team. A team given with a project or
// issue only scopes project name resolution and is not returned as parent.
func (s *DocumentService) resolveParent(project, issue, team string) (projectID, issueID, teamID string, err error) {
	if project != "" && issue != "" {
		return "", "", "", fmt.Errorf("only one parent is allowed: --project or --issue")
	}
	projectID, issueID, teamID, err = s.resolveParents(project, issue, team)
	if err != nil {
		return "", "", "", err
	}
	switch {
	case issueID != "":
		return "", issueID, "", nil
	case projectID != "":
		return projectID, "", "", nil
	default:
		return "", "", teamID, nil
	}
}

// resolveParents resolves human identifiers to UUIDs. Empty inputs stay empty.
// A resolved team also scopes project name resolution.
func (s *DocumentService) resolveParents(project, issue, team string) (projectID, issueID, teamID string, err error) {
	if team != "" {
		if identifiers.IsUUID(team) {
			teamID = team
		} else {
			teamID, err = s.client.ResolveTeamIdentifier(team)
			if err != nil {
				return "", "", "", fmt.Errorf("failed to resolve team '%s': %w", team, err)
			}
		}
	}

	if project != "" {
		projectID, err = s.client.ResolveProjectIdentifier(project, teamID)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to resolve project '%s': %w", project, err)
		}
	}

	if issue != "" {
		issueID, err = s.resolveIssueID(issue)
		if err != nil {
			return "", "", "", err
		}
	}

	return projectID, issueID, teamID, nil
}

// resolveIssueID resolves an issue identifier (e.g., "TEC-123") to UUID
func (s *DocumentService) resolveIssueID(issueID string) (string, error) {
	if identifiers.IsUUID(issueID) {
		return issueID, nil
	}
	if identifiers.IsIssueIdentifier(issueID) {
		resolved, err := s.client.ResolveIssueIdentifier(issueID)
		if err != nil {
			return "", fmt.Errorf("failed to resolve issue '%s': %w", issueID, err)
		}
		return resolved, nil
	}
	return "", fmt.Errorf("invalid issue identifier: %s (expected format like TEC-123 or UUID)", issueID)
}

// deref returns the string value or "" for nil
func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
