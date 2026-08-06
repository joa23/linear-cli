package service

import (
	"fmt"
	"sort"
	"strings"

	"github.com/joa23/linear-cli/pkg/linear/core"
)

// LabelConflictGroup describes multiple child labels selected from one
// mutually exclusive parent group.
type LabelConflictGroup struct {
	Labels []core.Label
	Parent core.LabelRef
}

// LabelConflictError is returned before a mutation when a label selection
// contains multiple children from one exclusive parent group.
type LabelConflictError struct {
	Groups []LabelConflictGroup
}

func (e *LabelConflictError) Error() string {
	parts := make([]string, 0, len(e.Groups))
	for _, group := range e.Groups {
		names := make([]string, 0, len(group.Labels))
		for _, label := range group.Labels {
			names = append(names, fmt.Sprintf("%q", label.Name))
		}
		groupName := group.Parent.Name
		if groupName == "" {
			groupName = group.Parent.ID
		}
		parts = append(parts, fmt.Sprintf("labels %s belong to the exclusive group %q — pick one", joinLabelNames(names), groupName))
	}
	return strings.Join(parts, "; ")
}

func validateLabelSelection(labels []core.Label) error {
	groups := make(map[string]LabelConflictGroup)
	seen := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		if label.ID == "" {
			return fmt.Errorf("label metadata is incomplete for %q: label ID is missing", label.Name)
		}
		if _, ok := seen[label.ID]; ok {
			continue
		}
		seen[label.ID] = struct{}{}
		if label.Parent == nil {
			continue
		}
		if label.Parent.ID == "" {
			return fmt.Errorf("label metadata is incomplete for %q: parent ID is missing", label.Name)
		}
		group := groups[label.Parent.ID]
		group.Parent = *label.Parent
		group.Labels = append(group.Labels, label)
		groups[label.Parent.ID] = group
	}

	conflicts := make([]LabelConflictGroup, 0)
	for _, group := range groups {
		if len(group.Labels) < 2 {
			continue
		}
		sort.Slice(group.Labels, func(i, j int) bool {
			left, right := strings.ToLower(group.Labels[i].Name), strings.ToLower(group.Labels[j].Name)
			if left == right {
				return group.Labels[i].ID < group.Labels[j].ID
			}
			return left < right
		})
		conflicts = append(conflicts, group)
	}
	if len(conflicts) == 0 {
		return nil
	}
	sort.Slice(conflicts, func(i, j int) bool {
		left, right := strings.ToLower(conflicts[i].Parent.Name), strings.ToLower(conflicts[j].Parent.Name)
		if left == right {
			return conflicts[i].Parent.ID < conflicts[j].Parent.ID
		}
		return left < right
	})
	return &LabelConflictError{Groups: conflicts}
}

func joinLabelNames(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, ", ")
	}
	if len(names) == 2 {
		return names[0] + " and " + names[1]
	}
	return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
}

func sortLabels(labels []core.Label) []core.Label {
	byID := make(map[string]core.Label, len(labels))
	for _, label := range labels {
		byID[label.ID] = label
	}
	result := make([]core.Label, 0, len(byID))
	for _, label := range byID {
		result = append(result, label)
	}
	sort.Slice(result, func(i, j int) bool {
		left, right := strings.ToLower(result[i].Name), strings.ToLower(result[j].Name)
		if left == right {
			return result[i].ID < result[j].ID
		}
		return left < right
	})
	return result
}
