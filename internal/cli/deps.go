package cli

import (
	"fmt"
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
	"github.com/dominikbraun/graph"
	"github.com/spf13/cobra"
)

// DepNode represents a node in the dependency graph
type DepNode struct {
	ID         string
	Identifier string
	Title      string
	State      string
}

// DepEdge represents a directed edge (dependency) in the graph
type DepEdge struct {
	From string // blocks
	To   string // is blocked by
	Type string
}

func newDepsCmd() *cobra.Command {
	var teamKey string

	cmd := &cobra.Command{
		Use:   "deps [issue-id]",
		Short: "Visualize issue dependency graph",
		Long: `Display issue dependencies as an ASCII tree.

Shows blocking relationships between issues. Each issue shows:
  - What it blocks (→)
  - What blocks it (←)

Detects and warns about circular dependencies.`,
		Example: `  # Show dependencies for a single issue
  linear deps ENG-100

  # Show all dependencies for a team
  linear deps --team ENG`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getLinearClient()
			if err != nil {
				return err
			}

			if len(args) > 0 {
				// Single issue mode
				return showIssueDeps(client, args[0])
			}

			if teamKey != "" {
				// Team mode
				return showTeamDeps(client, teamKey)
			}

			return fmt.Errorf("provide an issue ID or use --team to show team dependencies")
		},
	}

	cmd.Flags().StringVarP(&teamKey, "team", "t", "", "Show dependencies for all issues in team")

	return cmd
}

func showIssueDeps(client *linear.Client, issueID string) error {
	issue, err := client.Issues.GetIssueWithRelations(issueID)
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	// Build graph from this issue's relations
	nodes := make(map[string]*DepNode)
	var edges []DepEdge

	// Add root issue
	nodes[issue.Identifier] = &DepNode{
		ID:         issue.ID,
		Identifier: issue.Identifier,
		Title:      issue.Title,
		State:      issue.State.Name,
	}

	// Process relations (what this issue blocks)
	for _, rel := range issue.Relations.Nodes {
		if rel.Type == "blocks" && rel.RelatedIssue != nil {
			nodes[rel.RelatedIssue.Identifier] = &DepNode{
				ID:         rel.RelatedIssue.ID,
				Identifier: rel.RelatedIssue.Identifier,
				Title:      rel.RelatedIssue.Title,
				State:      rel.RelatedIssue.State.Name,
			}
			edges = append(edges, DepEdge{
				From: issue.Identifier,
				To:   rel.RelatedIssue.Identifier,
				Type: "blocks",
			})
		}
	}

	// Process inverse relations (what blocks this issue)
	for _, rel := range issue.InverseRelations.Nodes {
		if rel.Type == "blocks" && rel.Issue != nil {
			nodes[rel.Issue.Identifier] = &DepNode{
				ID:         rel.Issue.ID,
				Identifier: rel.Issue.Identifier,
				Title:      rel.Issue.Title,
				State:      rel.Issue.State.Name,
			}
			edges = append(edges, DepEdge{
				From: rel.Issue.Identifier,
				To:   issue.Identifier,
				Type: "blocks",
			})
		}
	}

	return renderDependencyGraph(issue.Identifier, nodes, edges)
}

func showTeamDeps(client *linear.Client, teamKey string) error {
	issues, err := client.Issues.GetTeamIssuesWithRelations(teamKey, 250)
	if err != nil {
		return fmt.Errorf("failed to get team issues: %w", err)
	}

	nodes := make(map[string]*DepNode)
	var edges []DepEdge

	for _, issue := range issues {
		nodes[issue.Identifier] = &DepNode{
			ID:         issue.ID,
			Identifier: issue.Identifier,
			Title:      issue.Title,
			State:      issue.State.Name,
		}

		// Process relations (what this issue blocks)
		for _, rel := range issue.Relations.Nodes {
			if rel.Type == "blocks" && rel.RelatedIssue != nil {
				if _, exists := nodes[rel.RelatedIssue.Identifier]; !exists {
					nodes[rel.RelatedIssue.Identifier] = &DepNode{
						ID:         rel.RelatedIssue.ID,
						Identifier: rel.RelatedIssue.Identifier,
						Title:      rel.RelatedIssue.Title,
						State:      rel.RelatedIssue.State.Name,
					}
				}
				edges = append(edges, DepEdge{
					From: issue.Identifier,
					To:   rel.RelatedIssue.Identifier,
					Type: "blocks",
				})
			}
		}
	}

	if len(edges) == 0 {
		fmt.Printf("No dependencies found for team %s\n", teamKey)
		return nil
	}

	return renderTeamDependencyGraph(teamKey, nodes, edges)
}

func renderDependencyGraph(rootID string, nodes map[string]*DepNode, edges []DepEdge) error {
	var b strings.Builder

	root := nodes[rootID]
	if root == nil {
		return fmt.Errorf("root issue not found")
	}

	// Header
	b.WriteString(fmt.Sprintf("DEPENDENCY GRAPH: %s\n", rootID))
	b.WriteString(strings.Repeat("═", 50))
	b.WriteString("\n")

	// Root issue
	b.WriteString(fmt.Sprintf("%s %s\n", root.Identifier, truncateTitle(root.Title, 40)))

	// Find issues this blocks
	var blocks []DepEdge
	for _, e := range edges {
		if e.From == rootID {
			blocks = append(blocks, e)
		}
	}

	// Find issues blocking this
	var blockedBy []DepEdge
	for _, e := range edges {
		if e.To == rootID {
			blockedBy = append(blockedBy, e)
		}
	}

	// Render what this issue blocks
	for i, e := range blocks {
		prefix := "├─"
		if i == len(blocks)-1 && len(blockedBy) == 0 {
			prefix = "└─"
		}
		node := nodes[e.To]
		if node != nil {
			b.WriteString(fmt.Sprintf("%s → %s [%s] %s\n",
				prefix, node.Identifier, node.State, truncateTitle(node.Title, 30)))
		}
	}

	// Render what blocks this issue
	for i, e := range blockedBy {
		prefix := "├─"
		if i == len(blockedBy)-1 {
			prefix = "└─"
		}
		node := nodes[e.From]
		if node != nil {
			b.WriteString(fmt.Sprintf("%s ← %s [%s] %s\n",
				prefix, node.Identifier, node.State, truncateTitle(node.Title, 30)))
		}
	}

	// Summary
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%d issues, %d dependencies", len(nodes), len(edges)))

	// Check for cycles
	cycles := detectCycles(nodes, edges)
	if len(cycles) > 0 {
		b.WriteString("\n\n⚠ Circular dependencies detected:\n")
		for _, cycle := range cycles {
			b.WriteString(fmt.Sprintf("  %s\n", strings.Join(cycle, " → ")))
		}
	}

	b.WriteString("\n")
	fmt.Print(b.String())
	return nil
}

func renderTeamDependencyGraph(teamKey string, nodes map[string]*DepNode, edges []DepEdge) error {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("DEPENDENCY GRAPH: Team %s\n", teamKey))
	b.WriteString(strings.Repeat("═", 50))
	b.WriteString("\n")

	// Find root issues (issues that block others but aren't blocked)
	blockers := make(map[string]bool)
	blocked := make(map[string]bool)
	for _, e := range edges {
		blockers[e.From] = true
		blocked[e.To] = true
	}

	var roots []string
	for id := range blockers {
		if !blocked[id] {
			roots = append(roots, id)
		}
	}

	// If no clear roots, just pick the first few blockers
	if len(roots) == 0 {
		for id := range blockers {
			roots = append(roots, id)
			if len(roots) >= 5 {
				break
			}
		}
	}

	// Build adjacency map
	adj := make(map[string][]string)
	for _, e := range edges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	// Render each root tree
	rendered := make(map[string]bool)
	for _, rootID := range roots {
		if rendered[rootID] {
			continue
		}
		renderSubtree(&b, rootID, "", true, nodes, adj, rendered, 0)
	}

	// Summary
	b.WriteString(strings.Repeat("─", 50))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("%d issues with dependencies, %d blocking relationships", len(nodes), len(edges)))

	// Check for cycles
	cycles := detectCycles(nodes, edges)
	if len(cycles) > 0 {
		b.WriteString("\n\n⚠ Circular dependencies detected:\n")
		for _, cycle := range cycles {
			b.WriteString(fmt.Sprintf("  %s\n", strings.Join(cycle, " → ")))
		}
	}

	b.WriteString("\n")
	fmt.Print(b.String())
	return nil
}

func renderSubtree(b *strings.Builder, nodeID, prefix string, isLast bool, nodes map[string]*DepNode, adj map[string][]string, rendered map[string]bool, depth int) {
	if depth > 10 {
		return // Prevent infinite recursion
	}

	node := nodes[nodeID]
	if node == nil {
		return
	}

	// Determine connector
	connector := "├─"
	if isLast {
		connector = "└─"
	}
	if prefix == "" {
		connector = ""
	}

	// Render this node
	if rendered[nodeID] {
		b.WriteString(fmt.Sprintf("%s%s %s [%s] (see above)\n",
			prefix, connector, node.Identifier, node.State))
		return
	}
	rendered[nodeID] = true

	b.WriteString(fmt.Sprintf("%s%s %s [%s] %s\n",
		prefix, connector, node.Identifier, node.State, truncateTitle(node.Title, 35)))

	// Render children
	children := adj[nodeID]
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "   "
		} else {
			childPrefix += "│  "
		}
	}

	for i, childID := range children {
		isLastChild := i == len(children)-1
		renderSubtree(b, childID, childPrefix, isLastChild, nodes, adj, rendered, depth+1)
	}
}

func detectCycles(nodes map[string]*DepNode, edges []DepEdge) [][]string {
	// Build graph using dominikbraun/graph
	g := graph.New(graph.StringHash, graph.Directed())

	// Add vertices
	for id := range nodes {
		_ = g.AddVertex(id)
	}

	// Add edges
	for _, e := range edges {
		_ = g.AddEdge(e.From, e.To)
	}

	// Detect cycles
	cycles, _ := graph.StronglyConnectedComponents(g)

	// Filter to only cycles (components with more than 1 node or self-loops)
	var result [][]string
	for _, component := range cycles {
		if len(component) > 1 {
			// Add first element again to show the cycle
			cycle := append(component, component[0])
			result = append(result, cycle)
		} else if len(component) == 1 {
			// Check for self-loop
			nodeID := component[0]
			for _, e := range edges {
				if e.From == nodeID && e.To == nodeID {
					result = append(result, []string{nodeID, nodeID})
					break
				}
			}
		}
	}

	return result
}

func truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	if maxLen <= 3 {
		return title[:maxLen]
	}
	return title[:maxLen-3] + "..."
}
