package service

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/linear"
	"github.com/joa23/linear-cli/internal/linear/attachments"
	"github.com/joa23/linear-cli/internal/linear/identifiers"
)

// AttachmentService handles attachment-related operations
type AttachmentService struct {
	client    *linear.Client
	formatter *format.Formatter
}

// AttachmentServiceInterface defines the operations for attachment management
type AttachmentServiceInterface interface {
	List(issueID string, verbosity format.Verbosity, outputType format.OutputType) (string, error)
	Create(input *AttachmentCreateParams) (string, error)
	Update(attachmentID string, input *AttachmentUpdateParams) (string, error)
	Delete(attachmentID string) error
}

// AttachmentCreateParams holds CLI-level parameters for creating an attachment
type AttachmentCreateParams struct {
	IssueID    string
	URL        string
	FilePath   string
	Title      string
	Subtitle   string
	Verbosity  format.Verbosity
	OutputType format.OutputType
}

// AttachmentUpdateParams holds CLI-level parameters for updating an attachment
type AttachmentUpdateParams struct {
	Title      string
	Subtitle   string
	Verbosity  format.Verbosity
	OutputType format.OutputType
}

// NewAttachmentService creates a new AttachmentService
func NewAttachmentService(client *linear.Client, formatter *format.Formatter) *AttachmentService {
	return &AttachmentService{
		client:    client,
		formatter: formatter,
	}
}

// List returns formatted attachment list for an issue
func (s *AttachmentService) List(issueID string, verbosity format.Verbosity, outputType format.OutputType) (string, error) {
	resolvedID, err := s.resolveIssueID(issueID)
	if err != nil {
		return "", err
	}

	atts, err := s.client.Attachments.ListAttachments(resolvedID)
	if err != nil {
		return "", fmt.Errorf("failed to list attachments: %w", err)
	}

	return s.formatter.RenderAttachmentList(atts, verbosity, outputType), nil
}

// Create creates a new attachment (URL or file upload)
func (s *AttachmentService) Create(params *AttachmentCreateParams) (string, error) {
	if params.URL == "" && params.FilePath == "" {
		return "", fmt.Errorf("either --url or --file is required")
	}
	if params.URL != "" && params.FilePath != "" {
		return "", fmt.Errorf("--url and --file are mutually exclusive")
	}
	if params.Title == "" {
		return "", fmt.Errorf("--title is required")
	}

	resolvedIssueID, err := s.resolveIssueID(params.IssueID)
	if err != nil {
		return "", err
	}

	attachURL := params.URL
	if params.FilePath != "" {
		assetURL, err := s.client.Attachments.UploadFileFromPath(params.FilePath)
		if err != nil {
			return "", fmt.Errorf("failed to upload file: %w", err)
		}
		attachURL = assetURL
	}

	input := &attachments.AttachmentCreateInput{
		IssueID:  resolvedIssueID,
		URL:      attachURL,
		Title:    params.Title,
		Subtitle: params.Subtitle,
	}

	att, err := s.client.Attachments.CreateAttachment(input)
	if err != nil {
		return "", fmt.Errorf("failed to create attachment: %w", err)
	}

	return s.formatter.RenderAttachment(att, params.Verbosity, params.OutputType), nil
}

// Update updates an existing attachment
func (s *AttachmentService) Update(attachmentID string, params *AttachmentUpdateParams) (string, error) {
	if params.Title == "" {
		return "", fmt.Errorf("--title is required")
	}

	input := &attachments.AttachmentUpdateInput{
		Title:    params.Title,
		Subtitle: params.Subtitle,
	}

	att, err := s.client.Attachments.UpdateAttachment(attachmentID, input)
	if err != nil {
		return "", fmt.Errorf("failed to update attachment: %w", err)
	}

	return s.formatter.RenderAttachment(att, params.Verbosity, params.OutputType), nil
}

// Delete deletes an attachment by UUID
func (s *AttachmentService) Delete(attachmentID string) error {
	return s.client.Attachments.DeleteAttachment(attachmentID)
}

// resolveIssueID resolves an issue identifier (e.g., "TEC-123") to UUID
func (s *AttachmentService) resolveIssueID(issueID string) (string, error) {
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
