// Package identifiers provides format detection and parsing for Linear identifiers.
//
// This package handles recognition and parsing of:
//   - Issue identifiers (TEAM-123 format)
//   - Email addresses
//   - UUIDs
//
// # Issue Identifier Parsing
//
// Issue identifiers follow the format "TEAM-NUMBER" where TEAM is the team key
// and NUMBER is the issue number:
//
//	if identifiers.IsIssueIdentifier("CEN-123") {
//	    team, num, err := identifiers.ParseIssueIdentifier("CEN-123")
//	    // team = "CEN", num = "123"
//	}
//
// # Format Detection
//
// The package provides quick format detection functions:
//
//	identifiers.IsEmail("user@example.com")     // true
//	identifiers.IsUUID("550e8400-e29b-41d4-...") // true
//	identifiers.IsIssueIdentifier("ENG-42")     // true
//
// These functions are used throughout the codebase to determine how to
// resolve user-provided identifiers to Linear UUIDs.
package identifiers
