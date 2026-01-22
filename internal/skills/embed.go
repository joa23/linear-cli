// Package skills provides embedded skill templates for Claude Code.
package skills

import "embed"

//go:embed prd/* triage/* cycle-plan/* retro/* deps/* link-deps/* release/*
var SkillFiles embed.FS

// SkillInfo describes an available skill
type SkillInfo struct {
	Name        string
	Description string
	Dir         string
}

// AvailableSkills lists all skills that can be installed
var AvailableSkills = []SkillInfo{
	{
		Name:        "prd",
		Description: "Create agent-friendly tickets with PRDs, sub-issues, and success criteria",
		Dir:         "prd",
	},
	{
		Name:        "triage",
		Description: "Analyze and prioritize Linear backlog based on staleness and blockers",
		Dir:         "triage",
	},
	{
		Name:        "cycle-plan",
		Description: "Plan cycles using velocity analytics and capacity",
		Dir:         "cycle-plan",
	},
	{
		Name:        "retro",
		Description: "Generate sprint retrospective analysis from completed cycles",
		Dir:         "retro",
	},
	{
		Name:        "deps",
		Description: "Analyze dependency chains, find blockers and circular dependencies",
		Dir:         "deps",
	},
	{
		Name:        "link-deps",
		Description: "Discover and link related issues as dependencies across your backlog",
		Dir:         "link-deps",
	},
	{
		Name:        "release",
		Description: "Pre-release checklist ensuring CHANGELOG, versions, tests, and docs are updated",
		Dir:         "release",
	},
}

// GetSkillByName returns a skill by name, or nil if not found
func GetSkillByName(name string) *SkillInfo {
	for _, s := range AvailableSkills {
		if s.Name == name {
			return &s
		}
	}
	return nil
}
