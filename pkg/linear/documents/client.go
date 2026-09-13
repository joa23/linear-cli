package documents

import (
	"fmt"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// Client handles document-related operations
type Client struct {
	base *core.BaseClient
}

// NewClient creates a new documents client
func NewClient(base *core.BaseClient) *Client {
	return &Client{base: base}
}

// documentListFields omits content — documents can be large and list output never shows it.
const documentListFields = `
	id
	slugId
	title
	icon
	color
	url
	createdAt
	updatedAt
	archivedAt
	creator { id name displayName }
	updatedBy { id name displayName }
	project { id name }
	issue { id identifier title }
	team { id key name }
`

// documentFields is the full selection set, including content.
const documentFields = `
	content
` + documentListFields

// ListFilter holds filter criteria for listing documents. All IDs must be UUIDs.
type ListFilter struct {
	ProjectID       string
	IssueID         string
	TeamID          string
	Query           string // Case-insensitive title substring match
	IncludeArchived bool
}

// ListResult holds a page of documents.
type ListResult struct {
	Documents   []core.Document
	HasNextPage bool
	EndCursor   string
}

// CreateInput holds parameters for creating a document. Exactly one parent ID should be set.
type CreateInput struct {
	Title     string  // Required
	Content   *string // nil = omit
	ProjectID string
	IssueID   string
	TeamID    string
	Icon      string
	Color     string
}

// UpdateInput holds parameters for updating a document. nil fields are not sent.
// For ProjectID, IssueID, and TeamID a pointer to "" is sent as null, which clears that parent.
type UpdateInput struct {
	Title     *string
	Content   *string
	ProjectID *string
	IssueID   *string
	TeamID    *string
	Icon      *string
	Color     *string
}

// buildFilter converts a ListFilter into a GraphQL DocumentFilter map.
// Returns nil when no criteria are set so the variable is omitted entirely.
func buildFilter(f *ListFilter) map[string]interface{} {
	if f == nil {
		return nil
	}
	filter := map[string]interface{}{}
	if f.ProjectID != "" {
		filter["project"] = map[string]interface{}{"id": map[string]interface{}{"eq": f.ProjectID}}
	}
	if f.IssueID != "" {
		filter["issue"] = map[string]interface{}{"id": map[string]interface{}{"eq": f.IssueID}}
	}
	if f.TeamID != "" {
		filter["team"] = map[string]interface{}{"id": map[string]interface{}{"eq": f.TeamID}}
	}
	if f.Query != "" {
		filter["title"] = map[string]interface{}{"containsIgnoreCase": f.Query}
	}
	if len(filter) == 0 {
		return nil
	}
	return filter
}

// ListDocuments returns documents matching the filter, newest updated first.
func (dc *Client) ListDocuments(filter *ListFilter, limit int) (*ListResult, error) {
	const query = `
		query Documents($first: Int!, $filter: DocumentFilter, $includeArchived: Boolean) {
			documents(first: $first, filter: $filter, includeArchived: $includeArchived, orderBy: updatedAt) {
				nodes {` + documentListFields + `}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`

	if limit <= 0 {
		limit = 50
	}

	variables := map[string]interface{}{
		"first": limit,
	}
	if f := buildFilter(filter); f != nil {
		variables["filter"] = f
	}
	if filter != nil && filter.IncludeArchived {
		variables["includeArchived"] = true
	}

	var response struct {
		Documents struct {
			Nodes    []core.Document `json:"nodes"`
			PageInfo struct {
				HasNextPage bool   `json:"hasNextPage"`
				EndCursor   string `json:"endCursor"`
			} `json:"pageInfo"`
		} `json:"documents"`
	}

	if err := dc.base.ExecuteRequest(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to list documents: %w", err)
	}

	return &ListResult{
		Documents:   response.Documents.Nodes,
		HasNextPage: response.Documents.PageInfo.HasNextPage,
		EndCursor:   response.Documents.PageInfo.EndCursor,
	}, nil
}

// GetDocument fetches a document by UUID, including its content.
func (dc *Client) GetDocument(id string) (*core.Document, error) {
	if id == "" {
		return nil, &core.ValidationError{Field: "id", Message: "document ID is required"}
	}

	const query = `
		query Document($id: String!) {
			document(id: $id) {` + documentFields + `}
		}
	`

	var response struct {
		Document core.Document `json:"document"`
	}

	if err := dc.base.ExecuteRequest(query, map[string]interface{}{"id": id}, &response); err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return &response.Document, nil
}

// GetDocumentBySlug fetches a document by its slugId (the trailing token in a Linear document URL).
func (dc *Client) GetDocumentBySlug(slug string) (*core.Document, error) {
	if slug == "" {
		return nil, &core.ValidationError{Field: "slug", Message: "document slug is required"}
	}

	const query = `
		query DocumentBySlug($slug: String!) {
			documents(first: 1, filter: { slugId: { eq: $slug } }, includeArchived: true) {
				nodes {` + documentFields + `}
			}
		}
	`

	var response struct {
		Documents struct {
			Nodes []core.Document `json:"nodes"`
		} `json:"documents"`
	}

	if err := dc.base.ExecuteRequest(query, map[string]interface{}{"slug": slug}, &response); err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	if len(response.Documents.Nodes) == 0 {
		return nil, &core.NotFoundError{ResourceType: "document", ResourceID: slug}
	}

	return &response.Documents.Nodes[0], nil
}

// buildCreateInput converts CreateInput into the GraphQL DocumentCreateInput map.
func buildCreateInput(input *CreateInput) map[string]interface{} {
	m := map[string]interface{}{
		"title": input.Title,
	}
	if input.Content != nil {
		m["content"] = *input.Content
	}
	if input.ProjectID != "" {
		m["projectId"] = input.ProjectID
	}
	if input.IssueID != "" {
		m["issueId"] = input.IssueID
	}
	if input.TeamID != "" {
		m["teamId"] = input.TeamID
	}
	if input.Icon != "" {
		m["icon"] = input.Icon
	}
	if input.Color != "" {
		m["color"] = input.Color
	}
	return m
}

// CreateDocument creates a new document.
func (dc *Client) CreateDocument(input *CreateInput) (*core.Document, error) {
	if input == nil || input.Title == "" {
		return nil, &core.ValidationError{Field: "title", Message: "document title is required"}
	}

	const mutation = `
		mutation DocumentCreate($input: DocumentCreateInput!) {
			documentCreate(input: $input) {
				success
				document {` + documentFields + `}
			}
		}
	`

	variables := map[string]interface{}{
		"input": buildCreateInput(input),
	}

	var response struct {
		DocumentCreate struct {
			Success  bool          `json:"success"`
			Document core.Document `json:"document"`
		} `json:"documentCreate"`
	}

	if err := dc.base.ExecuteRequest(mutation, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to create document: %w", err)
	}

	if !response.DocumentCreate.Success {
		return nil, fmt.Errorf("documentCreate returned success=false")
	}

	return &response.DocumentCreate.Document, nil
}

// buildUpdateInput converts UpdateInput into the GraphQL DocumentUpdateInput map.
// Only non-nil fields are included.
func buildUpdateInput(input *UpdateInput) map[string]interface{} {
	m := map[string]interface{}{}
	if input == nil {
		return m
	}
	set := func(key string, v *string) {
		if v != nil {
			m[key] = *v
		}
	}
	// Parent IDs: an explicit "" clears the parent (sent as null).
	setID := func(key string, v *string) {
		if v == nil {
			return
		}
		if *v == "" {
			m[key] = nil
			return
		}
		m[key] = *v
	}
	set("title", input.Title)
	set("content", input.Content)
	setID("projectId", input.ProjectID)
	setID("issueId", input.IssueID)
	setID("teamId", input.TeamID)
	set("icon", input.Icon)
	set("color", input.Color)
	return m
}

// UpdateDocument updates an existing document by UUID.
func (dc *Client) UpdateDocument(id string, input *UpdateInput) (*core.Document, error) {
	if id == "" {
		return nil, &core.ValidationError{Field: "id", Message: "document ID is required"}
	}

	inputMap := buildUpdateInput(input)
	if len(inputMap) == 0 {
		return nil, &core.ValidationError{Field: "input", Message: "no fields to update"}
	}

	const mutation = `
		mutation DocumentUpdate($id: String!, $input: DocumentUpdateInput!) {
			documentUpdate(id: $id, input: $input) {
				success
				document {` + documentFields + `}
			}
		}
	`

	variables := map[string]interface{}{
		"id":    id,
		"input": inputMap,
	}

	var response struct {
		DocumentUpdate struct {
			Success  bool          `json:"success"`
			Document core.Document `json:"document"`
		} `json:"documentUpdate"`
	}

	if err := dc.base.ExecuteRequest(mutation, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	if !response.DocumentUpdate.Success {
		return nil, fmt.Errorf("documentUpdate returned success=false")
	}

	return &response.DocumentUpdate.Document, nil
}

// DeleteDocument moves a document to trash by UUID. Linear archives rather than hard-deletes.
func (dc *Client) DeleteDocument(id string) error {
	if id == "" {
		return &core.ValidationError{Field: "id", Message: "document ID is required"}
	}

	const mutation = `
		mutation DocumentDelete($id: String!) {
			documentDelete(id: $id) {
				success
			}
		}
	`

	var response struct {
		DocumentDelete struct {
			Success bool `json:"success"`
		} `json:"documentDelete"`
	}

	if err := dc.base.ExecuteRequest(mutation, map[string]interface{}{"id": id}, &response); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	if !response.DocumentDelete.Success {
		return fmt.Errorf("documentDelete returned success=false")
	}

	return nil
}
