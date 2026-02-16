package cli

import (
	"fmt"

	"github.com/joa23/linear-cli/internal/format"
	"github.com/joa23/linear-cli/internal/service"
	"github.com/spf13/cobra"
)

func newAttachmentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attachments",
		Aliases: []string{"attachment", "att"},
		Short:   "Manage Linear attachment objects (sidebar cards)",
		Long: `Manage Linear attachment objects on issues.

Attachment objects are structured cards that appear in Linear's sidebar — GitHub PRs,
Slack threads, Figma designs, uploaded files, or any URL. They have titles, subtitles,
and source type metadata. Linear auto-detects source type from URLs.

NOTE: This is different from the --attach flag on issues create/update/comment/reply,
which embeds files as inline markdown images in the issue body. Use --attach for images
you want visible in the description or comment text. Use 'attachments create' for
tracked resources you want as sidebar cards.`,
	}

	cmd.AddCommand(
		newAttachmentsListCmd(),
		newAttachmentsCreateCmd(),
		newAttachmentsUpdateCmd(),
		newAttachmentsDeleteCmd(),
	)

	return cmd
}

func newAttachmentsListCmd() *cobra.Command {
	var formatStr, outputType string

	cmd := &cobra.Command{
		Use:   "list <issue-id>",
		Short: "List attachments on an issue",
		Long:  "List all Linear attachment objects on an issue. Shows structured cards (URLs, uploads), not inline markdown embeds.",
		Example: `  # List attachments
  linear attachments list TEC-123

  # JSON output for automation
  linear attachments list TEC-123 --output json

  # Full details (includes IDs for update/delete)
  linear attachments list TEC-123 --format full`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			verbosity, err := format.ParseVerbosity(formatStr)
			if err != nil {
				return err
			}
			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}

			result, err := deps.Attachments.List(issueID, verbosity, output)
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVarP(&formatStr, "format", "f", "compact", "Verbosity: minimal|compact|full")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newAttachmentsCreateCmd() *cobra.Command {
	var (
		url        string
		filePath   string
		title      string
		subtitle   string
		outputType string
	)

	cmd := &cobra.Command{
		Use:   "create <issue-id>",
		Short: "Create an attachment on an issue",
		Long: `Create a Linear attachment object (sidebar card) on an issue. Two modes:

  --url   Attach an external URL (GitHub PR, Slack thread, Figma design, etc.)
  --file  Upload a local file to Linear's CDN and create an attachment card for it

Linear auto-detects source type from URLs (github, slack, figma, etc.).
The URL is used as an idempotent key — creating with the same URL on the same
issue updates the existing attachment rather than duplicating it.

NOTE: To embed a file as an inline image in the issue description or a comment,
use 'issues create --attach' or 'issues comment --attach' instead.`,
		Example: `  # Attach a URL (sidebar card with auto-detected source type)
  linear attachments create TEC-123 --url "https://github.com/org/repo/pull/42" --title "PR #42"

  # Upload a file (title defaults to filename: "screenshot.png")
  linear attachments create TEC-123 --file /tmp/screenshot.png

  # Upload a file with custom title
  linear attachments create TEC-123 --file /tmp/screenshot.png --title "Bug screenshot"

  # With subtitle
  linear attachments create TEC-123 --url "https://figma.com/..." --title "Mockup v2" --subtitle "Login redesign"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID := args[0]
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}

			params := &service.AttachmentCreateParams{
				IssueID:    issueID,
				URL:        url,
				FilePath:   filePath,
				Title:      title,
				Subtitle:   subtitle,
				Verbosity:  format.VerbosityCompact,
				OutputType: output,
			}

			result, err := deps.Attachments.Create(params)
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "URL to attach (mutually exclusive with --file)")
	cmd.Flags().StringVar(&filePath, "file", "", "Local file to upload and attach (mutually exclusive with --url)")
	cmd.Flags().StringVar(&title, "title", "", "Attachment title (required for --url; defaults to filename for --file)")
	cmd.Flags().StringVar(&subtitle, "subtitle", "", "Attachment subtitle")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")

	return cmd
}

func newAttachmentsUpdateCmd() *cobra.Command {
	var (
		title      string
		subtitle   string
		outputType string
	)

	cmd := &cobra.Command{
		Use:   "update <attachment-id>",
		Short: "Update an attachment",
		Long:  "Update an existing attachment's title and subtitle. Get attachment IDs from 'attachments list --format full'.",
		Example: `  # Update title
  linear attachments update <uuid> --title "Updated PR title"

  # Update title and subtitle
  linear attachments update <uuid> --title "New title" --subtitle "New subtitle"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attachmentID := args[0]
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			output, err := format.ParseOutputType(outputType)
			if err != nil {
				return err
			}

			params := &service.AttachmentUpdateParams{
				Title:      title,
				Subtitle:   subtitle,
				Verbosity:  format.VerbosityCompact,
				OutputType: output,
			}

			result, err := deps.Attachments.Update(attachmentID, params)
			if err != nil {
				return err
			}

			fmt.Println(result)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "New title (required)")
	cmd.Flags().StringVar(&subtitle, "subtitle", "", "New subtitle")
	cmd.Flags().StringVarP(&outputType, "output", "o", "text", "Output: text|json")
	cmd.MarkFlagRequired("title")

	return cmd
}

func newAttachmentsDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <attachment-id>",
		Short: "Delete an attachment",
		Long:  "Delete a Linear attachment object. Get attachment IDs from 'attachments list --format full'.",
		Example: `  # Delete an attachment
  linear attachments delete <uuid>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			attachmentID := args[0]
			deps, err := getDeps(cmd)
			if err != nil {
				return err
			}

			err = deps.Attachments.Delete(attachmentID)
			if err != nil {
				return err
			}

			fmt.Println("Attachment deleted.")
			return nil
		},
	}

	return cmd
}
