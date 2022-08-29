// Package flaky implement blocker for those people not fix special kinds issues a long time.
// block to passing PR by the way label a `do-not-merge/blackhouse` label.
package flaky

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	githubql "github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/utils"
)

const PluginName = "ti-community-flaky"

const (
	typeLabelPrefix = "type/"
	bugTypeLabel    = "type/bug"

	severityLabelPrefix   = "severity/"
	majorSeverityLabel    = "severity/major"
	criticalSeverityLabel = "severity/critical"

	maxStatusDescriptionLength = 140
)

const (
	issueNeedTriagedContextName = "check-issue-triage-complete"

	statusDescAllBugIssueTriaged     = "All linked bug issues have been triaged complete."
	statusDescNoLinkedIssue          = "No linked issue found."
	statusDescFailedToGetIssue       = "Failed to get issue %s/%s#%d."
	statusDescTypeLabelNotFound      = "Can not find any type label on issue %s/%s#%d."
	statusDescSeverityLabelNotFound  = "Can not find any severity label on bug issue %s/%s#%d."
	statusDescMayAffectsLabelBeFound = "Found may-affects label on bug issue %s/%s#%d."
)

var (
	IssueNumberLineRe   = regexp.MustCompile("(?im)^Issue Number:.+")
	checkIssueTriagedRe = regexp.MustCompile(`(?mi)^/(run-)?check-issue-triage-complete\s*$`)
)

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
	AddLabels(org, repo string, number int, labels ...string) error
	RemoveLabel(org, repo string, number int, label string) error
	CreateStatus(owner, repo, ref string, status github.Status) error
	CreateComment(owner, repo string, number int, comment string) error
	GetIssue(org, repo string, number int) (*github.Issue, error)
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error)
	BotUserChecker() (func(candidate string) bool, error)
	Query(context.Context, interface{}, map[string]interface{}) error
}

type referencePullRequestQuery struct {
	Repository queryRepository `graphql:"repository(owner: $org, name: $repo)"`
	RateLimit  rateLimit
}

type queryRepository struct {
	Issue issue `graphql:"issue(number: $issueNumber)"`
}

type issue struct {
	TimelineItems timelineItems `graphql:"timelineItems(first: 10, itemTypes: [CROSS_REFERENCED_EVENT])"`
}

type rateLimit struct {
	Cost      githubql.Int
	Remaining githubql.Int
}

type timelineItems struct {
	Nodes []timelineItemNode
}

type timelineItemNode struct {
	CrossReferencedEvent crossReferencedEvent `graphql:"... on CrossReferencedEvent"`
}

type crossReferencedEvent struct {
	Source          crossReferencedEventSource
	WillCloseTarget githubql.Boolean
}

type crossReferencedEventSource struct {
	PullRequest pullRequest `graphql:"... on PullRequest"`
}

// See: https://developer.github.com/v4/object/pullrequest/.
type pullRequest struct {
	Number     githubql.Int
	Repository repository
	State      githubql.String
	Author     struct {
		Login githubql.String
	}
	BaseRefName githubql.String
	HeadRefOid  githubql.String
	Labels      struct {
		Nodes []struct {
			Name githubql.String
		}
	} `graphql:"labels(first:20)"`
	Body githubql.String
}

type repository struct {
	Name  githubql.String
	Owner struct {
		Login githubql.String
	}
	DefaultBranchRef struct {
		Name githubql.String
	}
}

type issueCache map[string]*github.Issue

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.IssueTriageFor(repo.Org, repo.Repo)
			var configInfoStrings []string

			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")

			if len(opts.AffectsLabelPrefix) != 0 {
				configInfoStrings = append(configInfoStrings,
					fmt.Sprintf("<li>The affects label prefix is: %s</li>", opts.AffectsLabelPrefix))
			}
			if len(opts.MayAffectsLabelPrefix) != 0 {
				configInfoStrings = append(configInfoStrings,
					fmt.Sprintf("<li>The may affects label prefix is: %s</li>", opts.MayAffectsLabelPrefix))
			}
			if len(opts.NeedTriagedLabel) != 0 {
				configInfoStrings = append(configInfoStrings,
					fmt.Sprintf("<li>The need triaged label prefix is: %s</li>", opts.NeedTriagedLabel))
			}
			if len(opts.NeedCherryPickLabelPrefix) != 0 {
				configInfoStrings = append(configInfoStrings,
					fmt.Sprintf("<li>The need cherry-pick label prefix is: %s</li>", opts.NeedCherryPickLabelPrefix))
			}
			if len(opts.StatusTargetURL) != 0 {
				configInfoStrings = append(
					configInfoStrings,
					fmt.Sprintf("<li>The status details will be targeted to: <a href=\"%s\">Link</a></li>",
						opts.StatusTargetURL),
				)
			}

			if len(opts.MaintainVersions) != 0 {
				var versionConfigStrings []string
				versionConfigStrings = append(versionConfigStrings,
					"The release branches that the current repository is maintaining: ")
				versionConfigStrings = append(versionConfigStrings, "<ul>")
				for _, version := range opts.MaintainVersions {
					versionConfigStrings = append(versionConfigStrings, fmt.Sprintf("<li>%s</li>", version))
				}
				versionConfigStrings = append(versionConfigStrings, "</ul>")
				configInfoStrings = append(configInfoStrings, strings.Join(versionConfigStrings, "\n"))
			}

			if len(opts.WontfixVersions) != 0 {
				wontFixStrings := []string{"The release branches won't fix in the current repository: ", "<ul>"}
				for _, version := range opts.WontfixVersions {
					wontFixStrings = append(wontFixStrings, fmt.Sprintf("<li>%s</li>", version))
				}
				wontFixStrings = append(wontFixStrings, "</ul>")
				configInfoStrings = append(configInfoStrings, strings.Join(wontFixStrings, "\n"))
			}

			configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
		}

		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityIssueTriage: []tiexternalplugins.TiCommunityIssueTriage{
				{
					Repos:                     []string{"ti-community-infra/test-dev"},
					WontfixVersions:           []string{"5.0", "6.0"},
					MaintainVersions:          []string{"5.1", "5.2", "5.3"},
					AffectsLabelPrefix:        "affects/",
					MayAffectsLabelPrefix:     "may-affects/",
					NeedTriagedLabel:          "do-not-merge/needs-triage-completed",
					NeedCherryPickLabelPrefix: "needs-cherry-pick-release-",
					StatusTargetURL:           "https://book.prow.tidb.io/#/plugins/issue-triage",
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: fmt.Sprintf("The %s plugin", PluginName),
			Config:      configInfo,
			Snippet:     yamlSnippet,
			Commands: []pluginhelp.Command{
				{
					Usage: "/[run-]check-issue-triage-complete",
					Examples: []string{
						"/run-check-issue-triage-complete",
						"/check-issue-triage-complete",
					},
					Description: "Forces rechecking of the check-issue-triage-complete status.",
					WhoCanUse:   "Anyone",
				},
			},
			Events: []string{
				tiexternalplugins.PullRequestEvent,
			},
		}

		return pluginHelp, nil
	}
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	WebhookSecretGenerator func() []byte
	GitHubTokenGenerator   func() []byte

	GitHubClient githubClient
	ConfigAgent  *tiexternalplugins.ConfigAgent

	mapLock sync.Mutex
	lockMap map[checkRequest]*sync.Mutex
}

type checkRequest struct {
	org  string
	repo string
	pr   int
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.WebhookSecretGenerator)
	if !ok {
		return
	}

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

// handleEvent distributed events and handles them.
func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := logrus.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	switch eventType {
	case tiexternalplugins.PullRequestEvent:
		var pe github.PullRequestEvent
		if err := json.Unmarshal(payload, &pe); err != nil {
			return err
		}
		go func() {
			if err := s.handlePullRequestEvent(&pe, l); err != nil {
				l.WithError(err).Info("Error handling event.")
			}
		}()
	case tiexternalplugins.IssuesEvent:
		var ie github.IssueEvent
		if err := json.Unmarshal(payload, &ie); err != nil {
			return err
		}
		go func() {
			if err := s.handleIssueEvent(&ie, l); err != nil {
				l.WithError(err).Info("Error handling event.")
			}
		}()
	case tiexternalplugins.IssueCommentEvent:
		var ice github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ice); err != nil {
			return err
		}
		go func() {
			if err := s.handleIssueCommentEvent(&ice, l); err != nil {
				l.WithError(err).Info("Error handling event.")
			}
		}()
	default:
		l.Debugf("received an event of type %q but didn't ask for it", eventType)
	}
	return nil
}

//nolint:gocritic
func (s *Server) handleIssueEvent(ie *github.IssueEvent, log *logrus.Entry) error {
	org := ie.Repo.Owner.Login
	repo := ie.Repo.Name
	num := ie.Issue.Number
	log = log.WithFields(logrus.Fields{
		"org":  org,
		"repo": repo,
		"num":  num,
	})

	// Notice: When we check out a new release-x.y branch, the release management tool will add affects-x.y labels
	// to bug issues in batches. At this time, a large number of requests will be generated. We need to handle
	// labeled events very carefully to avoid it consumes too many API points.
	if ie.Action != github.IssueActionLabeled && ie.Action != github.IssueActionUnlabeled {
		log.Debug("Skipping because not a labeled / unlabeled action.")
		return nil
	}

	cfg := s.ConfigAgent.Config().IssueTriageFor(org, repo)
	newLabel := ie.Label.Name
	existedLabels := sets.String{}
	for _, label := range ie.Issue.Labels {
		existedLabels.Insert(label.Name)
	}

	if !existedLabels.Has(bugTypeLabel) {
		log.Debug("Skipping because not a type/bug issue.")
		return nil
	}

	if ie.Action == github.IssueActionLabeled && (newLabel == majorSeverityLabel || newLabel == criticalSeverityLabel) {
		// Add may-affects labels when major or critical severity label was added.
		labelsNeedToAdd := sets.String{}
		for _, version := range cfg.MaintainVersions {
			affectVersionLabel := cfg.AffectsLabelPrefix + version
			mayAffectVersionLabel := cfg.MayAffectsLabelPrefix + version
			if !existedLabels.HasAny(affectVersionLabel, mayAffectVersionLabel) {
				labelsNeedToAdd.Insert(mayAffectVersionLabel)
			}
		}

		if labelsNeedToAdd.Len() > 0 {
			return s.GitHubClient.AddLabels(org, repo, num, labelsNeedToAdd.List()...)
		}
	} else if ie.Action == github.IssueActionLabeled && strings.HasPrefix(newLabel, cfg.AffectsLabelPrefix) {
		// Remove the may-affects label when the affects label was added.
		version := strings.TrimPrefix(newLabel, cfg.AffectsLabelPrefix)
		mayAffectsVersionLabel := cfg.MayAffectsLabelPrefix + version
		if existedLabels.Has(mayAffectsVersionLabel) {
			return s.GitHubClient.RemoveLabel(org, repo, num, mayAffectsVersionLabel)
		}
	} else if strings.HasPrefix(newLabel, cfg.MayAffectsLabelPrefix) {
		// Rerun the issue triage complete checker when may-affects label was removed.
		prs, err := s.getReferencePRList(log, org, repo, num)
		if err != nil {
			return err
		}
		if len(prs) == 0 {
			log.Infof("Can not find any reference PRs.")
			return nil
		}
		log.Infof("Found %d reference PRs.", len(prs))

		// Notice: wait a few seconds to reduce API consumption in case of batched label operations.
		time.Sleep(5 * time.Second)
		issue, err := s.GitHubClient.GetIssue(org, repo, num)
		if err != nil {
			return err
		}

		issues := make(issueCache)
		key := fmt.Sprintf("%s/%s#%d", org, repo, num)
		issues[key] = issue

		var errs []error
		for _, pr := range prs {
			defaultBranch := string(pr.Repository.DefaultBranchRef.Name)
			prOrg := string(pr.Repository.Owner.Login)
			prRepo := string(pr.Repository.Name)
			prNum := int(pr.Number)
			prState := string(pr.State)
			prSHA := string(pr.HeadRefOid)
			prBranch := string(pr.BaseRefName)
			log = log.WithFields(logrus.Fields{
				"org":  prOrg,
				"repo": prRepo,
				"num":  prNum,
			})

			if !isPRNeedToCheck(prState, prBranch, defaultBranch) {
				log.Debugf("Skipping the check, because it is not a open PR on the default branch.")
				continue
			}
			log.Infof("Checking triggered by issue %s/%s#%d.", org, repo, num)

			prLabels := sets.NewString()
			for _, node := range pr.Labels.Nodes {
				prLabels.Insert(string(node.Name))
			}

			issueNumberLine := IssueNumberLineRe.FindString(string(pr.Body))
			linkedIssueNumbers := utils.NormalizeIssueNumbers(issueNumberLine, org, repo)
			log.Infof("Found %d linked issues.", len(linkedIssueNumbers))

			err := s.handle(log, cfg, prOrg, prRepo, prSHA, prNum, prLabels, linkedIssueNumbers, issues)
			if err != nil {
				errs = append(errs, err)
			}
		}
		return utilerrors.NewAggregate(errs)
	}

	return nil
}

func (s *Server) handlePullRequestEvent(pe *github.PullRequestEvent, log *logrus.Entry) error {
	if pe.Action != github.PullRequestActionOpened && pe.Action != github.PullRequestActionReopened &&
		pe.Action != github.PullRequestActionEdited && pe.Action != github.PullRequestActionSynchronize {
		log.Debug("Skipping because not a opened / reopened / edited / synchronize action.")
		return nil
	}

	pr := pe.PullRequest
	prOrg := pe.Repo.Owner.Login
	prRepo := pe.Repo.Name
	prSHA := pr.Head.SHA
	prNum := pr.Number
	log = log.WithFields(logrus.Fields{
		"org":  prOrg,
		"repo": prRepo,
		"num":  prNum,
	})

	if !isPRNeedToCheck(pr.State, pr.Base.Ref, pe.Repo.DefaultBranch) {
		log.Debug("Skipping the check, because it is not a open PR on the default branch.")
		return nil
	}
	log.Infof("Checking triggered by PR %s/%s#%d.", prOrg, prRepo, prNum)

	prLabels := sets.NewString()
	for _, label := range pr.Labels {
		prLabels.Insert(label.Name)
	}

	cfg := s.ConfigAgent.Config().IssueTriageFor(prOrg, prRepo)
	issueNumberLine := IssueNumberLineRe.FindString(pr.Body)
	linkedIssueNumbers := utils.NormalizeIssueNumbers(issueNumberLine, prOrg, prRepo)
	log.Infof("Found %d linked issues.", len(linkedIssueNumbers))

	return s.handle(log, cfg, prOrg, prRepo, prSHA, prNum, prLabels, linkedIssueNumbers, issueCache{})
}

func (s *Server) handleIssueCommentEvent(ice *github.IssueCommentEvent, log *logrus.Entry) error {
	// Only reacted for pull request in open state.
	if ice.Action != github.IssueCommentActionCreated || !ice.Issue.IsPullRequest() {
		return nil
	}

	issue := ice.Issue
	comment := ice.Comment.Body
	prOrg := ice.Repo.Owner.Login
	prRepo := ice.Repo.Name
	prNum := issue.Number
	log = log.WithFields(logrus.Fields{
		"org":        prOrg,
		"repo":       prRepo,
		"num":        prNum,
		"comment_id": ice.Comment.ID,
	})

	if !checkIssueTriagedRe.MatchString(comment) {
		log.Debug("Skipping the check, because no command comment.")
		return nil
	}

	pr, err := s.GitHubClient.GetPullRequest(prOrg, prRepo, prNum)
	if err != nil {
		return err
	}

	if !isPRNeedToCheck(pr.State, pr.Base.Ref, ice.Repo.DefaultBranch) {
		log.Debug("Skipping the check because it is not a open PR on the default branch.")
		return nil
	}
	log.Infof("Checking triggered by comment %s/%s#%d:%d.", prOrg, prRepo, prNum, ice.Comment.ID)

	prSHA := pr.Head.SHA
	prLabels := sets.NewString()
	for _, label := range pr.Labels {
		prLabels.Insert(label.Name)
	}

	cfg := s.ConfigAgent.Config().IssueTriageFor(prOrg, prRepo)
	issueNumberLine := IssueNumberLineRe.FindString(pr.Body)
	linkedIssueNumbers := utils.NormalizeIssueNumbers(issueNumberLine, prOrg, prRepo)
	log.Infof("Found %d linked issues.", len(linkedIssueNumbers))

	return s.handle(log, cfg, prOrg, prRepo, prSHA, prNum, prLabels, linkedIssueNumbers, issueCache{})
}

func (s *Server) handle(log *logrus.Entry, cfg *tiexternalplugins.TiCommunityIssueTriage,
	prOrg, prRepo, prSHA string, prNum int, prLabels sets.String,
	issueKeys []utils.IssueNumberData, issueCache issueCache) error {
	// TODO: need function throttling.
	var lock *sync.Mutex
	func() {
		s.mapLock.Lock()
		defer s.mapLock.Unlock()
		if _, ok := s.lockMap[checkRequest{prOrg, prRepo, prNum}]; !ok {
			if s.lockMap == nil {
				s.lockMap = map[checkRequest]*sync.Mutex{}
			}
			s.lockMap[checkRequest{prOrg, prRepo, prNum}] = &sync.Mutex{}
		}
		lock = s.lockMap[checkRequest{prOrg, prRepo, prNum}]
	}()
	lock.Lock()
	defer lock.Unlock()

	existingStatus, existingDesc, existingURL, err := s.checkExistingStatus(log, prOrg, prRepo, prSHA)
	if err != nil {
		return err
	}

	wantStatus, wantDesc, affectsVersionLabels, err := s.checkLinkedIssues(cfg, issueKeys, issueCache)
	if err != nil {
		return err
	}
	log.Infof("Checking result: %s, found affects labels: %v", strings.ToLower(wantDesc), affectsVersionLabels)

	// Notice: check pass when all the linked issues needed to be triaged have triaged complete,
	// if a PR has no linked bug issues, the check will pass too.
	if wantStatus == github.StatusSuccess {
		// Remove the need triaged label.
		if len(cfg.NeedTriagedLabel) != 0 && prLabels.Has(cfg.NeedTriagedLabel) {
			err := s.GitHubClient.RemoveLabel(prOrg, prRepo, prNum, cfg.NeedTriagedLabel)
			if err != nil {
				return err
			}
		}

		// Add needs-cherry-pick label.
		labelsNeedToAdd := make([]string, 0)
		wontfixVersionSets := sets.NewString(cfg.WontfixVersions...)
		for _, affectsVersionLabel := range affectsVersionLabels.List() {
			affectVersion := strings.TrimPrefix(affectsVersionLabel, cfg.AffectsLabelPrefix)
			if wontfixVersionSets.Has(affectVersion) {
				continue
			}

			cherryPickLabel := cfg.NeedCherryPickLabelPrefix + affectVersion
			if !prLabels.Has(cherryPickLabel) {
				labelsNeedToAdd = append(labelsNeedToAdd, cherryPickLabel)
			}
		}

		if len(labelsNeedToAdd) > 0 {
			err := s.GitHubClient.AddLabels(prOrg, prRepo, prNum, labelsNeedToAdd...)
			if err != nil {
				return err
			}
		}
	} else if wantStatus == github.StatusError || wantStatus == github.StatusFailure {
		// Add the need triaged label.
		if len(cfg.NeedTriagedLabel) != 0 && !prLabels.Has(cfg.NeedTriagedLabel) {
			err := s.GitHubClient.AddLabel(prOrg, prRepo, prNum, cfg.NeedTriagedLabel)
			if err != nil {
				return err
			}
		}
	}

	// Update status.
	wantDesc = truncatedStatusDesc(log, wantDesc)
	return s.createStatus(log, prOrg, prRepo, prSHA, existingStatus, existingDesc, existingURL,
		wantStatus, wantDesc, cfg.StatusTargetURL)
}

// isPRNeedToCheck used to determine if PR needs to be checked.
// Only open PR on the master branch need to be checked.
func isPRNeedToCheck(state, prBranch, defaultBranch string) bool {
	return strings.EqualFold(state, github.PullRequestStateOpen) && strings.EqualFold(prBranch, defaultBranch)
}

// checkLinkedIssues used to check if a given set of bug issues have all been triaged.
func (s *Server) checkLinkedIssues(cfg *tiexternalplugins.TiCommunityIssueTriage,
	issueKeys []utils.IssueNumberData, issueCache issueCache) (string, string, sets.String, error) {
	affectsVersionLabels := sets.NewString()

	if len(issueKeys) == 0 {
		return github.StatusSuccess, statusDescNoLinkedIssue, affectsVersionLabels, nil
	}

	for _, issueKey := range issueKeys {
		org := issueKey.Org
		repo := issueKey.Repo
		num := issueKey.Number

		issue, err := s.getIssueWithCache(issueCache, org, repo, num)
		if err != nil {
			desc := fmt.Sprintf(statusDescFailedToGetIssue, org, repo, num)
			return github.StatusFailure, desc, sets.String{}, err
		}

		labels := sets.NewString()
		for _, label := range issue.Labels {
			labels.Insert(label.Name)
			if strings.HasPrefix(label.Name, cfg.AffectsLabelPrefix) {
				affectsVersionLabels.Insert(label.Name)
			}
		}

		// Issue without type label is not triaged.
		if !hasAnyTypeLabel(issue) {
			desc := fmt.Sprintf(statusDescTypeLabelNotFound, org, repo, num)
			return github.StatusError, desc, sets.String{}, nil
		}

		// Only bug issues need to be checked.
		if !labels.Has(bugTypeLabel) {
			continue
		}

		// Bug issue must have severity label, if not, it will be considered to triage.
		if !hasAnySeverityLabel(issue) {
			desc := fmt.Sprintf(statusDescSeverityLabelNotFound, org, repo, num)
			return github.StatusError, desc, sets.String{}, nil
		}

		// Only major or critical bug issues need to be checked.
		if !labels.HasAny(majorSeverityLabel, criticalSeverityLabel) {
			continue
		}

		// Check if issue has any may-affects labels, if any issue have
		// not yet been triaged, the checker will fail.
		if hasAnyMayAffectsLabel(issue, cfg.MayAffectsLabelPrefix) {
			desc := fmt.Sprintf(statusDescMayAffectsLabelBeFound, org, repo, num)
			return github.StatusError, desc, sets.String{}, nil
		}
	}

	return github.StatusSuccess, statusDescAllBugIssueTriaged, affectsVersionLabels, nil
}

// checkExistingStatus will retrieve the current status of the linked issue need triaged context
// for the provided SHA.
func (s *Server) checkExistingStatus(l *logrus.Entry, org, repo, sha string) (string, string, string, error) {
	combinedStatus, err := s.GitHubClient.GetCombinedStatus(org, repo, sha)
	if err != nil {
		return "", "", "", fmt.Errorf("error listing pull request combined statuses: %w", err)
	}

	existingStatus := ""
	existingDesc := ""
	existingURL := ""
	for _, status := range combinedStatus.Statuses {
		if status.Context != issueNeedTriagedContextName {
			continue
		}
		existingStatus = status.State
		existingDesc = status.Description
		existingURL = status.TargetURL
		break
	}
	l.Debugf("Existing linked issue need triaged status context status is %q", existingStatus)
	return existingStatus, existingDesc, existingURL, nil
}

func hasAnyTypeLabel(issue *github.Issue) bool {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label.Name, typeLabelPrefix) {
			return true
		}
	}
	return false
}

func hasAnySeverityLabel(issue *github.Issue) bool {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label.Name, severityLabelPrefix) {
			return true
		}
	}
	return false
}

func hasAnyMayAffectsLabel(issue *github.Issue, mayAffectsLabelPrefix string) bool {
	for _, label := range issue.Labels {
		if strings.HasPrefix(label.Name, mayAffectsLabelPrefix) {
			return true
		}
	}
	return false
}

func (s *Server) getIssueWithCache(issues issueCache, org, repo string, num int) (*github.Issue, error) {
	key := fmt.Sprintf("%s/%s#%d", org, repo, num)
	if issue, ok := issues[key]; ok {
		return issue, nil
	}

	issue, err := s.GitHubClient.GetIssue(org, repo, num)
	if err != nil {
		return nil, err
	}
	issues[key] = issue

	return issue, nil
}

func (s *Server) getReferencePRList(log *logrus.Entry, org, repo string, issueNumber int) ([]pullRequest, error) {
	var query referencePullRequestQuery
	vars := map[string]interface{}{
		"org":         githubql.String(org),
		"repo":        githubql.String(repo),
		"issueNumber": githubql.Int(issueNumber),
	}
	ctx := context.Background()
	if err := s.GitHubClient.Query(ctx, &query, vars); err != nil {
		return nil, err
	}

	totalCost := int(query.RateLimit.Cost)
	remaining := int(query.RateLimit.Remaining)

	ret := make([]pullRequest, 0)
	for _, node := range query.Repository.Issue.TimelineItems.Nodes {
		ret = append(ret, node.CrossReferencedEvent.Source.PullRequest)
	}

	log.Infof("Get reference PR list for issue %s/%s#%d cost %d point(s), %d remaining, found %d.",
		org, repo, issueNumber, totalCost, remaining, len(ret))

	return ret, nil
}

func (s *Server) createStatus(log *logrus.Entry, org, repo, sha, existingState, existingDesc, existingURL,
	wantState, wantDesc, wantURL string) error {
	if existingState != wantState || existingDesc != wantDesc || existingURL != wantURL {
		log.Debugf("Setting check-issue-triage-complete status context to %s.", wantState)
		if err := s.GitHubClient.CreateStatus(org, repo, sha, github.Status{
			Context:     issueNeedTriagedContextName,
			State:       wantState,
			TargetURL:   wantURL,
			Description: wantDesc,
		}); err != nil {
			return fmt.Errorf("error setting pull request status: %w", err)
		}
	} else {
		log.Debugf("The check-issue-triage-complete status context is already %s.", wantState)
	}

	return nil
}

func truncatedStatusDesc(log *logrus.Entry, wantDesc string) string {
	if len(wantDesc) > maxStatusDescriptionLength {
		original := wantDesc
		wantDesc = fmt.Sprintf("%s...", wantDesc[0:(maxStatusDescriptionLength-3)])
		log.WithField("original-desc", original).Warn("GitHub status description needed to be truncated to fit GH API limit")
	}
	return wantDesc
}
