package issuetriage

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"

	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/github"
)

var botName = "ti-chi-bot"

// fghc is a fake GitHub client.
type fghc struct {
	Issues  map[int]*github.Issue
	IssueID int

	IssueComments  map[int][]github.IssueComment
	IssueCommentID int
	// org/repo#number:body
	IssueCommentsAdded []string
	// org/repo#issuecommentid
	IssueCommentsDeleted []string

	// org/repo#number:label
	IssueLabelsAdded    []string
	IssueLabelsExisting []string
	IssueLabelsRemoved  []string

	PullRequests     map[int]*github.PullRequest
	CombinedStatuses map[string]*github.CombinedStatus
	CreatedStatuses  map[string][]github.Status
	// Error will be returned if set. Currently only implemented for CreateStatus
	Error error

	// QueryResult will be returned for Query interface.
	QueryResult interface{}

	// lock to be thread safe
	lock sync.RWMutex
}

func (f *fghc) AddLabel(org, repo string, number int, label string) error {
	return f.AddLabels(org, repo, number, label)
}

func (f *fghc) AddLabels(org, repo string, number int, labels ...string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	for _, label := range labels {
		labelString := fmt.Sprintf("%s/%s#%d:%s", org, repo, number, label)
		if sets.NewString(f.IssueLabelsAdded...).Has(labelString) {
			return fmt.Errorf("cannot add %v to %s/%s/#%d", label, org, repo, number)
		}
		f.IssueLabelsAdded = append(f.IssueLabelsAdded, labelString)
	}
	return nil
}

func (f *fghc) RemoveLabel(owner, repo string, number int, label string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	labelString := fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, label)
	if !sets.NewString(f.IssueLabelsRemoved...).Has(labelString) {
		f.IssueLabelsRemoved = append(f.IssueLabelsRemoved, labelString)
		return nil
	}
	return fmt.Errorf("cannot remove %v from %s/%s/#%d", label, owner, repo, number)
}

func (f *fghc) CreateStatus(owner, repo, sha string, s github.Status) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	if f.Error != nil {
		return f.Error
	}
	if f.CreatedStatuses == nil {
		f.CreatedStatuses = make(map[string][]github.Status)
	}
	statuses := f.CreatedStatuses[sha]
	var updated bool
	for i := range statuses {
		if statuses[i].Context == s.Context {
			statuses[i] = s
			updated = true
		}
	}
	if !updated {
		statuses = append(statuses, s)
	}
	f.CreatedStatuses[sha] = statuses
	f.CombinedStatuses[sha] = &github.CombinedStatus{
		SHA:      sha,
		Statuses: statuses,
	}
	return nil
}

func (f *fghc) GetCombinedStatus(owner, repo, ref string) (*github.CombinedStatus, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.CombinedStatuses[ref], nil
}

func (f *fghc) GetPullRequest(owner, repo string, number int) (*github.PullRequest, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	val, exists := f.PullRequests[number]
	if !exists {
		return nil, fmt.Errorf("pull request number %d does not exist", number)
	}
	return val, nil
}

func (f *fghc) GetIssue(owner, repo string, number int) (*github.Issue, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	val, exists := f.Issues[number]
	if !exists {
		return nil, fmt.Errorf("issue number %d does not exist", number)
	}
	return val, nil
}

func (f *fghc) CreateComment(owner, repo string, number int, comment string) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.IssueCommentID++
	f.IssueCommentsAdded = append(f.IssueCommentsAdded, fmt.Sprintf("%s/%s#%d:%s", owner, repo, number, comment))
	f.IssueComments[number] = append(f.IssueComments[number], github.IssueComment{
		ID:   f.IssueCommentID,
		Body: comment,
		User: github.User{Login: botName},
	})
	return nil
}

func (f *fghc) BotUserChecker() (func(candidate string) bool, error) {
	return func(candidate string) bool {
		candidate = strings.TrimSuffix(candidate, "[bot]")
		return candidate == botName
	}, nil
}

func (f *fghc) Query(ctx context.Context, q interface{}, vars map[string]interface{}) error {
	sq, ok := q.(*referencePullRequestQuery)
	if !ok {
		return errors.New("unexpected query type")
	}

	res, ok := f.QueryResult.(*referencePullRequestQuery)
	if !ok {
		return errors.New("unexpected query result type")
	}

	sq.Repository = res.Repository
	sq.RateLimit = res.RateLimit

	return nil
}

func TestHandleIssueEvent(t *testing.T) {
	var testcases = []struct {
		name   string
		config externalplugins.TiCommunityIssueTriage
		action github.IssueEventAction
		// labels which have already existed on the issue.
		labels []string
		// label that will be labeled or unlabeled.
		label       string
		statusState string
		issues      map[int]*github.Issue

		expectAddedLabels        []string
		expectRemovedLabels      []string
		expectCreatedStatusState string
	}{
		{
			name:   "add a security/major label to a type/bug issue",
			action: github.IssueActionLabeled,
			labels: []string{
				bugTypeLabel,
				majorSeverityLabel,
			},
			label: majorSeverityLabel,

			expectAddedLabels: []string{
				"org/repo#1:may-affects/5.1",
				"org/repo#1:may-affects/5.2",
				"org/repo#1:may-affects/5.3",
			},
			expectRemovedLabels: []string{},
		},
		{
			name:   "add a security/major label to a type/feature issue",
			action: github.IssueActionLabeled,
			labels: []string{
				"type/feature",
			},
			label: majorSeverityLabel,

			expectAddedLabels:   []string{},
			expectRemovedLabels: []string{},
		},
		{
			name:   "add a security/major label to a type/bug issue that has affects label",
			action: github.IssueActionLabeled,
			labels: []string{
				bugTypeLabel,
				majorSeverityLabel,
				"affects/5.2",
			},
			label: majorSeverityLabel,

			expectAddedLabels: []string{
				"org/repo#1:may-affects/5.1",
				"org/repo#1:may-affects/5.3",
			},
			expectRemovedLabels: []string{},
		},
		{
			name:   "add a security/major label to a type/bug issue that has may-affects label",
			action: github.IssueActionLabeled,
			labels: []string{
				bugTypeLabel,
				majorSeverityLabel,
				"may-affects/5.2",
			},
			label: majorSeverityLabel,

			expectAddedLabels: []string{
				"org/repo#1:may-affects/5.1",
				"org/repo#1:may-affects/5.3",
			},
			expectRemovedLabels: []string{},
		},
		{
			name:   "add a affects label from the type/bug issue that has may-affects label",
			action: github.IssueActionLabeled,
			labels: []string{
				bugTypeLabel,
				majorSeverityLabel,
				"affects/5.2",
				"may-affects/5.1",
				"may-affects/5.2",
				"may-affects/5.3",
			},
			label: "affects/5.2",

			expectAddedLabels: []string{},
			expectRemovedLabels: []string{
				"org/repo#1:may-affects/5.2",
			},
		},
		{
			name:   "the type/bug issue removed a may-affects label but still has other may-affects labels",
			action: github.IssueActionUnlabeled,
			label:  "may-affects/5.2",
			labels: []string{
				bugTypeLabel,
				majorSeverityLabel,
				"affects/5.2",
				"may-affects/5.1",
				"may-affects/5.3",
			},

			expectAddedLabels: []string{
				"org/repo#2:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:   "the type/bug issue removed a may-affects label and does not have any other may-affects labels",
			action: github.IssueActionUnlabeled,
			label:  "may-affects/5.2",
			labels: []string{
				bugTypeLabel,
				majorSeverityLabel,
				"affects/5.2",
			},
			statusState: github.StatusPending,

			expectAddedLabels: []string{
				"org/repo#2:needs-cherry-pick-release-5.2",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
	}

	for _, testcase := range testcases {
		tc := testcase

		// Prepare issue labels data.
		prefix := "org/repo#1:"
		labels := make([]github.Label, 0)
		labelsWithPrefix := make([]string, 0)
		for _, label := range tc.labels {
			labels = append(labels, github.Label{
				Name: label,
			})
			labelsWithPrefix = append(labelsWithPrefix, prefix+label)
		}

		// Prepare plugin config.
		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityIssueTriage = []externalplugins.TiCommunityIssueTriage{
			{
				Repos:                     []string{"org/repo"},
				MaintainVersions:          []string{"5.1", "5.2", "5.3"},
				AffectsLabelPrefix:        "affects/",
				MayAffectsLabelPrefix:     "may-affects/",
				NeedTriagedLabel:          "do-not-merge/needs-triage-completed",
				NeedCherryPickLabelPrefix: "needs-cherry-pick-release-",
			},
		}
		ca := &externalplugins.ConfigAgent{}
		ca.Set(cfg)

		// Mock GitHub client and webhook event.
		repo := repository{
			Owner: struct{ Login githubql.String }{
				Login: githubql.String("org"),
			},
			Name: githubql.String("repo"),
			DefaultBranchRef: struct{ Name githubql.String }{
				Name: githubql.String("master"),
			},
		}

		pullRequest1 := pullRequest{
			Number:      2,
			Repository:  repo,
			State:       githubql.String(github.PullRequestStateOpen),
			BaseRefName: "master",
			HeadRefOid:  "sha",
			Labels: struct {
				Nodes []struct {
					Name githubql.String
				}
			}{},
			Body: "Issue Number: close #1",
		}

		pullRequest2 := pullRequest{
			Number:      3,
			Repository:  repo,
			State:       githubql.String(github.PullRequestStateOpen),
			BaseRefName: "release-5.1",
			HeadRefOid:  "sha2",
			Labels: struct {
				Nodes []struct {
					Name githubql.String
				}
			}{},
			Body: "Issue Number: close #1",
		}

		prStatus := github.StatusSuccess
		if len(tc.statusState) != 0 {
			prStatus = tc.statusState
		}

		fc := &fghc{
			Issues:              tc.issues,
			IssueLabelsAdded:    []string{},
			IssueLabelsRemoved:  []string{},
			IssueLabelsExisting: labelsWithPrefix,
			CombinedStatuses: map[string]*github.CombinedStatus{
				"sha": {
					SHA: "sha",
					Statuses: []github.Status{
						{
							State:       prStatus,
							Description: "...",
							Context:     issueNeedTriagedContextName,
						},
					},
				},
				"sha2": {
					SHA: "sha",
					Statuses: []github.Status{
						{
							State:       github.StatusSuccess,
							Description: "...",
							Context:     issueNeedTriagedContextName,
						},
					},
				},
			},
			QueryResult: &referencePullRequestQuery{
				Repository: queryRepository{
					Issue: issue{
						TimelineItems: timelineItems{
							Nodes: []timelineItemNode{
								{
									CrossReferencedEvent: crossReferencedEvent{
										Source: crossReferencedEventSource{
											PullRequest: pullRequest1,
										},
										WillCloseTarget: true,
									},
								},
								{
									CrossReferencedEvent: crossReferencedEvent{
										Source: crossReferencedEventSource{
											PullRequest: pullRequest2,
										},
										WillCloseTarget: false,
									},
								},
							},
						},
					},
				},
				RateLimit: rateLimit{
					Cost:      githubql.Int(5),
					Remaining: githubql.Int(100),
				},
			},
		}
		ie := &github.IssueEvent{
			Action: tc.action,
			Label: github.Label{
				Name: tc.label,
			},
			Issue: github.Issue{
				Number: 1,
				Labels: labels,
			},
			Repo: github.Repo{
				Owner: github.User{
					Login: "org",
				},
				Name: "repo",
			},
		}

		getSecret := func() []byte {
			return []byte("sha=abcdefg")
		}

		getGithubToken := func() []byte {
			return []byte("token")
		}

		// Mock Server.
		s := &Server{
			ConfigAgent:            ca,
			GitHubClient:           fc,
			WebhookSecretGenerator: getSecret,
			GitHubTokenGenerator:   getGithubToken,
			Log:                    logrus.StandardLogger().WithField("client", "issue-triage"),
		}

		err := s.handleIssueEvent(ie, logrus.WithField("plugin", PluginName))
		if err != nil {
			t.Errorf("For case [%s], didn't expect error: %v", tc.name, err)
		}

		if tc.expectAddedLabels != nil {
			sort.Strings(tc.expectAddedLabels)
			sort.Strings(fc.IssueLabelsAdded)
			if !reflect.DeepEqual(tc.expectAddedLabels, fc.IssueLabelsAdded) {
				t.Errorf("For case [%s], expect added labels: \n%v\nbut got: \n%v\n",
					tc.name, tc.expectAddedLabels, fc.IssueLabelsAdded)
			}
		}

		if tc.expectRemovedLabels != nil {
			sort.Strings(tc.expectRemovedLabels)
			sort.Strings(fc.IssueLabelsRemoved)
			if !reflect.DeepEqual(tc.expectRemovedLabels, fc.IssueLabelsRemoved) {
				t.Errorf("For case [%s], expect removed labels: \n%v\nbut got: \n%v\n",
					tc.name, tc.expectRemovedLabels, fc.IssueLabelsRemoved)
			}
		}

		if len(tc.expectCreatedStatusState) != 0 {
			createdStatuses, ok := fc.CreatedStatuses["sha"]
			if !ok || len(createdStatuses) != 1 {
				t.Errorf("For case [%s], expect created status: %s, but got: none.\n",
					tc.name, tc.expectCreatedStatusState)
			} else if tc.expectCreatedStatusState != createdStatuses[0].State {
				t.Errorf("For case [%s], expect status state: %s, but got: %s.\n",
					tc.name, tc.expectCreatedStatusState, createdStatuses[0].State)
			}
		}
	}
}

func TestHandlePullRequestEvent(t *testing.T) {
	var testcases = []struct {
		name         string
		config       externalplugins.TiCommunityIssueTriage
		action       github.PullRequestEventAction
		labels       []string
		targetBranch string
		body         string
		draft        bool
		state        string
		issues       map[int]*github.Issue
		statusState  string

		expectAddedLabels        []string
		expectRemovedLabels      []string
		expectCreatedStatusState string
	}{
		{
			name:         "open a pull request with empty body",
			action:       github.PullRequestActionOpened,
			body:         "",
			state:        "open",
			targetBranch: "master",

			expectAddedLabels:        []string{},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "open a pull request linked to a feature issue",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{
							Name: "type/feature",
						},
					},
				},
			},

			expectAddedLabels:        []string{},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "open a pull request linked to a bug issue without severity label",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{
							Name: "type/bug",
						},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "open a pull request linked to a bug issue with severity/moderate label",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/moderate"},
					},
				},
			},

			expectAddedLabels:        []string{},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "open a pull request linked to a bug issue with severity/major label",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "open a pull request linked to a bug issue with severity/major label and may-affects/* label",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "may-affects/5.1"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "open a pull request linked to a bug issue with severity/major label and affects/* label",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.1"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:needs-cherry-pick-release-5.1",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "open a pull request linked to a bug issue with severity/major, affects/* and may-affects/* labels",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.1"},
						{Name: "may-affects/5.2"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "open a pull request linked to a non-triaged bug issue and a feature issue",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2, ref #3",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "may-affects/5.2"},
					},
				},
				3: {
					Number: 3,
					Labels: []github.Label{
						{Name: "type/feature"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "open a pull request linked to a triaged issue and a non-triaged issue",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2, ref #3",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.2"},
					},
				},
				3: {
					Number: 3,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "may-affects/5.2"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "open a pull request linked to two triaged issue",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2, ref #3",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.2"},
					},
				},
				3: {
					Number: 3,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.3"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:needs-cherry-pick-release-5.2",
				"org/repo#1:needs-cherry-pick-release-5.3",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "open a pull request on release branch and link to a bug issue with may-affects/* labels",
			action:       github.PullRequestActionOpened,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        github.PullRequestStateOpen,
			targetBranch: "release-5.1",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "may-affects/5.2"},
					},
				},
			},

			expectAddedLabels:   []string{},
			expectRemovedLabels: []string{},
		},
		{
			name:         "edit a closed pull request linked to a bug issue with severity/major and may-affects/* labels",
			action:       github.PullRequestActionEdited,
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        github.PullRequestStateClosed,
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "may-affects/5.2"},
					},
				},
			},

			expectAddedLabels:   []string{},
			expectRemovedLabels: []string{},
		},
		{
			name:   "open a pull request with needs-cherry-pick-release-* labels and linked to a triaged issue",
			action: github.PullRequestActionOpened,
			labels: []string{
				"needs-cherry-pick-release-5.2",
			},
			body:         "Issue Number: close #2",
			state:        github.PullRequestStateOpen,
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.1"},
						{Name: "affects/5.2"},
						{Name: "affects/5.3"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:needs-cherry-pick-release-5.1",
				"org/repo#1:needs-cherry-pick-release-5.3",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
	}

	for _, testcase := range testcases {
		tc := testcase

		// Prepare issue labels data.
		prefix := "org/repo#1:"
		labels := make([]github.Label, 0)
		labelsWithPrefix := make([]string, 0)
		for _, label := range tc.labels {
			labels = append(labels, github.Label{
				Name: label,
			})
			labelsWithPrefix = append(labelsWithPrefix, prefix+label)
		}

		// Prepare plugin config.
		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityIssueTriage = []externalplugins.TiCommunityIssueTriage{
			{
				Repos:                     []string{"org/repo"},
				MaintainVersions:          []string{"5.1", "5.2", "5.3"},
				AffectsLabelPrefix:        "affects/",
				MayAffectsLabelPrefix:     "may-affects/",
				NeedTriagedLabel:          "do-not-merge/needs-triage-completed",
				NeedCherryPickLabelPrefix: "needs-cherry-pick-release-",
			},
		}
		ca := &externalplugins.ConfigAgent{}
		ca.Set(cfg)

		// Mock GitHub client and webhook event.
		fc := &fghc{
			Issues:              tc.issues,
			IssueLabelsAdded:    []string{},
			IssueLabelsRemoved:  []string{},
			IssueLabelsExisting: labelsWithPrefix,
			CombinedStatuses: map[string]*github.CombinedStatus{
				"sha": {
					SHA: "sha",
					Statuses: []github.Status{
						{
							State:       tc.statusState,
							Description: "...",
							Context:     issueNeedTriagedContextName,
						},
					},
				},
			},
		}

		pe := &github.PullRequestEvent{
			Action: tc.action,
			PullRequest: github.PullRequest{
				Number: 1,
				State:  tc.state,
				Draft:  tc.draft,
				Body:   tc.body,
				Head: github.PullRequestBranch{
					SHA: "sha",
				},
				Base: github.PullRequestBranch{
					Ref: tc.targetBranch,
				},
				Labels: labels,
			},
			Repo: github.Repo{
				Owner: github.User{
					Login: "org",
				},
				Name:          "repo",
				DefaultBranch: "master",
			},
		}

		getSecret := func() []byte {
			return []byte("sha=abcdefg")
		}

		getGithubToken := func() []byte {
			return []byte("token")
		}

		// Mock Server.
		s := &Server{
			ConfigAgent:            ca,
			GitHubClient:           fc,
			WebhookSecretGenerator: getSecret,
			GitHubTokenGenerator:   getGithubToken,
			Log:                    logrus.StandardLogger().WithField("client", "issue-triage"),
		}

		err := s.handlePullRequestEvent(pe, logrus.WithField("plugin", PluginName))
		if err != nil {
			t.Errorf("For case [%s], didn't expect error: %v", tc.name, err)
		}

		if tc.expectAddedLabels != nil {
			sort.Strings(tc.expectAddedLabels)
			sort.Strings(fc.IssueLabelsAdded)
			if !reflect.DeepEqual(tc.expectAddedLabels, fc.IssueLabelsAdded) {
				t.Errorf("For case [%s], expect added labels: \n%v\nbut got: \n%v\n",
					tc.name, tc.expectAddedLabels, fc.IssueLabelsAdded)
			}
		}

		if tc.expectRemovedLabels != nil {
			sort.Strings(tc.expectRemovedLabels)
			sort.Strings(fc.IssueLabelsRemoved)
			if !reflect.DeepEqual(tc.expectRemovedLabels, fc.IssueLabelsRemoved) {
				t.Errorf("For case [%s], expect removed labels: \n%v\nbut got: \n%v\n",
					tc.name, tc.expectRemovedLabels, fc.IssueLabelsRemoved)
			}
		}

		if len(tc.expectCreatedStatusState) != 0 {
			createdStatuses, ok := fc.CreatedStatuses["sha"]
			if !ok || len(createdStatuses) != 1 {
				t.Errorf("For case [%s], expect created status: %s, but got: none.\n",
					tc.name, tc.expectCreatedStatusState)
			} else if tc.expectCreatedStatusState != createdStatuses[0].State {
				t.Errorf("For case [%s], expect status state: %s, but got: %s.\n",
					tc.name, tc.expectCreatedStatusState, createdStatuses[0].State)
			}
		}
	}
}

func TestHandleIssueCommentEvent(t *testing.T) {
	var testcases = []struct {
		name         string
		config       externalplugins.TiCommunityIssueTriage
		action       github.IssueCommentEventAction
		comment      string
		labels       []string
		targetBranch string
		body         string
		draft        bool
		state        string
		issues       map[int]*github.Issue
		statusState  string

		expectAddedLabels        []string
		expectRemovedLabels      []string
		expectCreatedStatusState string
	}{
		{
			name:         "comment to a pull request with empty body",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			body:         "",
			state:        "open",
			targetBranch: "master",

			expectAddedLabels:        []string{},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "comment to a pull request linked to a feature issue",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{
							Name: "type/feature",
						},
					},
				},
			},

			expectAddedLabels:        []string{},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "comment to a pull request linked to a bug issue without severity label",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{
							Name: "type/bug",
						},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "comment to a pull request linked to a bug issue with severity/moderate label",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/moderate"},
					},
				},
			},

			expectAddedLabels:        []string{},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "comment to a pull request linked to a bug issue with severity/major label",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "comment to a pull request linked to a bug issue with severity/major label and may-affects/* label",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "may-affects/5.1"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "comment to a pull request linked to a bug issue with severity/major label and affects/* label",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.1"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:needs-cherry-pick-release-5.1",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
		{
			name:         "comment to a pull request linked to a bug issue with severity/major label and *-affects/* label",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.1"},
						{Name: "may-affects/5.2"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "comment to a pull request linked to a non-triaged bug issue and a feature issue",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2, ref #3",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "may-affects/5.2"},
					},
				},
				3: {
					Number: 3,
					Labels: []github.Label{
						{Name: "type/feature"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "comment to a pull request linked to a triaged issue and a non-triaged issue",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2, ref #3",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.2"},
					},
				},
				3: {
					Number: 3,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "may-affects/5.2"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:do-not-merge/needs-triage-completed",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusPending,
		},
		{
			name:         "comment to a pull request linked to two triaged issue",
			action:       github.IssueCommentActionCreated,
			comment:      "/run-check-issue-triage-complete",
			labels:       []string{},
			body:         "Issue Number: close #2, ref #3",
			state:        "open",
			targetBranch: "master",
			issues: map[int]*github.Issue{
				2: {
					Number: 2,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.2"},
					},
				},
				3: {
					Number: 3,
					Labels: []github.Label{
						{Name: "type/bug"},
						{Name: "severity/major"},
						{Name: "affects/5.3"},
					},
				},
			},

			expectAddedLabels: []string{
				"org/repo#1:needs-cherry-pick-release-5.2",
				"org/repo#1:needs-cherry-pick-release-5.3",
			},
			expectRemovedLabels:      []string{},
			expectCreatedStatusState: github.StatusSuccess,
		},
	}

	for _, testcase := range testcases {
		tc := testcase

		// Prepare issue labels data.
		prefix := "org/repo#1:"
		labels := make([]github.Label, 0)
		labelsWithPrefix := make([]string, 0)
		for _, label := range tc.labels {
			labels = append(labels, github.Label{
				Name: label,
			})
			labelsWithPrefix = append(labelsWithPrefix, prefix+label)
		}

		// Prepare plugin config.
		cfg := &externalplugins.Configuration{}
		cfg.TiCommunityIssueTriage = []externalplugins.TiCommunityIssueTriage{
			{
				Repos:                     []string{"org/repo"},
				MaintainVersions:          []string{"5.1", "5.2", "5.3"},
				AffectsLabelPrefix:        "affects/",
				MayAffectsLabelPrefix:     "may-affects/",
				NeedTriagedLabel:          "do-not-merge/needs-triage-completed",
				NeedCherryPickLabelPrefix: "needs-cherry-pick-release-",
			},
		}
		ca := &externalplugins.ConfigAgent{}
		ca.Set(cfg)

		// Mock GitHub client and webhook event.
		fc := &fghc{
			Issues:              tc.issues,
			IssueLabelsAdded:    []string{},
			IssueLabelsRemoved:  []string{},
			IssueLabelsExisting: labelsWithPrefix,
			PullRequests: map[int]*github.PullRequest{
				1: {
					Number: 1,
					State:  tc.state,
					Draft:  tc.draft,
					Body:   tc.body,
					Head: github.PullRequestBranch{
						SHA: "sha",
					},
					Base: github.PullRequestBranch{
						Ref: tc.targetBranch,
					},
					Labels: labels,
				},
			},
			CombinedStatuses: map[string]*github.CombinedStatus{
				"sha": {
					SHA: "sha",
					Statuses: []github.Status{
						{
							State:       tc.statusState,
							Description: "...",
							Context:     issueNeedTriagedContextName,
						},
					},
				},
			},
		}

		ice := &github.IssueCommentEvent{
			Action: tc.action,
			Comment: github.IssueComment{
				Body: tc.comment,
			},
			Issue: github.Issue{
				Number:      1,
				State:       tc.state,
				Body:        tc.body,
				Labels:      labels,
				PullRequest: &struct{}{},
			},
			Repo: github.Repo{
				Owner: github.User{
					Login: "org",
				},
				Name:          "repo",
				DefaultBranch: "master",
			},
		}

		getSecret := func() []byte {
			return []byte("sha=abcdefg")
		}

		getGithubToken := func() []byte {
			return []byte("token")
		}

		// Mock Server.
		s := &Server{
			ConfigAgent:            ca,
			GitHubClient:           fc,
			WebhookSecretGenerator: getSecret,
			GitHubTokenGenerator:   getGithubToken,
			Log:                    logrus.StandardLogger().WithField("client", "issue-triage"),
		}

		err := s.handleIssueCommentEvent(ice, logrus.WithField("plugin", PluginName))
		if err != nil {
			t.Errorf("For case [%s], didn't expect error: %v", tc.name, err)
		}

		if tc.expectAddedLabels != nil {
			sort.Strings(tc.expectAddedLabels)
			sort.Strings(fc.IssueLabelsAdded)
			if !reflect.DeepEqual(tc.expectAddedLabels, fc.IssueLabelsAdded) {
				t.Errorf("For case [%s], expect added labels: \n%v\nbut got: \n%v\n",
					tc.name, tc.expectAddedLabels, fc.IssueLabelsAdded)
			}
		}

		if tc.expectRemovedLabels != nil {
			sort.Strings(tc.expectRemovedLabels)
			sort.Strings(fc.IssueLabelsRemoved)
			if !reflect.DeepEqual(tc.expectRemovedLabels, fc.IssueLabelsRemoved) {
				t.Errorf("For case [%s], expect removed labels: \n%v\nbut got: \n%v\n",
					tc.name, tc.expectRemovedLabels, fc.IssueLabelsRemoved)
			}
		}

		if len(tc.expectCreatedStatusState) != 0 {
			createdStatuses, ok := fc.CreatedStatuses["sha"]
			if !ok || len(createdStatuses) != 1 {
				t.Errorf("For case [%s], expect created status: %s, but got: none.\n",
					tc.name, tc.expectCreatedStatusState)
			} else if tc.expectCreatedStatusState != createdStatuses[0].State {
				t.Errorf("For case [%s], expect status state: %s, but got: %s.\n",
					tc.name, tc.expectCreatedStatusState, createdStatuses[0].State)
			}
		}
	}
}
