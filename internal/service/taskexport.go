package service

import (
	"fmt"
	"strings"

	"github.com/joa23/linear-cli/internal/linear/core"
	"github.com/joa23/linear-cli/internal/taskwriter"
)

// TaskExportService handles exporting Linear issues to Claude Code task format
type TaskExportService struct {
	issueClient IssueClientOperations
}

// NewTaskExportService creates a new TaskExportService
func NewTaskExportService(issueClient IssueClientOperations) *TaskExportService {
	return &TaskExportService{
		issueClient: issueClient,
	}
}

// ExportResult contains the result of an export operation
type ExportResult struct {
	Tasks              []taskwriter.ClaudeTask
	TotalTasks         int
	DependencyCount    int
	CircularDepsFound  []string
}

// Export exports a Linear issue and its complete dependency tree to Claude Code task format
// Returns an error if circular dependencies are detected
func (s *TaskExportService) Export(identifier string, outputFolder string, dryRun bool) (*ExportResult, error) {
	// Fetch complete tree with cycle detection
	issues, cycles, err := s.fetchCompleteTree(identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dependency tree: %w", err)
	}

	// Error on circular dependencies
	if len(cycles) > 0 {
		cyclePaths := make([]string, len(cycles))
		for i, cycle := range cycles {
			cyclePaths[i] = strings.Join(cycle, " → ")
		}
		return nil, fmt.Errorf("circular dependency detected:\n  %s\n\nCannot export. Please resolve circular dependencies in Linear first",
			strings.Join(cyclePaths, "\n  "))
	}

	// Convert to Claude tasks
	tasks := s.convertToTasks(issues)

	// Compute dependency count
	depCount := 0
	for _, task := range tasks {
		depCount += len(task.BlockedBy)
	}

	result := &ExportResult{
		Tasks:           tasks,
		TotalTasks:      len(tasks),
		DependencyCount: depCount,
	}

	// Write to folder if not dry run
	if !dryRun {
		writer := taskwriter.NewWriter()
		if err := writer.WriteTasks(outputFolder, tasks); err != nil {
			return nil, fmt.Errorf("failed to write tasks: %w", err)
		}
	}

	return result, nil
}

// issueNode represents an issue in the dependency graph
type issueNode struct {
	issue       *core.Issue
	children    []string // child issue identifiers
	dependencies []string // blocking dependencies (what this is blocked by)
}

// fetchCompleteTree fetches the complete dependency tree starting from the root issue
// Returns all issues, detected cycles, and any error
func (s *TaskExportService) fetchCompleteTree(rootID string) (map[string]*issueNode, [][]string, error) {
	visited := make(map[string]*issueNode)
	stack := make(map[string]bool) // For cycle detection
	var cycles [][]string

	// DFS to fetch all issues and their dependencies
	var dfs func(identifier string, path []string) error
	dfs = func(identifier string, path []string) error {
		// Cycle detection
		if stack[identifier] {
			// Found a cycle - build the cycle path
			cycleStart := -1
			for i, id := range path {
				if id == identifier {
					cycleStart = i
					break
				}
			}
			if cycleStart >= 0 {
				cycle := append(path[cycleStart:], identifier)
				cycles = append(cycles, cycle)
			}
			return nil // Continue processing other paths
		}

		// Already visited - skip
		if _, exists := visited[identifier]; exists {
			return nil
		}

		// Mark as visiting (on stack)
		stack[identifier] = true
		currentPath := append(path, identifier)

		// Fetch issue
		issue, err := s.issueClient.GetIssue(identifier)
		if err != nil {
			return fmt.Errorf("failed to fetch issue %s: %w", identifier, err)
		}

		// Create node
		node := &issueNode{
			issue:        issue,
			children:     []string{},
			dependencies: []string{},
		}

		// Extract children
		for _, child := range issue.Children.Nodes {
			node.children = append(node.children, child.Identifier)
		}

		// Extract dependencies from relations
		issueWithRel, err := s.issueClient.IssueClient().GetIssueWithRelations(identifier)
		if err == nil {
			// InverseRelations = issues that block this one
			for _, rel := range issueWithRel.InverseRelations.Nodes {
				if rel.Type == core.RelationBlocks && rel.Issue != nil {
					node.dependencies = append(node.dependencies, rel.Issue.Identifier)
				}
			}
		}

		// Store node
		visited[identifier] = node

		// Recursively fetch children
		for _, childID := range node.children {
			if err := dfs(childID, currentPath); err != nil {
				return err
			}
		}

		// Recursively fetch dependencies
		for _, depID := range node.dependencies {
			if err := dfs(depID, currentPath); err != nil {
				return err
			}
		}

		// Remove from stack (done visiting)
		delete(stack, identifier)
		return nil
	}

	// Start DFS from root
	if err := dfs(rootID, []string{}); err != nil {
		return nil, nil, err
	}

	return visited, cycles, nil
}

// convertToTasks converts Linear issues to Claude Code task format
// Implements bottom-up hierarchy: children block parent
func (s *TaskExportService) convertToTasks(nodes map[string]*issueNode) []taskwriter.ClaudeTask {
	tasks := make([]taskwriter.ClaudeTask, 0, len(nodes))

	for _, node := range nodes {
		issue := node.issue

		// Build blockedBy list: children + dependencies
		blockedBy := make([]string, 0)
		blockedBy = append(blockedBy, node.children...)
		blockedBy = append(blockedBy, node.dependencies...)

		task := taskwriter.ClaudeTask{
			ID:          issue.Identifier,
			Subject:     issue.Title,
			Description: s.buildTaskDescription(issue),
			ActiveForm:  s.buildActiveForm(issue.Title),
			Status:      "pending",
			Blocks:      []string{}, // Computed from other tasks' blockedBy
			BlockedBy:   blockedBy,
		}

		tasks = append(tasks, task)
	}

	return tasks
}

// buildTaskDescription creates a Claude Code task description from a Linear issue
func (s *TaskExportService) buildTaskDescription(issue *core.Issue) string {
	var b strings.Builder

	// Header with Linear issue link
	b.WriteString(fmt.Sprintf("**Linear Issue:** %s\n\n", issue.Identifier))

	// Description
	if issue.Description != "" {
		b.WriteString(issue.Description)
		b.WriteString("\n\n")
	}

	// Metadata section
	b.WriteString("---\n")
	b.WriteString(fmt.Sprintf("**State:** %s\n", issue.State.Name))

	if issue.Priority != nil {
		b.WriteString(fmt.Sprintf("**Priority:** %d\n", *issue.Priority))
	}

	if issue.Assignee != nil {
		b.WriteString(fmt.Sprintf("**Assignee:** %s\n", issue.Assignee.Name))
	}

	if issue.Estimate != nil {
		b.WriteString(fmt.Sprintf("**Estimate:** %.0f\n", *issue.Estimate))
	}

	if issue.DueDate != nil && *issue.DueDate != "" {
		b.WriteString(fmt.Sprintf("**Due:** %s\n", *issue.DueDate))
	}

	if issue.URL != "" {
		b.WriteString(fmt.Sprintf("**URL:** %s\n", issue.URL))
	}

	return b.String()
}

// buildActiveForm converts a title to present continuous form
// Examples: "Fix bug" -> "Fixing bug", "Add feature" -> "Adding feature"
func (s *TaskExportService) buildActiveForm(title string) string {
	if title == "" {
		return "Working on task"
	}

	// Simple heuristic: convert common imperative verbs to -ing form
	words := strings.Fields(title)
	if len(words) == 0 {
		return "Working on task"
	}

	firstWord := strings.ToLower(words[0])

	// Map of common imperative verbs to -ing forms
	verbMap := map[string]string{
		"add":        "Adding",
		"fix":        "Fixing",
		"update":     "Updating",
		"create":     "Creating",
		"implement":  "Implementing",
		"remove":     "Removing",
		"delete":     "Deleting",
		"refactor":   "Refactoring",
		"improve":    "Improving",
		"optimize":   "Optimizing",
		"test":       "Testing",
		"debug":      "Debugging",
		"migrate":    "Migrating",
		"upgrade":    "Upgrading",
		"downgrade":  "Downgrading",
		"install":    "Installing",
		"configure":  "Configuring",
		"setup":      "Setting up",
		"set":        "Setting",
		"write":      "Writing",
		"read":       "Reading",
		"parse":      "Parsing",
		"validate":   "Validating",
		"verify":     "Verifying",
		"check":      "Checking",
		"review":     "Reviewing",
		"investigate": "Investigating",
		"analyze":    "Analyzing",
		"design":     "Designing",
		"plan":       "Planning",
		"research":   "Researching",
		"document":   "Documenting",
		"clean":      "Cleaning",
		"rebuild":    "Rebuilding",
		"deploy":     "Deploying",
		"release":    "Releasing",
		"merge":      "Merging",
		"rebase":     "Rebasing",
		"revert":     "Reverting",
		"restore":    "Restoring",
		"backup":     "Backing up",
		"archive":    "Archiving",
	}

	if ingForm, exists := verbMap[firstWord]; exists {
		// Replace first word with -ing form
		words[0] = ingForm
		return strings.Join(words, " ")
	}

	// Fallback: capitalize first word and return as-is
	// Don't try to add -ing for unknown verbs as it can produce incorrect results
	words[0] = strings.Title(firstWord)
	return strings.Join(words, " ")
}
