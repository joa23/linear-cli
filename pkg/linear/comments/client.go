package comments

import (
	"github.com/joa23/linear-cli/pkg/linear/core"
	"github.com/joa23/linear-cli/pkg/linear/guidance"
	"github.com/joa23/linear-cli/pkg/linear/validation"
	"fmt"
)

// CommentClient handles all comment and reaction operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and focuses on
// collaboration features like comments, replies, and reactions.
type Client struct {
	base *core.BaseClient
}

// NewCommentClient creates a new comment client with the provided base client
func NewClient(base *core.BaseClient) *Client {
	return &Client{base: base}
}

// CreateComment creates a new comment on an issue
// Why: Comments are essential for collaboration on issues. This method
// enables users to add context, updates, and discussions to issues.
func (cc *Client) CreateComment(issueID, body string) (*core.Comment, error) {
	// Validate required inputs
	// Why: Both issue ID and comment body are required. Empty comments
	// provide no value and would be rejected by the API.
	if issueID == "" {
		return nil, guidance.ValidationErrorWithExample("issueID", "cannot be empty",
			`// Get issue first, then add comment
issue = linear_get_issue("LIN-123")
linear_create_comment(issue.id, "This is my comment")`)
	}
	if body == "" {
		return nil, guidance.ValidationErrorWithExample("body", "cannot be empty",
			`linear_create_comment(issueId, "Progress update: Completed authentication module")`)
	}
	
	const mutation = `
		mutation CreateComment($issueId: String!, $body: String!) {
			commentCreate(
				input: {
					issueId: $issueId
					body: $body
				}
			) {
				success
				comment {
					id
					body
					createdAt
					updatedAt
					user {
						id
						name
						email
					}
					issue {
						id
						identifier
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"issueId": issueID,
		"body":    body,
	}
	
	var response struct {
		CommentCreate struct {
			Success bool    `json:"success"`
			Comment core.Comment `json:"comment"`
		} `json:"commentCreate"`
	}
	
	err := cc.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, guidance.EnhanceGenericError("create comment", err)
	}
	
	if !response.CommentCreate.Success {
		return nil, guidance.OperationFailedError("Create comment", "comment", []string{
			"Verify the issue ID exists using linear_get_issue",
			"Check if you have permission to comment on this issue",
			"Ensure the issue is not locked or archived",
		})
	}
	
	return &response.CommentCreate.Comment, nil
}

// GetIssueComments retrieves all comments for an issue
func (cc *Client) GetIssueComments(issueID string) ([]core.Comment, error) {
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}

	const query = `
		query GetIssueComments($issueId: String!) {
			issue(id: $issueId) {
				comments {
					nodes {
						id
						body
						createdAt
						updatedAt
						user {
							id
							name
							email
						}
						parent {
							id
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"issueId": issueID,
	}

	var response struct {
		Issue struct {
			Comments struct {
				Nodes []struct {
					ID        string `json:"id"`
					Body      string `json:"body"`
					CreatedAt string `json:"createdAt"`
					UpdatedAt string `json:"updatedAt"`
					User      struct {
						ID    string `json:"id"`
						Name  string `json:"name"`
						Email string `json:"email"`
					} `json:"user"`
					Parent *struct {
						ID string `json:"id"`
					} `json:"parent"`
				} `json:"nodes"`
			} `json:"comments"`
		} `json:"issue"`
	}

	err := cc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue comments: %w", err)
	}

	// Convert to Comment slice
	comments := make([]core.Comment, len(response.Issue.Comments.Nodes))
	for i, node := range response.Issue.Comments.Nodes {
		comments[i] = core.Comment{
			ID:        node.ID,
			Body:      node.Body,
			CreatedAt: node.CreatedAt,
			UpdatedAt: node.UpdatedAt,
			User: core.User{
				ID:    node.User.ID,
				Name:  node.User.Name,
				Email: node.User.Email,
			},
		}
		if node.Parent != nil {
			comments[i].Parent = &core.CommentParent{ID: node.Parent.ID}
		}
	}

	return comments, nil
}

// CreateCommentReply creates a reply to an existing comment
// Why: Threaded discussions allow for more organized conversations.
// This method enables users to reply directly to specific comments.
func (cc *Client) CreateCommentReply(issueID, parentID, body string) (*core.Comment, error) {
	// Validate all required inputs
	// Why: All three parameters are needed to properly create a reply.
	// The issueID ensures the reply is associated with the correct issue.
	if issueID == "" {
		return nil, &core.ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	if parentID == "" {
		return nil, &core.ValidationError{Field: "parentID", Message: "parentID cannot be empty"}
	}
	if body == "" {
		return nil, &core.ValidationError{Field: "body", Message: "body cannot be empty"}
	}
	
	const mutation = `
		mutation CreateCommentReply($issueId: String!, $parentId: String!, $body: String!) {
			commentCreate(
				input: {
					issueId: $issueId
					parentId: $parentId
					body: $body
				}
			) {
				success
				comment {
					id
					body
					createdAt
					updatedAt
					user {
						id
						name
						email
					}
					parent {
						id
					}
					issue {
						id
						identifier
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"issueId":  issueID,
		"parentId": parentID,
		"body":     body,
	}
	
	var response struct {
		CommentCreate struct {
			Success bool    `json:"success"`
			Comment core.Comment `json:"comment"`
		} `json:"commentCreate"`
	}
	
	err := cc.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment reply: %w", err)
	}
	
	if !response.CommentCreate.Success {
		return nil, fmt.Errorf("comment reply creation was not successful")
	}
	
	return &response.CommentCreate.Comment, nil
}

// GetCommentWithReplies retrieves a comment and all its replies
// Why: Understanding the full context of a discussion requires seeing
// all replies. This method provides the complete comment thread.
func (cc *Client) GetCommentWithReplies(commentID string) (*core.CommentWithReplies, error) {
	// Validate input
	// Why: Comment ID is required to identify which comment thread to retrieve.
	if commentID == "" {
		return nil, &core.ValidationError{Field: "commentID", Message: "commentID cannot be empty"}
	}
	
	const query = `
		query GetCommentWithReplies($id: String!) {
			comment(id: $id) {
				id
				body
				createdAt
				updatedAt
				user {
					id
					name
					email
				}
				issue {
					id
					identifier
					title
				}
				parent {
					id
				}
				children {
					nodes {
						id
						body
						createdAt
						updatedAt
						user {
							id
							name
							email
						}
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"id": commentID,
	}
	
	var response struct {
		Comment struct {
			ID        string `json:"id"`
			Body      string `json:"body"`
			CreatedAt string `json:"createdAt"`
			UpdatedAt string `json:"updatedAt"`
			User      struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"user"`
			Issue struct {
				ID         string `json:"id"`
				Identifier string `json:"identifier"`
				Title      string `json:"title"`
			} `json:"issue"`
			Parent *struct {
				ID string `json:"id"`
			} `json:"parent"`
			Children struct {
				Nodes []struct {
					ID        string `json:"id"`
					Body      string `json:"body"`
					CreatedAt string `json:"createdAt"`
					UpdatedAt string `json:"updatedAt"`
					User      struct {
						ID    string `json:"id"`
						Name  string `json:"name"`
						Email string `json:"email"`
					} `json:"user"`
				} `json:"nodes"`
			} `json:"children"`
		} `json:"comment"`
	}
	
	err := cc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment with replies: %w", err)
	}

	// Check if comment was found
	if response.Comment.ID == "" {
		return nil, &core.NotFoundError{
			ResourceType: "comment",
			ResourceID:   commentID,
		}
	}

	// Build the result structure
	// Why: We transform the GraphQL response into our domain model
	// for easier consumption by callers.
	result := &core.CommentWithReplies{
		Comment: core.Comment{
			ID:        response.Comment.ID,
			Body:      response.Comment.Body,
			CreatedAt: response.Comment.CreatedAt,
			UpdatedAt: response.Comment.UpdatedAt,
			User: core.User{
				ID:    response.Comment.User.ID,
				Name:  response.Comment.User.Name,
				Email: response.Comment.User.Email,
			},
			Issue: core.CommentIssue{
				ID:         response.Comment.Issue.ID,
				Identifier: response.Comment.Issue.Identifier,
				Title:      response.Comment.Issue.Title,
			},
		},
		Replies: make([]core.Comment, len(response.Comment.Children.Nodes)),
	}
	
	// Map replies
	// Why: Convert the nested reply structure into our Comment model
	// for consistent representation across the API.
	for i, reply := range response.Comment.Children.Nodes {
		result.Replies[i] = core.Comment{
			ID:        reply.ID,
			Body:      reply.Body,
			CreatedAt: reply.CreatedAt,
			UpdatedAt: reply.UpdatedAt,
			User: core.User{
				ID:    reply.User.ID,
				Name:  reply.User.Name,
				Email: reply.User.Email,
			},
			Parent: &core.CommentParent{ID: commentID},
			Issue: core.CommentIssue{
				ID:         response.Comment.Issue.ID,
				Identifier: response.Comment.Issue.Identifier,
				Title:      response.Comment.Issue.Title,
			},
		}
	}
	
	return result, nil
}

// AddReaction adds an emoji reaction to an issue or comment
// Why: Reactions provide quick, non-verbal feedback on issues and comments.
// They're useful for acknowledging, agreeing, or expressing sentiment.
func (cc *Client) AddReaction(targetID, emoji string) error {
	// Validate inputs
	// Why: Both the target (issue/comment) and emoji are required.
	// Empty values would cause the mutation to fail.
	if targetID == "" {
		return &core.ValidationError{Field: "targetID", Message: "targetID cannot be empty"}
	}
	if emoji == "" {
		return &core.ValidationError{Field: "emoji", Message: "emoji cannot be empty"}
	}
	if !validation.IsValidEmoji(emoji) {
		return &core.ValidationError{Field: "emoji", Value: emoji, Reason: "must be a single valid emoji"}
	}
	
	const mutation = `
		mutation AddReaction($id: String!, $emoji: String!) {
			reactionCreate(
				input: {
					id: $id
					emoji: $emoji
				}
			) {
				success
				reaction {
					id
					emoji
					user {
						id
						name
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"id":    targetID,
		"emoji": emoji,
	}
	
	var response struct {
		ReactionCreate struct {
			Success bool `json:"success"`
			Reaction struct {
				ID    string `json:"id"`
				Emoji string `json:"emoji"`
				User  struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"user"`
			} `json:"reaction"`
		} `json:"reactionCreate"`
	}
	
	err := cc.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	
	if !response.ReactionCreate.Success {
		return fmt.Errorf("reaction creation was not successful")
	}
	
	return nil
}