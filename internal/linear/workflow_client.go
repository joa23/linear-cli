package linear

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// workflowStateCache represents cached workflow states for a team
type workflowStateCache struct {
	states     []WorkflowState
	cachedAt   time.Time
}

// WorkflowClient handles all workflow-related operations for the Linear API.
// It uses the shared BaseClient for HTTP communication and manages workflow
// states and transitions.
type WorkflowClient struct {
	base *BaseClient
	
	// Cache for workflow states per team
	cache    map[string]*workflowStateCache
	cacheMux sync.RWMutex
	cacheTTL time.Duration
}

// NewWorkflowClient creates a new workflow client with the provided base client
func NewWorkflowClient(base *BaseClient) *WorkflowClient {
	return &WorkflowClient{
		base:     base,
		cache:    make(map[string]*workflowStateCache),
		cacheTTL: 5 * time.Minute, // Default cache TTL of 5 minutes
	}
}

// GetWorkflowStates retrieves all available workflow states with caching
// Why: Understanding available workflow states is crucial for proper issue
// state management. This method provides all possible states an issue can
// transition to, optionally filtered by team. Results are cached per team
// to reduce API calls.
func (wc *WorkflowClient) GetWorkflowStates(teamID string) ([]WorkflowState, error) {
	// Use teamID as cache key, or "global" for no team filter
	cacheKey := teamID
	if cacheKey == "" {
		cacheKey = "global"
	}
	
	// Check cache first
	wc.cacheMux.RLock()
	cached, exists := wc.cache[cacheKey]
	if exists && time.Since(cached.cachedAt) < wc.cacheTTL {
		states := make([]WorkflowState, len(cached.states))
		copy(states, cached.states)
		wc.cacheMux.RUnlock()
		return states, nil
	}
	wc.cacheMux.RUnlock()
	
	// Cache miss or expired, fetch from API
	const query = `
		query GetWorkflowStates($filter: WorkflowStateFilter) {
			workflowStates(filter: $filter) {
				nodes {
					id
					name
					type
					color
					position
					description
					team {
						id
						name
					}
				}
			}
		}
	`
	
	// Build filter if team ID is provided
	// Why: Teams can have custom workflow states. Filtering by team
	// ensures we only get relevant states for that team's workflow.
	var variables map[string]interface{}
	if teamID != "" {
		variables = map[string]interface{}{
			"filter": map[string]interface{}{
				"team": map[string]interface{}{
					"id": map[string]interface{}{
						"eq": teamID,
					},
				},
			},
		}
	}
	
	var response struct {
		WorkflowStates struct {
			Nodes []WorkflowState `json:"nodes"`
		} `json:"workflowStates"`
	}
	
	err := wc.base.executeRequest(query, variables, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow states: %w", err)
	}
	
	// Update cache
	wc.cacheMux.Lock()
	wc.cache[cacheKey] = &workflowStateCache{
		states:   response.WorkflowStates.Nodes,
		cachedAt: time.Now(),
	}
	wc.cacheMux.Unlock()
	
	return response.WorkflowStates.Nodes, nil
}

// GetWorkflowStateByName retrieves a specific workflow state by name for a team
// Why: When updating issue states, users often know the state name (e.g., "In Progress")
// but need the state ID. This helper provides a convenient way to look up states by name.
// The search is case-insensitive to improve usability.
func (wc *WorkflowClient) GetWorkflowStateByName(teamID, stateName string) (*WorkflowState, error) {
	// Get all workflow states for the team (will use cache if available)
	states, err := wc.GetWorkflowStates(teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow states: %w", err)
	}
	
	// Search for the state by name (case-insensitive)
	lowerStateName := strings.ToLower(stateName)
	for _, state := range states {
		if strings.ToLower(state.Name) == lowerStateName {
			// Return a copy to prevent cache modification
			stateCopy := state
			return &stateCopy, nil
		}
	}
	
	// State not found
	return nil, nil
}