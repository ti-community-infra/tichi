package labelblocker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
)

const PluginName = "ti-community-label-blocker"

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
	TeamHasMember(org string, teamID int, memberLogin string) (bool, error)
	GetTeamBySlug(slug string, org string) (*github.Team, error)
	ListTeamMembers(org string, id int, role string) ([]github.TeamMember, error)
}

// labelCtx contains information about each label event.
type labelCtx struct {
	repo          github.Repo
	author, label string
	action        string
	number        int
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *externalplugins.ConfigAgent) func(
	enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.LabelBlockerFor(repo.Org, repo.Repo)
			var isConfigured bool
			var configInfoStrings []string

			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")

			if len(opts.BlockLabels) != 0 {
				isConfigured = true
			}

			for _, blockLabel := range opts.BlockLabels {
				configInfoStrings = append(configInfoStrings, "<li>")

				trustedTeamNames := strings.Join(blockLabel.TrustedTeams, ", ")
				trustedUserNames := strings.Join(blockLabel.TrustedUsers, ", ")

				configInfoStrings = append(configInfoStrings, blockLabel.Regex+": trusted team ("+
					trustedTeamNames+") trusted user ("+trustedUserNames+")")
				configInfoStrings = append(configInfoStrings, "</li>")
			}

			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}
		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The ti-community-label-blocker will trigger an automatic reply when the comment matches a regex.",
			Config:      configInfo,
		}

		return pluginHelp, nil
	}
}

// HandleIssueEvent handles a GitHub issue event and auto respond it.
func HandlePullRequestEvent(gc githubClient, pullRequestEvent *github.PullRequestEvent,
	cfg *externalplugins.Configuration, log *logrus.Entry) error {
	// Only consider the labeled/unlabeled actions.
	if pullRequestEvent.Action != github.PullRequestActionLabeled &&
		pullRequestEvent.Action != github.PullRequestActionUnlabeled {
		return nil
	}

	ctx := labelCtx{
		repo:   pullRequestEvent.Repo,
		author: pullRequestEvent.Sender.Login,
		label:  pullRequestEvent.Label.Name,
		action: string(pullRequestEvent.Action),
		number: pullRequestEvent.PullRequest.Number,
	}

	// Use common handler to do the rest.
	return handle(cfg, ctx, gc, log)
}

func handle(cfg *externalplugins.Configuration, ctx labelCtx, gc githubClient, log *logrus.Entry) error {
	owner := ctx.repo.Owner.Login
	repo := ctx.repo.Name
	labelBlocker := cfg.LabelBlockerFor(owner, repo)

	for _, blockLabel := range labelBlocker.BlockLabels {
		regex := regexp.MustCompile(blockLabel.Regex)

		// If this rule does not match, try to match the next rule.
		if !regex.MatchString(ctx.label) || !isMatchAction(ctx.action, blockLabel.Actions) {
			continue
		}

		// If the operator is a trusted user, don’t trigger interception.
		allTrustedUserLogins := listAllTrustedUserLogins(owner, blockLabel.TrustedTeams, blockLabel.TrustedUsers, gc, log)

		trusted := false
		for _, login := range allTrustedUserLogins {
			if login == ctx.author {
				trusted = true
				break
			}
		}

		if trusted {
			continue
		}

		// Undo the illegal operation.
		if ctx.action == "labeled" {
			err := gc.RemoveLabel(owner, repo, ctx.number, ctx.label)

			if err != nil {
				return fmt.Errorf("failed to remove illegal label added manually, %s", err)
			}
		} else if ctx.action == "unlabeled" {
			err := gc.AddLabel(owner, repo, ctx.number, ctx.label)

			if err != nil {
				return fmt.Errorf("failed to restore the manually removed label, %s", err)
			}
		}
	}

	return nil
}

// listAllTrustedUserLogins used to obtain all trusted user login names, contains the members of trusted team.
func listAllTrustedUserLogins(owner string, trustTeams, trustedUsers []string,
	gc githubClient, log *logrus.Entry) []string {
	trustedUserSets := sets.String{}

	// Treat members of the trusted team as trusted users.
	for _, slug := range trustTeams {
		team, err := gc.GetTeamBySlug(slug, owner)

		if err != nil {
			log.Errorf("failed to get trusted team by slug %s", slug)
			continue
		}

		trustTeamMembers, err := gc.ListTeamMembers(owner, team.ID, "all")

		if err != nil {
			log.Errorf("failed to get trusted team members %s", slug)
			continue
		}

		for _, member := range trustTeamMembers {
			trustedUserSets.Insert(member.Login)
		}
	}

	for _, trustUserLogin := range trustedUsers {
		trustedUserSets.Insert(trustUserLogin)
	}

	return trustedUserSets.List()
}

// isMatchAction used to determine whether it matches action.
func isMatchAction(action string, blockActions []string) bool {
	for _, blockAction := range blockActions {
		if blockAction == action {
			return true
		}
	}

	return false
}
