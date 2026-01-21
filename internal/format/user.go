package format

import (
	"strings"

	"github.com/joa23/linear-cli/internal/linear"
)

// User formats a single user
func (f *Formatter) User(user *linear.User) string {
	if user == nil {
		return ""
	}

	var b strings.Builder

	// Header
	name := user.Name
	if user.DisplayName != "" {
		name = user.DisplayName
	}
	b.WriteString(fmtSprintf("%s\n", name))
	b.WriteString(line(50))
	b.WriteString("\n")

	// Email
	b.WriteString(fmtSprintf("Email: %s\n", user.Email))

	// Status
	status := "Active"
	if !user.Active {
		status = "Inactive"
	}
	if user.Admin {
		status += " (Admin)"
	}
	b.WriteString(fmtSprintf("Status: %s\n", status))

	// Created
	b.WriteString(fmtSprintf("Joined: %s\n", formatDate(user.CreatedAt)))

	// Teams if available
	if len(user.Teams) > 0 {
		b.WriteString("\nTEAMS\n")
		b.WriteString(line(30))
		b.WriteString("\n")
		for _, team := range user.Teams {
			b.WriteString(fmtSprintf("  %s (%s)\n", team.Name, team.Key))
		}
	}

	return b.String()
}

// UserList formats a list of users with optional pagination
func (f *Formatter) UserList(users []linear.User, page *Pagination) string {
	if len(users) == 0 {
		return "No users found."
	}

	var b strings.Builder

	// Header
	b.WriteString(fmtSprintf("USERS (%d)\n", len(users)))
	b.WriteString(line(40))
	b.WriteString("\n")

	// Format each user
	for _, user := range users {
		b.WriteString(f.userCompact(&user))
		b.WriteString("\n")
	}

	// Pagination footer
	if page != nil && page.HasNextPage && page.EndCursor != "" {
		b.WriteString(line(40))
		b.WriteString("\n")
		b.WriteString(fmtSprintf("Next: cursor=%s\n", page.EndCursor))
	}

	return b.String()
}

func (f *Formatter) userCompact(user *linear.User) string {
	var b strings.Builder

	// Line 1: Name and email
	name := user.Name
	if user.DisplayName != "" {
		name = user.DisplayName
	}
	b.WriteString(fmtSprintf("%s <%s>\n", name, user.Email))

	// Line 2: Status and teams
	var meta []string

	if !user.Active {
		meta = append(meta, "Inactive")
	}
	if user.Admin {
		meta = append(meta, "Admin")
	}
	if len(user.Teams) > 0 {
		var teamKeys []string
		for _, team := range user.Teams {
			teamKeys = append(teamKeys, team.Key)
		}
		meta = append(meta, fmtSprintf("Teams: %s", strings.Join(teamKeys, ", ")))
	}

	if len(meta) > 0 {
		b.WriteString(fmtSprintf("  %s\n", strings.Join(meta, " | ")))
	}

	return b.String()
}
