# Linear Client Integration Tests

This directory contains comprehensive tests for all read operations in the Linear client.

## Test Files

### 1. `integration_read_test.go`
Full integration test that exercises all read operations against the real Linear API.
- Requires valid Linear API credentials
- Tests all read functions comprehensively
- Includes performance tests for concurrent operations
- Can be skipped with `SKIP_INTEGRATION_TESTS=1`

Run with:
```bash
# With Linear API token
export LINEAR_TOKEN=your-token-here
go test -v ./internal/linear -run TestReadOperationsIntegration

# Skip integration tests
SKIP_INTEGRATION_TESTS=1 go test ./internal/linear
```

### 2. `read_operations_test.go`
Mock-based tests that don't require Linear API access.
- Tests all read operations with mocked responses
- Verifies correct handling of metadata extraction
- Tests error scenarios
- Always runs, no credentials needed

Run with:
```bash
go test -v ./internal/linear -run TestAllReadOperations
```

### 3. `test_all_reads.go` (in project root)
Standalone test program for manual testing.
- Human-readable output
- Shows example data from your Linear workspace
- Good for verifying the client works with your data

Run with:
```bash
./run_read_tests.sh
# or
LINEAR_TOKEN=your-token go run test_all_reads.go
```

## Read Operations Tested

1. **Authentication & Connection**
   - `TestConnection()` - Verify API connectivity
   - `GetViewer()` - Get authenticated user details
   - `GetAppUserID()` - Get user ID only

2. **Team Operations**
   - `GetTeams()` - List all teams in workspace

3. **Workflow Operations**
   - `GetWorkflowStates(teamID)` - Get available workflow states

4. **Issue Operations**
   - `ListAssignedIssues(userID)` - Get issues assigned to user
   - `GetIssue(issueID)` - Get full issue details with metadata
   - `GetIssueWithProjectContext(issueID)` - Include project info
   - `GetIssueWithParentContext(issueID)` - Include parent issue info
   - `GetSubIssues(parentIssueID)` - Get child issues

5. **Project Operations**
   - `ListAllProjects()` - Get all projects
   - `ListUserProjects(userID)` - Get projects with user's issues
   - `GetProject(projectID)` - Get full project details with metadata

6. **Notification Operations**
   - `GetNotifications(includeRead, limit)` - Get user notifications

7. **Comment Operations**
   - `GetCommentWithReplies(commentID)` - Get comment thread

## Features Verified

- **Metadata Extraction**: Both issues and projects can have metadata stored in descriptions
- **State IDs**: All workflow states include both ID and name
- **Concurrent Operations**: HTTP client connection pooling works correctly
- **Error Handling**: Proper error messages for various failure scenarios

## No Delete Operations

As requested, this codebase contains no delete operations. All functions are read-only or create/update only.