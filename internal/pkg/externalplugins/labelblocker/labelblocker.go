package labelblocker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
)

const PluginName = "ti-community-label-blocker"

const (
	LabeledAction   = "labeled"
	UnlabeledAction = "unlabeled"
)

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
	GetTeamBySlug(slug string, org string) (*github.Team, error)
	ListTeamMembers(org string, id int, role string) ([]github.TeamMember, error)
	CreateComment(org, repo string, number int, comment string) error
}

// labelCtx contains information about each label event.
type labelCtx struct {
	repo                  github.Repo
	sender, label, action string
	number                int
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
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
				configInfoStrings = append(configInfoStrings, "<h6>"+blockLabel.Regex+": </h6>")

				if len(blockLabel.TrustedTeams) != 0 {
					trustedTeamNames := strings.Join(blockLabel.TrustedTeams, ", ")
					configInfoStrings = append(configInfoStrings, "trusted team ("+trustedTeamNames+"), ")
				}

				if len(blockLabel.TrustedUsers) != 0 {
					trustedUserNames := strings.Join(blockLabel.TrustedUsers, ", ")
					configInfoStrings = append(configInfoStrings, "trusted user ("+trustedUserNames+")")
				}

				configInfoStrings = append(configInfoStrings, "</li>")
			}

			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}
		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityLabelBlocker: []tiexternalplugins.TiCommunityLabelBlocker{
				{
					Repos: []string{"ti-community-infra/test-dev"},
					BlockLabels: []tiexternalplugins.BlockLabel{
						{
							Regex:        "^status/LGT[\\d]+$",
							Actions:      []string{"labeled"},
							TrustedTeams: []string{"release-team"},
							TrustedUsers: []string{"hi-rustin"},
							Message:      "You can't add the status/can-merge label.",
						},
					},
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}
		pluginHelp := &pluginhelp.PluginHelp{
			Description: "The ti-community-label-blocker will prevent untrusted users from adding or removing labels.",
			Config:      configInfo,
			Snippet:     yamlSnippet,
			Events:      []string{tiexternalplugins.PullRequestEvent, tiexternalplugins.IssuesEvent},
		}

		return pluginHelp, nil
	}
}

// HandlePullRequestEvent handles a GitHub pull request event.
func HandlePullRequestEvent(gc githubClient, pullRequestEvent *github.PullRequestEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	// Only consider the labeled / unlabeled actions.
	if pullRequestEvent.Action != github.PullRequestActionLabeled &&
		pullRequestEvent.Action != github.PullRequestActionUnlabeled {
		return nil
	}

	ctx := labelCtx{
		repo:   pullRequestEvent.Repo,
		sender: pullRequestEvent.Sender.Login,
		label:  pullRequestEvent.Label.Name,
		action: string(pullRequestEvent.Action),
		number: pullRequestEvent.PullRequest.Number,
	}

	// Use a common handler to do the rest.
	return handle(cfg, ctx, gc, log)
}

// HandleIssueEvent handles a GitHub issue event.
func HandleIssueEvent(gc githubClient, issueEvent *github.IssueEvent,
	cfg *tiexternalplugins.Configuration, log *logrus.Entry) error {
	// Only consider the labeled / unlabeled actions.
	if issueEvent.Action != github.IssueActionLabeled &&
		issueEvent.Action != github.IssueActionUnlabeled {
		return nil
	}

	ctx := labelCtx{
		repo:   issueEvent.Repo,
		sender: issueEvent.Sender.Login,
		label:  issueEvent.Label.Name,
		action: string(issueEvent.Action),
		number: issueEvent.Issue.Number,
	}

	// Use a common handler to do the rest.
	return handle(cfg, ctx, gc, log)
}

func handle(cfg *tiexternalplugins.Configuration, ctx labelCtx, gc githubClient, log *logrus.Entry) error {
	owner := ctx.repo.Owner.Login
	repo := ctx.repo.Name
	labelBlocker := cfg.LabelBlockerFor(owner, repo)

	for _, blockLabel := range labelBlocker.BlockLabels {
		regex := regexp.MustCompile(blockLabel.Regex)

		// If this rule does not match, try to match the next rule.
		if !regex.MatchString(ctx.label) || !isMatchAction(ctx.action, blockLabel.Actions) {
			log.Infof("%s:%s does not match regex or action.", regex, ctx.action)
			continue
		}

		// If the operator is a trusted user, don’t trigger blocking.
		allTrustedUserLogins := listAllTrustedUserLogins(owner, blockLabel.TrustedTeams, blockLabel.TrustedUsers, gc, log)
		if allTrustedUserLogins.Has(ctx.sender) {
			log.Infof("Operator %s is trusted by the %s rule.", ctx.sender, blockLabel.Regex)
			continue
		}

		// Undo the illegal operation.
		if ctx.action == LabeledAction {
			// Remove the label added illegally.
			err := gc.RemoveLabel(owner, repo, ctx.number, ctx.label)

			if err == nil {
				log.Infof("Remove %s label added illegally.", ctx.label)
			} else {
				return fmt.Errorf("failed to remove illegal label added illegally, %s", err)
			}
		} else if ctx.action == UnlabeledAction {
			// Restore the label removed illegally.
			err := gc.AddLabel(owner, repo, ctx.number, ctx.label)

			if err == nil {
				log.Infof("Restore %s label removed illegally.", ctx.label)
			} else {
				return fmt.Errorf("failed to restore the illegally removed label, %s", err)
			}
		}

		// Reply to a message explaining why robot do this.
		if len(blockLabel.Message) != 0 {
			var operate string
			if ctx.action == LabeledAction {
				operate = "adding"
			} else if ctx.action == UnlabeledAction {
				operate = "removing"
			}

			reason := fmt.Sprintf("In response to %s label named %s.", operate, ctx.label)
			response := tiexternalplugins.FormatResponse(ctx.sender, blockLabel.Message, reason)
			err := gc.CreateComment(owner, repo, ctx.number, response)

			if err != nil {
				return fmt.Errorf("failed to respond message, %s", err)
			}
		}
	}

	return nil
}

// listAllTrustedUserLogins used to obtain all trusted user login names, contains the members of trusted team.
func listAllTrustedUserLogins(owner string, trustTeams, trustedUsers []string,
	gc githubClient, log *logrus.Entry) sets.String {
	trustedUserLogins := sets.String{}

	trustedUserLogins.Insert(trustedUsers...)
	log.Infof("trusted user: %s", trustedUsers)

	// Treat members of the trusted team as trusted users.
	for _, slug := range trustTeams {
		team, err := gc.GetTeamBySlug(slug, owner)

		if err == nil {
			log.Infof("Get trusted team by slug %s successfully", slug)
		} else {
			log.Errorf("Failed to get trusted team by slug %s", slug)
			continue
		}

		trustTeamMembers, err := gc.ListTeamMembers(owner, team.ID, github.RoleAll)

		if err == nil {
			log.Infof("Get the members of trusted team named %s successfully", slug)
		} else {
			log.Errorf("Failed to get the members of trusted team named %s", slug)
			continue
		}

		for _, member := range trustTeamMembers {
			trustedUserLogins.Insert(member.Login)
		}
	}

	return trustedUserLogins
}

// isMatchAction used to determine whether given action matches block action.
func isMatchAction(action string, blockActions []string) bool {
	for _, blockAction := range blockActions {
		if blockAction == action {
			return true
		}
	}

	return false
}
