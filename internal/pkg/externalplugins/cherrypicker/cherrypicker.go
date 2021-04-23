/*
Copyright 2017 The Kubernetes Authors.
Copyright 2021 The TiChi Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

The original file of the code is at:
https://github.com/kubernetes/test-infra/blob/master/prow/external-plugins/cherrypicker/server.go,
which we modified to add support for copying the labels and reviewers.
*/
package cherrypicker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"
	"k8s.io/utils/exec"
)

const PluginName = "ti-community-cherrypicker"

var (
	cherryPickRe        = regexp.MustCompile(`(?m)^(?:/cherrypick|/cherry-pick)\s+(.+)$`)
	cherryPickBranchFmt = "cherry-pick-%d-to-%s"
)

type githubClient interface {
	AddLabels(org, repo string, number int, labels ...string) error
	AssignIssue(org, repo string, number int, logins []string) error
	RequestReview(org, repo string, number int, logins []string) error
	CreateComment(org, repo string, number int, comment string) error
	CreateFork(org, repo string) (string, error)
	CreatePullRequest(org, repo, title, body, head, base string, canModify bool) (int, error)
	CreateIssue(org, repo, title, body string, milestone int, labels, assignees []string) (int, error)
	EnsureFork(forkingUser, org, repo string) (string, error)
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetPullRequestPatch(org, repo string, number int) ([]byte, error)
	GetPullRequests(org, repo string) ([]github.PullRequest, error)
	GetRepo(owner, name string) (github.FullRepo, error)
	IsMember(org, user string) (bool, error)
	ListIssueComments(org, repo string, number int) ([]github.IssueComment, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	ListOrgMembers(org, role string) ([]github.TeamMember, error)
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.CherrypickerFor(repo.Org, repo.Repo)
			var configInfoStrings []string

			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")

			if len(opts.LabelPrefix) != 0 {
				configInfoStrings = append(configInfoStrings, "<li>The current label prefix for cherrypicker is: "+
					opts.LabelPrefix+"</li>")
			}

			if len(opts.PickedLabelPrefix) != 0 {
				configInfoStrings = append(configInfoStrings, "<li>The current picked label prefix for cherrypicker is: "+
					opts.PickedLabelPrefix+"</li>")
			}

			if opts.AllowAll {
				configInfoStrings = append(configInfoStrings, "<li>For this repository, cherry picking is available to all.</li>")
			} else {
				configInfoStrings = append(configInfoStrings, "<li>For this repository, "+
					"only Org members are allowed to do cherry picking.</li>")
			}

			if opts.IssueOnConflict {
				configInfoStrings = append(configInfoStrings, "<li>When a cherry picking PR conflicts, "+
					"an issue will be created to track it.</li>")
			}

			configInfoStrings = append(configInfoStrings, "</ul>")
			configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
		}

		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityCherrypicker: []tiexternalplugins.TiCommunityCherrypicker{
				{
					Repos:             []string{"ti-community-infra/test-dev"},
					LabelPrefix:       "cherrypick/",
					PickedLabelPrefix: "type/cherrypick-for-",
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The cherrypicker plugin is used for cherrypicking PRs across branches. " +
				"For every successful cherrypick invocation a new PR is opened " +
				"against the target branch and assigned to the requestor. ",
			Config:  configInfo,
			Snippet: yamlSnippet,
			Events:  []string{tiexternalplugins.PullRequestEvent, tiexternalplugins.IssueCommentEvent},
		}

		pluginHelp.AddCommand(pluginhelp.Command{
			Usage: "/cherrypick [branch]",
			Description: "Cherrypick a PR to a different branch. " +
				"This command works both in merged PRs (the cherrypick PR is opened immediately) " +
				"and open PRs (the cherrypick PR opens as soon as the original PR merges).",
			Featured:  true,
			WhoCanUse: "Members of the trusted organization for the repo or anyone(depends on the AllowAll configuration).",
			Examples:  []string{"/cherrypick release-3.9", "/cherry-pick release-1.15"},
		})

		return pluginHelp, nil
	}
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	TokenGenerator func() []byte
	BotUser        *github.UserData
	Email          string

	GitClient git.ClientFactory
	// Used for unit testing
	Push         func(forkName, newBranch string, force bool) error
	GitHubClient githubClient
	Log          *logrus.Entry
	ConfigAgent  *tiexternalplugins.ConfigAgent

	Bare     *http.Client
	PatchURL string

	repoLock sync.Mutex
	Repos    []github.Repo

	mapLock sync.Mutex
	lockMap map[cherryPickRequest]*sync.Mutex
}

type cherryPickRequest struct {
	org          string
	repo         string
	pr           int
	targetBranch string
}

// ServeHTTP validates an incoming webhook and puts it into the event channel.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.TokenGenerator)
	if !ok {
		return
	}
	fmt.Fprint(w, "Event received. Have a nice day.")

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := logrus.WithFields(logrus.Fields{
		"event-type":     eventType,
		github.EventGUID: eventGUID,
	})
	switch eventType {
	case "issue_comment":
		var ic github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ic); err != nil {
			return err
		}
		go func() {
			if err := s.handleIssueComment(l, ic); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Info("Cherry-pick failed.")
			}
		}()
	case "pull_request":
		var pr github.PullRequestEvent
		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		go func() {
			if err := s.handlePullRequest(l, pr); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Info("Cherry-pick failed.")
			}
		}()
	default:
		logrus.Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handleIssueComment(l *logrus.Entry, ic github.IssueCommentEvent) error {
	// Only consider new comments in PRs.
	if !ic.Issue.IsPullRequest() || ic.Action != github.IssueCommentActionCreated {
		return nil
	}

	org := ic.Repo.Owner.Login
	repo := ic.Repo.Name
	num := ic.Issue.Number
	commentAuthor := ic.Comment.User.Login
	opts := s.ConfigAgent.Config().CherrypickerFor(org, repo)

	// Do not create a new logger, its fields are re-used by the caller in case of errors.
	*l = *l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   num,
	})

	cherryPickMatches := cherryPickRe.FindAllStringSubmatch(ic.Comment.Body, -1)
	if len(cherryPickMatches) == 0 || len(cherryPickMatches[0]) != 2 {
		return nil
	}
	targetBranch := strings.TrimSpace(cherryPickMatches[0][1])

	if ic.Issue.State != "closed" {
		if !opts.AllowAll {
			// Only members should be able to do cherry-picks.
			ok, err := s.GitHubClient.IsMember(org, commentAuthor)
			if err != nil {
				return err
			}
			if !ok {
				resp := fmt.Sprintf("only [%s](https://github.com/orgs/%s/people) org members may request cherry-picks. "+
					"You can still do the cherry-pick manually.", org, org)
				l.Info(resp)
				return s.GitHubClient.CreateComment(org, repo, num, tiexternalplugins.FormatICResponse(ic.Comment, resp))
			}
		}
		resp := fmt.Sprintf("once the present PR merges, "+
			"I will cherry-pick it on top of %s in a new PR and assign it to you.", targetBranch)
		l.Info(resp)
		return s.GitHubClient.CreateComment(org, repo, num, tiexternalplugins.FormatICResponse(ic.Comment, resp))
	}

	pr, err := s.GitHubClient.GetPullRequest(org, repo, num)
	if err != nil {
		return fmt.Errorf("failed to get pull request %s/%s#%d: %w", org, repo, num, err)
	}
	baseBranch := pr.Base.Ref

	// Cherry-pick only merged PRs.
	if !pr.Merged {
		resp := "cannot cherry-pick an unmerged PR."
		l.Info(resp)
		return s.GitHubClient.CreateComment(org, repo, num, tiexternalplugins.FormatICResponse(ic.Comment, resp))
	}

	if baseBranch == targetBranch {
		resp := fmt.Sprintf("base branch (%s) needs to differ from target branch (%s).", baseBranch, targetBranch)
		l.Info(resp)
		return s.GitHubClient.CreateComment(org, repo, num, tiexternalplugins.FormatICResponse(ic.Comment, resp))
	}

	if !opts.AllowAll {
		// Only org members should be able to do cherry-picks.
		ok, err := s.GitHubClient.IsMember(org, commentAuthor)
		if err != nil {
			return err
		}
		if !ok {
			resp := fmt.Sprintf("only [%s](https://github.com/orgs/%s/people) org members may request cherry picks. "+
				"You can still do the cherry-pick manually.", org, org)
			l.Info(resp)
			return s.GitHubClient.CreateComment(org, repo, num, tiexternalplugins.FormatICResponse(ic.Comment, resp))
		}
	}

	*l = *l.WithFields(logrus.Fields{
		"requestor":     ic.Comment.User.Login,
		"target_branch": targetBranch,
	})
	l.Debug("Cherrypick request.")
	return s.handle(l, ic.Comment.User.Login, &ic.Comment, org, repo, targetBranch, pr)
}

func (s *Server) handlePullRequest(l *logrus.Entry, pre github.PullRequestEvent) error {
	// Only consider newly merged PRs.
	if pre.Action != github.PullRequestActionClosed && pre.Action != github.PullRequestActionLabeled {
		return nil
	}

	pr := pre.PullRequest
	if !pr.Merged || pr.MergeSHA == nil {
		return nil
	}

	org := pr.Base.Repo.Owner.Login
	repo := pr.Base.Repo.Name
	baseBranch := pr.Base.Ref
	num := pr.Number
	opts := s.ConfigAgent.Config().CherrypickerFor(org, repo)

	// Do not create a new logger, its fields are re-used by the caller in case of errors.
	*l = *l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   num,
	})

	comments, err := s.GitHubClient.ListIssueComments(org, repo, num)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	// requestor -> target branch -> issue comment.
	requestorToComments := make(map[string]map[string]*github.IssueComment)

	// First look for our special comments.
	for i := range comments {
		c := comments[i]
		cherryPickMatches := cherryPickRe.FindAllStringSubmatch(c.Body, -1)
		for _, match := range cherryPickMatches {
			targetBranch := strings.TrimSpace(match[1])
			if requestorToComments[c.User.Login] == nil {
				requestorToComments[c.User.Login] = make(map[string]*github.IssueComment)
			}
			requestorToComments[c.User.Login][targetBranch] = &c
		}
	}

	foundCherryPickComments := len(requestorToComments) != 0

	// Now look for our special labels.
	labels, err := s.GitHubClient.GetIssueLabels(org, repo, num)
	if err != nil {
		return fmt.Errorf("failed to get issue labels: %w", err)
	}

	// NOTICE: This will set the requestor to the author of the PR.
	if requestorToComments[pr.User.Login] == nil {
		requestorToComments[pr.User.Login] = make(map[string]*github.IssueComment)
	}

	foundCherryPickLabels := false
	for _, label := range labels {
		if strings.HasPrefix(label.Name, opts.LabelPrefix) {
			// leave this nil which indicates a label-initiated cherry-pick.
			requestorToComments[pr.User.Login][label.Name[len(opts.LabelPrefix):]] = nil
			foundCherryPickLabels = true
		}
	}

	// No need to cherry pick.
	if !foundCherryPickComments && !foundCherryPickLabels {
		return nil
	}

	if !foundCherryPickLabels && pre.Action == github.PullRequestActionLabeled {
		return nil
	}

	// Figure out membership.
	if !opts.AllowAll {
		members, err := s.GitHubClient.ListOrgMembers(org, "all")
		if err != nil {
			return err
		}
		for requestor := range requestorToComments {
			isMember := false
			for _, m := range members {
				if requestor == m.Login {
					isMember = true
					break
				}
			}
			if !isMember {
				delete(requestorToComments, requestor)
			}
		}
	}

	// Handle multiple comments serially. Make sure to filter out
	// comments targeting the same branch.
	handledBranches := make(map[string]bool)
	var errs []error
	for requestor, branches := range requestorToComments {
		for targetBranch, ic := range branches {
			if handledBranches[targetBranch] {
				// Branch already handled. Skip.
				continue
			}
			if targetBranch == baseBranch {
				resp := fmt.Sprintf("base branch (%s) needs to differ from target branch (%s).", baseBranch, targetBranch)
				l.Info(resp)
				if err := s.createComment(l, org, repo, num, ic, resp); err != nil {
					l.WithError(err).WithField("response", resp).Error("Failed to create comment.")
				}
				continue
			}
			handledBranches[targetBranch] = true
			l := l.WithFields(logrus.Fields{
				"requestor":     requestor,
				"target_branch": targetBranch,
			})
			l.Debug("Cherrypick request.")
			err := s.handle(l, requestor, ic, org, repo, targetBranch, &pr)
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to create cherrypick: %w", err))
			}
		}
	}
	return utilerrors.NewAggregate(errs)
}

//nolint:gocyclo
// TODO: refactoring to reduce complexity.
func (s *Server) handle(logger *logrus.Entry, requestor string,
	comment *github.IssueComment, org, repo, targetBranch string, pr *github.PullRequest) error {
	num := pr.Number
	title := pr.Title
	body := pr.Body
	var lock *sync.Mutex
	func() {
		s.mapLock.Lock()
		defer s.mapLock.Unlock()
		if _, ok := s.lockMap[cherryPickRequest{org, repo, num, targetBranch}]; !ok {
			if s.lockMap == nil {
				s.lockMap = map[cherryPickRequest]*sync.Mutex{}
			}
			s.lockMap[cherryPickRequest{org, repo, num, targetBranch}] = &sync.Mutex{}
		}
		lock = s.lockMap[cherryPickRequest{org, repo, num, targetBranch}]
	}()
	lock.Lock()
	defer lock.Unlock()

	opts := s.ConfigAgent.Config().CherrypickerFor(org, repo)

	forkName, err := s.ensureForkExists(org, repo)
	if err != nil {
		logger.WithError(err).Warn("failed to ensure fork exists")
		resp := fmt.Sprintf("cannot fork %s/%s: %v.", org, repo, err)
		return s.createComment(logger, org, repo, num, comment, resp)
	}

	// Clone the repo, checkout the target branch.
	startClone := time.Now()
	r, err := s.GitClient.ClientFor(org, repo)
	if err != nil {
		return fmt.Errorf("failed to get git client for %s/%s: %w", org, forkName, err)
	}
	defer func() {
		if err := r.Clean(); err != nil {
			logger.WithError(err).Error("Error cleaning up repo.")
		}
	}()
	if err := r.Checkout(targetBranch); err != nil {
		logger.WithError(err).Warn("failed to checkout target branch")
		resp := fmt.Sprintf("cannot checkout `%s`: %v.", targetBranch, err)
		return s.createComment(logger, org, repo, num, comment, resp)
	}
	logger.WithField("duration", time.Since(startClone)).Info("Cloned and checked out target branch.")

	// Fetch the patch from GitHub
	localPath, err := s.getPatch(org, repo, targetBranch, num)
	if err != nil {
		return fmt.Errorf("failed to get patch: %w", err)
	}

	// Setup git name and email.
	if err := r.Config("user.name", s.BotUser.Login); err != nil {
		return fmt.Errorf("failed to configure git user: %w", err)
	}
	email := s.Email
	if email == "" {
		email = s.BotUser.Email
	}
	if err := r.Config("user.email", email); err != nil {
		return fmt.Errorf("failed to configure git Email: %w", err)
	}

	// New branch for the cherry-pick.
	newBranch := fmt.Sprintf(cherryPickBranchFmt, num, targetBranch)

	// Check if that branch already exists, which means there is already a PR for that cherry-pick.
	if r.BranchExists(newBranch) {
		// Find the PR and link to it.
		prs, err := s.GitHubClient.GetPullRequests(org, repo)
		if err != nil {
			return fmt.Errorf("failed to get pullrequests for %s/%s: %w", org, repo, err)
		}
		for _, pr := range prs {
			if pr.Head.Ref == fmt.Sprintf("%s:%s", s.BotUser.Login, newBranch) {
				logger.WithField("preexisting_cherrypick", pr.HTMLURL).Info("PR already has cherrypick")
				resp := fmt.Sprintf("Looks like #%d has already been cherry picked in %s.", num, pr.HTMLURL)
				return s.createComment(logger, org, repo, num, comment, resp)
			}
		}
	}

	// Create the branch for the cherry-pick.
	if err := r.CheckoutNewBranch(newBranch); err != nil {
		return fmt.Errorf("failed to checkout %s: %w", newBranch, err)
	}

	// Title for GitHub issue/PR.
	title = fmt.Sprintf("%s (#%d)", title, num)

	// Apply the patch.
	ex := exec.New()
	dir := r.Directory()
	am := ex.Command("git", "am", "--3way", localPath)
	am.SetDir(dir)

	// Try git am --3way localPath.
	if out, err := am.CombinedOutput(); err != nil {
		var errs []error
		logger.WithError(err).Warnf("failed to apply PR on top of target branch and the output look like: %s", out)
		if opts.IssueOnConflict {
			resp := fmt.Sprintf("Manual cherrypick required.\n\nFailed to apply #%d on top of branch %q:\n```\n%v\n```",
				num, targetBranch, err)
			if err := s.createIssue(logger, org, repo, title, resp, num, comment, nil, []string{requestor}); err != nil {
				errs = append(errs, fmt.Errorf("failed to create issue: %w", err))
			}
		} else {
			// Try git add *.
			add := ex.Command("git", "add", "*")
			add.SetDir(dir)
			out, err := add.CombinedOutput()
			if err != nil {
				logger.WithError(err).Warnf("failed to git add conflicting files and the output look like: %s", out)
				errs = append(errs, fmt.Errorf("failed to git add conflicting files: %w", err))
			}

			//  Try git am --continue.
			amContinue := ex.Command("git", "am", "--continue")
			amContinue.SetDir(dir)
			out, err = amContinue.CombinedOutput()
			if err != nil {
				logger.WithError(err).Warnf("failed to continue git am and the output look like: %s", out)
				errs = append(errs, fmt.Errorf("failed to continue git am: %w", err))
			}
		}

		if utilerrors.NewAggregate(errs) != nil {
			resp := fmt.Sprintf("Failed to apply #%d on top of branch %q:\n```\n%v\n```",
				num, targetBranch, utilerrors.NewAggregate(errs).Error())
			if err := s.createComment(logger, org, repo, num, comment, resp); err != nil {
				errs = append(errs, fmt.Errorf("failed to create comment: %w", err))
			}
			return utilerrors.NewAggregate(errs)
		}
	}

	push := r.PushToNamedFork
	if s.Push != nil {
		push = s.Push
	}

	// Push the new branch in the bot's fork.
	if err := push(forkName, newBranch, true); err != nil {
		logger.WithError(err).Warn("failed to Push chery-picked changes to GitHub")
		resp := fmt.Sprintf("failed to Push cherry-picked changes in GitHub: %v.", err)
		return utilerrors.NewAggregate([]error{err, s.createComment(logger, org, repo, num, comment, resp)})
	}

	// Open a PR in GitHub.
	cherryPickBody := createCherrypickBody(num, body)
	head := fmt.Sprintf("%s:%s", s.BotUser.Login, newBranch)
	createdNum, err := s.GitHubClient.CreatePullRequest(org, repo, title, cherryPickBody, head, targetBranch, true)
	if err != nil {
		logger.WithError(err).Warn("failed to create new pull request")
		resp := fmt.Sprintf("new pull request could not be created: %v.", err)
		return utilerrors.NewAggregate([]error{err, s.createComment(logger, org, repo, num, comment, resp)})
	}
	*logger = *logger.WithField("new_pull_request_number", createdNum)
	resp := fmt.Sprintf("new pull request created: #%d.", createdNum)
	logger.Info("new pull request created")
	if err := s.createComment(logger, org, repo, num, comment, resp); err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	// Copying original pull request labels.
	excludeLabelsSet := sets.NewString(opts.ExcludeLabels...)
	labels := sets.NewString()
	for _, label := range pr.Labels {
		if !excludeLabelsSet.Has(label.Name) && !strings.HasPrefix(label.Name, opts.LabelPrefix) {
			labels.Insert(label.Name)
		}
	}

	// Add picked label.
	if len(opts.PickedLabelPrefix) > 0 {
		pickedLabel := opts.PickedLabelPrefix + targetBranch
		labels.Insert(pickedLabel)
	}

	if err := s.GitHubClient.AddLabels(org, repo, createdNum, labels.List()...); err != nil {
		logger.WithError(err).Warnf("Failed to add labels %v", labels.List())
	}

	// Copying original pull request reviewers.
	var reviewers []string
	for _, reviewer := range pr.RequestedReviewers {
		reviewers = append(reviewers, reviewer.Login)
	}
	if err := s.GitHubClient.RequestReview(org, repo, createdNum, reviewers); err != nil {
		logger.WithError(err).Warn("failed to request review to new PR")
		// Ignore returning errors on failure to request review as this is likely
		// due to users not being members of the org so that they can't be requested
		// in PRs.
		return nil
	}

	// Assign pull request to requestor.
	if err := s.GitHubClient.AssignIssue(org, repo, createdNum, []string{requestor}); err != nil {
		logger.WithError(err).Warn("failed to assign to new PR")
		// Ignore returning errors on failure to assign as this is most likely
		// due to users not being members of the org so that they can't be assigned
		// in PRs.
		return nil
	}
	return nil
}

func (s *Server) createComment(l *logrus.Entry, org, repo string,
	num int, comment *github.IssueComment, resp string) error {
	if err := func() error {
		if comment != nil {
			return s.GitHubClient.CreateComment(org, repo, num, tiexternalplugins.FormatICResponse(*comment, resp))
		}
		return s.GitHubClient.CreateComment(org, repo, num, fmt.Sprintf("In response to a cherrypick label: %s", resp))
	}(); err != nil {
		l.WithError(err).Warn("failed to create comment")
		return err
	}
	logrus.Debug("Created comment")
	return nil
}

// createIssue creates an issue on GitHub.
func (s *Server) createIssue(l *logrus.Entry, org, repo, title, body string, num int,
	comment *github.IssueComment, labels, assignees []string) error {
	issueNum, err := s.GitHubClient.CreateIssue(org, repo, title, body, 0, labels, assignees)
	if err != nil {
		return s.createComment(l, org, repo, num,
			comment, fmt.Sprintf("new issue could not be created for failed cherrypick: %v", err))
	}

	return s.createComment(l, org, repo, num, comment,
		fmt.Sprintf("new issue created for failed cherrypick: #%d", issueNum))
}

// ensureForkExists ensures a fork of org/repo exists for the bot.
func (s *Server) ensureForkExists(org, repo string) (string, error) {
	fork := s.BotUser.Login + "/" + repo

	// fork repo if it doesn't exist.
	repo, err := s.GitHubClient.EnsureFork(s.BotUser.Login, org, repo)
	if err != nil {
		return repo, err
	}

	s.repoLock.Lock()
	defer s.repoLock.Unlock()
	s.Repos = append(s.Repos, github.Repo{FullName: fork, Fork: true})
	return repo, nil
}

// getPatch gets the patch for the provided PR and creates a local
// copy of it. It returns its location in the filesystem and any
// encountered error.
func (s *Server) getPatch(org, repo, targetBranch string, num int) (string, error) {
	patch, err := s.GitHubClient.GetPullRequestPatch(org, repo, num)
	if err != nil {
		return "", err
	}
	localPath := fmt.Sprintf("/tmp/%s_%s_%d_%s.patch", org, repo, num, normalize(targetBranch))
	out, err := os.Create(localPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, bytes.NewBuffer(patch)); err != nil {
		return "", err
	}
	return localPath, nil
}

func normalize(input string) string {
	return strings.ReplaceAll(input, "/", "-")
}

// CreateCherrypickBody creates the body of a cherrypick PR
func createCherrypickBody(num int, note string) string {
	cherryPickBody := fmt.Sprintf("This is an automated cherry-pick of #%d", num)
	if len(note) != 0 {
		cherryPickBody = fmt.Sprintf("%s\n\n%s", cherryPickBody, note)
	}
	return cherryPickBody
}
