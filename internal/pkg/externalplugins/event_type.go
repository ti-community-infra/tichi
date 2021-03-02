package externalplugins

// EventType is an alias of string which describe the GitHub webhook event.
type EventType = string

// Event type constants.
const (
	IssuesEvent       EventType = "issues"
	IssueCommentEvent EventType = "issue_comment"

	PullRequestEvent              EventType = "pull_request"
	PullRequestReviewEvent        EventType = "pull_request_review"
	PullRequestReviewCommentEvent EventType = "pull_request_review_comment"

	PushEvent EventType = "push"

	StatusEvent EventType = "status"
)
