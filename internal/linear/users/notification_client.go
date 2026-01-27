package users

import (
	"github.com/joa23/linear-cli/internal/linear/core"
	"fmt"
	"time"
)

// NotificationClient handles all notification-related operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and manages user
// notifications including mentions and updates.
type NotificationClient struct {
	base *core.BaseClient
}

// NewNotificationClient creates a new notification client with the provided base client
func NewNotificationClient(base *core.BaseClient) *NotificationClient {
	return &NotificationClient{base: base}
}

// GetNotifications retrieves notifications for the authenticated user
// Why: Notifications keep users informed about mentions, assignments, and
// updates. This method provides access to that activity stream.
func (nc *NotificationClient) GetNotifications(includeRead bool, limit int) ([]core.Notification, error) {
	// Validate limit
	// Why: Limits must be explicitly specified and within acceptable ranges.
	// Zero or negative limits don't make sense and excessive limits could
	// overload the API or cause performance issues.
	if limit <= 0 {
		return nil, &core.ValidationError{Field: "limit", Value: limit, Reason: "must be positive"}
	}
	if limit > 1000 {
		return nil, &core.ValidationError{Field: "limit", Value: limit, Reason: "cannot exceed 1000"}
	}

	const query = `
		query GetNotifications($includeArchived: Boolean!, $first: Int) {
			notifications(includeArchived: $includeArchived, first: $first) {
				nodes {
					id
					type
					createdAt
					readAt
					archivedAt
					snoozedUntilAt
					user {
						id
						name
						email
					}
					... on IssueNotification {
						issue {
							id
							identifier
							title
						}
					}
					... on ProjectNotification {
						project {
							id
							name
						}
					}
				}
			}
		}
	`
	
	variables := map[string]interface{}{
		"includeArchived": includeRead,
		"first":           limit,
	}
	
	var response struct {
		Notifications struct {
			Nodes []core.Notification `json:"nodes"`
		} `json:"notifications"`
	}
	
	err := nc.base.ExecuteRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	
	return response.Notifications.Nodes, nil
}

// MarkNotificationRead marks a notification as read
// Why: Users need to acknowledge notifications to keep their inbox
// manageable. This method provides that acknowledgment capability.
func (nc *NotificationClient) MarkNotificationRead(notificationID string) error {
	// Validate input
	// Why: Notification ID is required to identify which notification
	// to mark as read. Empty ID would cause the mutation to fail.
	if notificationID == "" {
		return &core.ValidationError{Field: "notificationID", Message: "notificationID cannot be empty"}
	}

	const mutation = `
		mutation MarkNotificationAsRead($id: String!, $input: NotificationUpdateInput!) {
			notificationUpdate(id: $id, input: $input) {
				success
				notification {
					id
					readAt
				}
			}
		}
	`

	variables := map[string]interface{}{
		"id": notificationID,
		"input": map[string]interface{}{
			"readAt": time.Now().Format(time.RFC3339),
		},
	}

	var response struct {
		NotificationUpdate struct {
			Success bool `json:"success"`
		} `json:"notificationUpdate"`
	}

	err := nc.base.ExecuteRequest(mutation, variables, &response)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}

	if !response.NotificationUpdate.Success {
		return fmt.Errorf("marking notification as read was not successful")
	}

	return nil
}