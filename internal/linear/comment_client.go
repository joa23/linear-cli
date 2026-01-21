package linear

import (
	"fmt"
)

// CommentClient handles all comment and reaction operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and focuses on
// collaboration features like comments, replies, and reactions.
type CommentClient struct {
	base *BaseClient
}

// NewCommentClient creates a new comment client with the provided base client
func NewCommentClient(base *BaseClient) *CommentClient {
	return &CommentClient{base: base}
}

// CreateComment creates a new comment on an issue
// Why: Comments are essential for collaboration on issues. This method
// enables users to add context, updates, and discussions to issues.
func (cc *CommentClient) CreateComment(issueID, body string) (*Comment, error) {
	// Validate required inputs
	// Why: Both issue ID and comment body are required. Empty comments
	// provide no value and would be rejected by the API.
	if issueID == "" {
		return nil, ValidationErrorWithExample("issueID", "cannot be empty",
			`// Get issue first, then add comment
issue = linear_get_issue("LIN-123")
linear_create_comment(issue.id, "This is my comment")`)
	}
	if body == "" {
		return nil, ValidationErrorWithExample("body", "cannot be empty",
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
			Comment Comment `json:"comment"`
		} `json:"commentCreate"`
	}
	
	err := cc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		return nil, EnhanceGenericError("create comment", err)
	}
	
	if !response.CommentCreate.Success {
		return nil, OperationFailedError("Create comment", "comment", []string{
			"Verify the issue ID exists using linear_get_issue",
			"Check if you have permission to comment on this issue",
			"Ensure the issue is not locked or archived",
		})
	}
	
	return &response.CommentCreate.Comment, nil
}

// GetIssueComments retrieves all comments for an issue
func (cc *CommentClient) GetIssueComments(issueID string) ([]Comment, error) {
	if issueID == "" {
		return nil, &ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
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

	err := cc.base.executeRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get issue comments: %w", err)
	}

	// Convert to Comment slice
	comments := make([]Comment, len(response.Issue.Comments.Nodes))
	for i, node := range response.Issue.Comments.Nodes {
		comments[i] = Comment{
			ID:        node.ID,
			Body:      node.Body,
			CreatedAt: node.CreatedAt,
			UpdatedAt: node.UpdatedAt,
			User: User{
				ID:    node.User.ID,
				Name:  node.User.Name,
				Email: node.User.Email,
			},
		}
		if node.Parent != nil {
			comments[i].Parent = &CommentParent{ID: node.Parent.ID}
		}
	}

	return comments, nil
}

// CreateCommentReply creates a reply to an existing comment
// Why: Threaded discussions allow for more organized conversations.
// This method enables users to reply directly to specific comments.
func (cc *CommentClient) CreateCommentReply(issueID, parentID, body string) (*Comment, error) {
	// Validate all required inputs
	// Why: All three parameters are needed to properly create a reply.
	// The issueID ensures the reply is associated with the correct issue.
	if issueID == "" {
		return nil, &ValidationError{Field: "issueID", Message: "issueID cannot be empty"}
	}
	if parentID == "" {
		return nil, &ValidationError{Field: "parentID", Message: "parentID cannot be empty"}
	}
	if body == "" {
		return nil, &ValidationError{Field: "body", Message: "body cannot be empty"}
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
			Comment Comment `json:"comment"`
		} `json:"commentCreate"`
	}
	
	err := cc.base.executeRequest(mutation, variables, &response)
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
func (cc *CommentClient) GetCommentWithReplies(commentID string) (*CommentWithReplies, error) {
	// Validate input
	// Why: Comment ID is required to identify which comment thread to retrieve.
	if commentID == "" {
		return nil, &ValidationError{Field: "commentID", Message: "commentID cannot be empty"}
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
	
	err := cc.base.executeRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment with replies: %w", err)
	}

	// Check if comment was found
	if response.Comment.ID == "" {
		return nil, &NotFoundError{
			ResourceType: "comment",
			ResourceID:   commentID,
		}
	}

	// Build the result structure
	// Why: We transform the GraphQL response into our domain model
	// for easier consumption by callers.
	result := &CommentWithReplies{
		Comment: Comment{
			ID:        response.Comment.ID,
			Body:      response.Comment.Body,
			CreatedAt: response.Comment.CreatedAt,
			UpdatedAt: response.Comment.UpdatedAt,
			User: User{
				ID:    response.Comment.User.ID,
				Name:  response.Comment.User.Name,
				Email: response.Comment.User.Email,
			},
			Issue: CommentIssue{
				ID:         response.Comment.Issue.ID,
				Identifier: response.Comment.Issue.Identifier,
				Title:      response.Comment.Issue.Title,
			},
		},
		Replies: make([]Comment, len(response.Comment.Children.Nodes)),
	}
	
	// Map replies
	// Why: Convert the nested reply structure into our Comment model
	// for consistent representation across the API.
	for i, reply := range response.Comment.Children.Nodes {
		result.Replies[i] = Comment{
			ID:        reply.ID,
			Body:      reply.Body,
			CreatedAt: reply.CreatedAt,
			UpdatedAt: reply.UpdatedAt,
			User: User{
				ID:    reply.User.ID,
				Name:  reply.User.Name,
				Email: reply.User.Email,
			},
			Parent: &CommentParent{ID: commentID},
			Issue: CommentIssue{
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
func (cc *CommentClient) AddReaction(targetID, emoji string) error {
	// Validate inputs
	// Why: Both the target (issue/comment) and emoji are required.
	// Empty values would cause the mutation to fail.
	if targetID == "" {
		return &ValidationError{Field: "targetID", Message: "targetID cannot be empty"}
	}
	if emoji == "" {
		return &ValidationError{Field: "emoji", Message: "emoji cannot be empty"}
	}
	if !isValidEmoji(emoji) {
		return &ValidationError{Field: "emoji", Value: emoji, Reason: "must be a single valid emoji"}
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
	
	err := cc.base.executeRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	
	if !response.ReactionCreate.Success {
		return fmt.Errorf("reaction creation was not successful")
	}
	
	return nil
}