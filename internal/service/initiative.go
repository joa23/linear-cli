package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/pkg/linear"
	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/identifiers"
	"github.com/joa23/linear-cli/pkg/linear/initiatives"
)

// InitiativeService handles initiative-related operations
type InitiativeService struct {
	client    *linear.Client
	formatter *format.Formatter
}

// InitiativeServiceInterface defines the operations for initiative management
type InitiativeServiceInterface interface {
	List(params *InitiativeListParams) (string, error)
	Get(ref string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Create(params *InitiativeCreateParams) (string, error)
	Update(ref string, params *InitiativeUpdateParams) (string, error)
	Archive(ref string) (string, error)
	Unarchive(ref string) (string, error)
	Delete(ref string) (string, error)
	AddProject(initiativeRef, projectRef string) (string, error)
	RemoveProject(initiativeRef, projectRef string) (string, error)
	ListUpdates(ref string, limit int, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	CreateUpdate(params *InitiativeStatusUpdateParams) (string, error)
}

// InitiativeListParams holds CLI-level parameters for listing initiatives.
// Owner is a human identifier ("me", name, or email) and is resolved here.
type InitiativeListParams struct {
	Status          string
	Owner           string
	Query           string
	IncludeArchived bool
	Limit           int
	Verbosity       format.Verbosity
	OutputType      format.OutputType
}

// InitiativeCreateParams holds CLI-level parameters for creating an initiative.
type InitiativeCreateParams struct {
	Name        string
	Description string
	Content     string
	ContentSet  bool // true when the caller supplied content (so "" is sent explicitly)
	Owner       string
	Status      string
	TargetDate  string
	Icon        string
	Color       string
	Projects    []string // Project names or UUIDs to link after creation
	Verbosity   format.Verbosity
	OutputType  format.OutputType
}

// InitiativeUpdateParams holds CLI-level parameters for updating an initiative. nil = unchanged.
// Owner and TargetDate set to "" clear the value.
type InitiativeUpdateParams struct {
	Name        *string
	Description *string
	Content     *string
	Owner       *string
	Status      *string
	TargetDate  *string
	Icon        *string
	Color       *string
	Verbosity   format.Verbosity
	OutputType  format.OutputType
}

// InitiativeStatusUpdateParams holds CLI-level parameters for posting a status update.
type InitiativeStatusUpdateParams struct {
	Initiative string
	Body       string
	Health     string
	Verbosity  format.Verbosity
	OutputType format.OutputType
}

// NewInitiativeService creates a new InitiativeService
func NewInitiativeService(client *linear.Client, formatter *format.Formatter) *InitiativeService {
	return &InitiativeService{
		client:    client,
		formatter: formatter,
	}
}

// List returns a formatted list of initiatives matching the params
func (s *InitiativeService) List(params *InitiativeListParams) (string, error) {
	status, err := normalizeInitiativeStatus(params.Status)
	if err != nil {
		return "", err
	}

	filter := &initiatives.ListFilter{
		Status:          status,
		Query:           params.Query,
		IncludeArchived: params.IncludeArchived,
	}
	if params.Owner != "" {
		owner, err := s.client.ResolveUserIdentifier(params.Owner)
		if err != nil {
			return "", fmt.Errorf("failed to resolve owner '%s': %w", params.Owner, err)
		}
		filter.OwnerID = owner.ID
	}

	result, err := s.client.Initiatives.ListInitiatives(filter, params.Limit)
	if err != nil {
		return "", err
	}

	page := &format.Pagination{HasNextPage: result.HasNextPage, EndCursor: result.EndCursor}
	return s.formatter.RenderInitiativeList(result.Initiatives, params.Verbosity, params.OutputType, page), nil
}

// Get returns a formatted initiative. ref may be a UUID, slug, URL, or exact name.
func (s *InitiativeService) Get(ref string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	ini, err := fetchInitiative(s.client.Initiatives, ref)
	if err != nil {
		return "", err
	}
	return s.formatter.RenderInitiative(ini, verbosity, outputType), nil
}

// Create creates a new initiative and links any requested projects
func (s *InitiativeService) Create(params *InitiativeCreateParams) (string, error) {
	if strings.TrimSpace(params.Name) == "" {
		return "", fmt.Errorf("initiative name is required")
	}
	status, err := normalizeInitiativeStatus(params.Status)
	if err != nil {
		return "", err
	}
	if err := validateTargetDate(params.TargetDate); err != nil {
		return "", err
	}

	// Resolve everything before creating so a typo doesn't leave a half-configured initiative.
	projectIDs, err := s.resolveProjects(params.Projects)
	if err != nil {
		return "", err
	}

	input := &initiatives.CreateInput{
		Name:        params.Name,
		Description: params.Description,
		Status:      status,
		TargetDate:  params.TargetDate,
		Icon:        params.Icon,
		Color:       params.Color,
	}
	if params.ContentSet {
		content := params.Content
		input.Content = &content
	}
	if params.Owner != "" {
		owner, err := s.client.ResolveUserIdentifier(params.Owner)
		if err != nil {
			return "", fmt.Errorf("failed to resolve owner '%s': %w", params.Owner, err)
		}
		input.OwnerID = owner.ID
	}

	ini, err := s.client.Initiatives.CreateInitiative(input)
	if err != nil {
		return "", err
	}

	if len(projectIDs) > 0 {
		for _, pid := range projectIDs {
			if _, err := s.client.Initiatives.AddProject(ini.ID, pid); err != nil {
				return "", fmt.Errorf("initiative %q created (%s) but linking project %s failed: %w", ini.Name, ini.ID, pid, err)
			}
		}
		// Re-fetch so the rendered output shows the linked projects
		refreshed, err := s.client.Initiatives.GetInitiative(ini.ID)
		if err == nil {
			ini = refreshed
		}
	}

	return s.formatter.RenderInitiative(ini, params.Verbosity, params.OutputType), nil
}

// Update updates an existing initiative. ref may be a UUID, slug, URL, or exact name.
func (s *InitiativeService) Update(ref string, params *InitiativeUpdateParams) (string, error) {
	if params.Name == nil && params.Description == nil && params.Content == nil && params.Owner == nil &&
		params.Status == nil && params.TargetDate == nil && params.Icon == nil && params.Color == nil {
		return "", fmt.Errorf("nothing to update: provide at least one of --name, --description, --content, --content-file, --owner, --status, --target-date, --icon, --color")
	}
	if params.Name != nil && strings.TrimSpace(*params.Name) == "" {
		return "", fmt.Errorf("--name cannot be empty")
	}

	input := &initiatives.UpdateInput{
		Name:        params.Name,
		Description: params.Description,
		Content:     params.Content,
		Icon:        params.Icon,
		Color:       params.Color,
		TargetDate:  params.TargetDate,
	}
	if params.Status != nil {
		status, err := normalizeInitiativeStatus(*params.Status)
		if err != nil {
			return "", err
		}
		if status == "" {
			return "", fmt.Errorf("--status cannot be empty")
		}
		input.Status = &status
	}
	if params.TargetDate != nil {
		if err := validateTargetDate(*params.TargetDate); err != nil {
			return "", err
		}
	}

	id, err := resolveInitiativeID(s.client.Initiatives, ref)
	if err != nil {
		return "", err
	}

	if params.Owner != nil {
		ownerID := ""
		if *params.Owner != "" {
			owner, err := s.client.ResolveUserIdentifier(*params.Owner)
			if err != nil {
				return "", fmt.Errorf("failed to resolve owner '%s': %w", *params.Owner, err)
			}
			ownerID = owner.ID
		}
		input.OwnerID = &ownerID
	}

	ini, err := s.client.Initiatives.UpdateInitiative(id, input)
	if err != nil {
		return "", err
	}

	return s.formatter.RenderInitiative(ini, params.Verbosity, params.OutputType), nil
}

// Archive archives an initiative and returns a confirmation message
func (s *InitiativeService) Archive(ref string) (string, error) {
	ini, err := fetchInitiativeSummary(s.client.Initiatives, ref)
	if err != nil {
		return "", err
	}
	if err := s.client.Initiatives.ArchiveInitiative(ini.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Archived initiative %q (%s). Restore with 'linear initiatives unarchive %s'.", ini.Name, ini.ID, ini.ID), nil
}

// Unarchive restores an archived initiative and returns a confirmation message
func (s *InitiativeService) Unarchive(ref string) (string, error) {
	ini, err := fetchInitiativeSummary(s.client.Initiatives, ref)
	if err != nil {
		return "", err
	}
	if err := s.client.Initiatives.UnarchiveInitiative(ini.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Unarchived initiative %q (%s).", ini.Name, ini.ID), nil
}

// Delete moves an initiative to trash and returns a confirmation message
func (s *InitiativeService) Delete(ref string) (string, error) {
	ini, err := fetchInitiativeSummary(s.client.Initiatives, ref)
	if err != nil {
		return "", err
	}
	if err := s.client.Initiatives.DeleteInitiative(ini.ID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Deleted initiative %q (%s). Linear moves it to trash; restore from the Linear UI. Linked projects are not deleted.", ini.Name, ini.ID), nil
}

// AddProject links a project to an initiative
func (s *InitiativeService) AddProject(initiativeRef, projectRef string) (string, error) {
	ini, err := fetchInitiativeSummary(s.client.Initiatives, initiativeRef)
	if err != nil {
		return "", err
	}
	projectID, err := s.resolveProject(projectRef)
	if err != nil {
		return "", err
	}

	link, err := s.client.Initiatives.AddProject(ini.ID, projectID)
	if err != nil {
		return "", err
	}

	projectName := projectRef
	if link.Project != nil && link.Project.Name != "" {
		projectName = link.Project.Name
	}
	return fmt.Sprintf("Added project %q to initiative %q.", projectName, ini.Name), nil
}

// RemoveProject unlinks a project from an initiative (the project itself is kept)
func (s *InitiativeService) RemoveProject(initiativeRef, projectRef string) (string, error) {
	ini, err := fetchInitiativeSummary(s.client.Initiatives, initiativeRef)
	if err != nil {
		return "", err
	}
	projectID, err := s.resolveProject(projectRef)
	if err != nil {
		return "", err
	}

	linkID, err := s.client.Initiatives.FindProjectLink(ini.ID, projectID)
	if err != nil {
		if core.IsNotFoundError(err) {
			return "", fmt.Errorf("project '%s' is not linked to initiative %q", projectRef, ini.Name)
		}
		return "", err
	}

	if err := s.client.Initiatives.RemoveProjectLink(linkID); err != nil {
		return "", err
	}
	return fmt.Sprintf("Removed project '%s' from initiative %q.", projectRef, ini.Name), nil
}

// ListUpdates returns formatted status updates for an initiative, newest first
func (s *InitiativeService) ListUpdates(ref string, limit int, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	id, err := resolveInitiativeID(s.client.Initiatives, ref)
	if err != nil {
		return "", err
	}
	updates, err := s.client.Initiatives.ListUpdates(id, limit)
	if err != nil {
		return "", err
	}
	return s.formatter.RenderInitiativeUpdateList(updates, verbosity, outputType), nil
}

// CreateUpdate posts a status update (body + optional health) on an initiative
func (s *InitiativeService) CreateUpdate(params *InitiativeStatusUpdateParams) (string, error) {
	if strings.TrimSpace(params.Body) == "" {
		return "", fmt.Errorf("update body is required: use --body, --body - (stdin), or --body-file")
	}
	health, err := normalizeHealth(params.Health)
	if err != nil {
		return "", err
	}

	id, err := resolveInitiativeID(s.client.Initiatives, params.Initiative)
	if err != nil {
		return "", err
	}

	update, err := s.client.Initiatives.CreateUpdate(&initiatives.StatusUpdateInput{
		InitiativeID: id,
		Body:         params.Body,
		Health:       health,
	})
	if err != nil {
		return "", err
	}
	return s.formatter.RenderInitiativeUpdate(update, params.Verbosity, params.OutputType), nil
}

// resolveProjects resolves project names/UUIDs, deduplicating while keeping order.
func (s *InitiativeService) resolveProjects(refs []string) ([]string, error) {
	seen := map[string]bool{}
	var ids []string
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		id, err := s.resolveProject(ref)
		if err != nil {
			return nil, err
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// resolveProject resolves a project name or UUID to a UUID
func (s *InitiativeService) resolveProject(ref string) (string, error) {
	id, err := s.client.ResolveProjectIdentifier(ref, "")
	if err != nil {
		return "", fmt.Errorf("failed to resolve project '%s': %w", ref, err)
	}
	return id, nil
}

// --- Reference resolution (shared with ProjectService) ---

// initiativeSlugPattern matches a Linear slugId (12 lowercase hex chars), optionally
// preceded by a "<title-slug>-" prefix as it appears in initiative URLs.
var initiativeSlugPattern = regexp.MustCompile(`(?:^|-)([0-9a-f]{12})$`)

// initiativeRefKind classifies an initiative reference.
type initiativeRefKind int

const (
	initiativeRefInvalid initiativeRefKind = iota
	initiativeRefUUID
	initiativeRefSlug // URL or slug-looking token; falls back to name lookup if not found
	initiativeRefName
)

// parseInitiativeRef classifies ref and returns the normalized value to look up:
// the UUID, the slugId, or the trimmed name.
// URLs must be Linear initiative URLs (…/initiative/<title>-<slugId>[/…]).
func parseInitiativeRef(ref string) (initiativeRefKind, string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return initiativeRefInvalid, ""
	}
	if identifiers.IsUUID(ref) {
		return initiativeRefUUID, ref
	}

	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		path := ref
		if i := strings.IndexAny(path, "?#"); i >= 0 {
			path = path[:i]
		}
		segments := strings.Split(strings.TrimRight(path, "/"), "/")
		for i, seg := range segments {
			if seg == "initiative" && i+1 < len(segments) {
				if m := initiativeSlugPattern.FindStringSubmatch(segments[i+1]); m != nil {
					return initiativeRefSlug, m[1]
				}
			}
		}
		return initiativeRefInvalid, ""
	}

	if m := initiativeSlugPattern.FindStringSubmatch(ref); m != nil {
		return initiativeRefSlug, m[1]
	}
	return initiativeRefName, ref
}

// fetchInitiative loads a full initiative by UUID, slug, URL, or exact name (case-insensitive).
func fetchInitiative(ic *initiatives.Client, ref string) (*core.Initiative, error) {
	kind, value := parseInitiativeRef(ref)
	switch kind {
	case initiativeRefUUID:
		return ic.GetInitiative(value)
	case initiativeRefSlug:
		ini, err := ic.GetInitiativeBySlug(value)
		if err == nil {
			return ini, nil
		}
		if !core.IsNotFoundError(err) || strings.HasPrefix(strings.TrimSpace(ref), "http") {
			return nil, fmt.Errorf("initiative not found: %s", ref)
		}
		// A name that happens to look like a slug: fall through to name lookup.
	case initiativeRefInvalid:
		return nil, fmt.Errorf("invalid initiative reference: %q (expected UUID, slug, Linear initiative URL, or name)", ref)
	}

	match, err := findInitiativeByName(ic, strings.TrimSpace(ref))
	if err != nil {
		return nil, err
	}
	return ic.GetInitiative(match.ID)
}

// fetchInitiativeSummary resolves ref to an initiative with at least ID and Name populated,
// avoiding a full fetch when the name lookup already returns them.
func fetchInitiativeSummary(ic *initiatives.Client, ref string) (*core.Initiative, error) {
	if kind, _ := parseInitiativeRef(ref); kind == initiativeRefName {
		return findInitiativeByName(ic, strings.TrimSpace(ref))
	}
	return fetchInitiative(ic, ref)
}

// resolveInitiativeID converts a UUID, slug, URL, or name into an initiative UUID.
func resolveInitiativeID(ic *initiatives.Client, ref string) (string, error) {
	if kind, value := parseInitiativeRef(ref); kind == initiativeRefUUID {
		return value, nil
	}
	ini, err := fetchInitiativeSummary(ic, ref)
	if err != nil {
		return "", err
	}
	return ini.ID, nil
}

// findInitiativeByName returns the single initiative whose name equals name (case-insensitive).
func findInitiativeByName(ic *initiatives.Client, name string) (*core.Initiative, error) {
	matches, err := ic.FindInitiativesByName(name)
	if err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("initiative not found: %q (use 'linear initiatives list --query %q' to search)", name, name)
	case 1:
		return &matches[0], nil
	default:
		var options []string
		for _, m := range matches {
			options = append(options, fmt.Sprintf("%s (%s)", m.Name, m.SlugID))
		}
		return nil, fmt.Errorf("multiple initiatives named %q: %s; use the slug or UUID", name, strings.Join(options, ", "))
	}
}

// --- Validation helpers ---

// initiativeStatuses maps lowercase input to the InitiativeStatus enum value.
// Proposed and Canceled are newer statuses present in Linear's current public schema.
var initiativeStatuses = map[string]string{
	"planned":   "Planned",
	"active":    "Active",
	"completed": "Completed",
	"proposed":  "Proposed",
	"canceled":  "Canceled",
	"cancelled": "Canceled",
}

// normalizeInitiativeStatus maps case-insensitive input to the enum value. "" stays "".
func normalizeInitiativeStatus(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if v, ok := initiativeStatuses[strings.ToLower(s)]; ok {
		return v, nil
	}
	return "", fmt.Errorf("invalid status %q: expected Planned, Active, or Completed (newer workspaces also accept Proposed, Canceled)", s)
}

// normalizeHealth maps input like "onTrack", "on-track", "ON_TRACK" to the health enum. "" stays "".
func normalizeHealth(s string) (string, error) {
	key := strings.ToLower(strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.TrimSpace(s)))
	switch key {
	case "":
		return "", nil
	case "ontrack":
		return "onTrack", nil
	case "atrisk":
		return "atRisk", nil
	case "offtrack":
		return "offTrack", nil
	}
	return "", fmt.Errorf("invalid health %q: expected onTrack, atRisk, or offTrack", s)
}

// validateTargetDate accepts "" or a YYYY-MM-DD date.
func validateTargetDate(s string) error {
	if s == "" {
		return nil
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("invalid target date %q: expected YYYY-MM-DD", s)
	}
	return nil
}
