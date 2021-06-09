package autoresponder

import (
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

const PluginName = "ti-community-autoresponder"

type githubClient interface {
	CreateComment(owner, repo string, number int, comment string) error
}

// reviewCtx contains information about each comment event.
type reviewCtx struct {
	repo                  github.Repo
	author, body, htmlURL string
	number                int
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.AutoresponderFor(repo.Org, repo.Repo)
			var isConfigured bool
			var configInfoStrings []string

			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")

			if len(opts.AutoResponds) != 0 {
				isConfigured = true
			}

			for _, respond := range opts.AutoResponds {
				configInfoStrings = append(configInfoStrings, "<li>"+respond.Regex+":"+respond.Message+"</li>")
			}

			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}
		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityAutoresponder: []tiexternalplugins.TiCommunityAutoresponder{
				{
					Repos: []string{"ti-community-infra/test-dev"},
					AutoResponds: []tiexternalplugins.AutoRespond{
						{
							Regex:   "(?mi)^/ping\\s*$",
							Message: "pong",
						},
					},
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The ti-community-autoresponder will trigger an automatic reply when the comment matches a regex.",
			Config:      configInfo,
			Snippet:     yamlSnippet,
			Events: []string{
				tiexternalplugins.PullRequestEvent,
				tiexternalplugins.PullRequestReviewEvent,
				tiexternalplugins.PullRequestReviewCommentEvent,
				tiexternalplugins.IssuesEvent,
				tiexternalplugins.IssueCommentEvent,
			},
		}

		return pluginHelp, nil
	}
}

// HandleIssueCommentEvent handles a GitHub issue comment event and auto respond it.
func HandleIssueCommentEvent(gc githubClient, ice *github.IssueCommentEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	// Only consider open issues or PRs and new comments.
	if ice.Issue.State != "open" || ice.Action != github.IssueCommentActionCreated {
		return nil
	}

	rc := reviewCtx{
		repo:    ice.Repo,
		author:  ice.Comment.User.Login,
		body:    ice.Comment.Body,
		htmlURL: ice.Comment.HTMLURL,
		number:  ice.Issue.Number,
	}
	// Use common handler to do the rest.
	return handle(cfg, rc, gc, log)
}

// HandlePullReviewCommentEvent handles a GitHub pull request review comment event and auto respond it.
func HandlePullReviewCommentEvent(gc githubClient, pullReviewCommentEvent *github.ReviewCommentEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	// Only consider open PRs and new comments.
	if pullReviewCommentEvent.PullRequest.State != "open" ||
		pullReviewCommentEvent.Action != github.ReviewCommentActionCreated {
		return nil
	}

	rc := reviewCtx{
		author:  pullReviewCommentEvent.Comment.User.Login,
		body:    pullReviewCommentEvent.Comment.Body,
		htmlURL: pullReviewCommentEvent.Comment.HTMLURL,
		repo:    pullReviewCommentEvent.Repo,
		number:  pullReviewCommentEvent.PullRequest.Number,
	}

	// Use common handler to do the rest.
	return handle(cfg, rc, gc, log)
}

// HandlePullReviewEvent handles a GitHub pull request review event and auto respond it.
func HandlePullReviewEvent(gc githubClient, pullReviewEvent *github.ReviewEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	// Only consider open PRs and submit actions.
	if pullReviewEvent.PullRequest.State != "open" || pullReviewEvent.Action != github.ReviewActionSubmitted {
		return nil
	}

	rc := reviewCtx{
		repo:    pullReviewEvent.Repo,
		author:  pullReviewEvent.Review.User.Login,
		body:    pullReviewEvent.Review.Body,
		htmlURL: pullReviewEvent.Review.HTMLURL,
		number:  pullReviewEvent.PullRequest.Number,
	}

	// Use common handler to do the rest.
	return handle(cfg, rc, gc, log)
}

// HandlePullRequestEvent handles a GitHub pull request event and auto respond it.
func HandlePullRequestEvent(gc githubClient, pullRequestEvent *github.PullRequestEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	// Only consider open PRs and opened/edited actions.
	if pullRequestEvent.PullRequest.State != "open" ||
		(pullRequestEvent.Action != github.PullRequestActionOpened &&
			pullRequestEvent.Action != github.PullRequestActionEdited) {
		return nil
	}

	rc := reviewCtx{
		repo:    pullRequestEvent.Repo,
		author:  pullRequestEvent.PullRequest.User.Login,
		body:    pullRequestEvent.PullRequest.Body,
		htmlURL: pullRequestEvent.PullRequest.HTMLURL,
		number:  pullRequestEvent.PullRequest.Number,
	}

	// Use common handler to do the rest.
	return handle(cfg, rc, gc, log)
}

// HandleIssueEvent handles a GitHub issue event and auto respond it.
func HandleIssueEvent(gc githubClient, issueEvent *github.IssueEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	// Only consider open issues and opened/edited actions.
	if issueEvent.Issue.State != "open" ||
		(issueEvent.Action != github.IssueActionOpened &&
			issueEvent.Action != github.IssueActionEdited) {
		return nil
	}

	rc := reviewCtx{
		repo:    issueEvent.Repo,
		author:  issueEvent.Issue.User.Login,
		body:    issueEvent.Issue.Body,
		htmlURL: issueEvent.Issue.HTMLURL,
		number:  issueEvent.Issue.Number,
	}

	// Use common handler to do the rest.
	return handle(cfg, rc, gc, log)
}

func handle(cfg *tiexternalplugins.Configuration, rc reviewCtx, gc githubClient, log *logrus.Entry) error {
	owner := rc.repo.Owner.Login
	repo := rc.repo.Name
	body := rc.body
	autoResponder := cfg.AutoresponderFor(owner, repo)

	for _, autoRespond := range autoResponder.AutoResponds {
		regex := regexp.MustCompile(autoRespond.Regex)
		if regex.MatchString(body) {
			resp := autoRespond.Message
			log.Infof("Commenting \"%s\".", resp)
			err := gc.CreateComment(owner, repo, rc.number, tiexternalplugins.FormatSimpleResponse(rc.author, resp))
			// When we got an err direly return.
			if err != nil {
				return err
			}
		}
	}

	return nil
}
