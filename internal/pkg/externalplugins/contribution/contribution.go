package contribution

import (
	"fmt"
	"strings"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"
)

const PluginName = "ti-community-contribution"

const (
	// firstTimer means author has not previously committed to GitHub.
	firstTimer = "FIRST_TIMER"
	// firstTimeContributor means author has not previously committed to the repository.
	firstTimeContributor = "FIRST_TIME_CONTRIBUTOR"
)

type githubClient interface {
	AddLabels(org, repo string, number int, labels ...string) error
	CreateComment(owner, repo string, number int, comment string) error
	IsMember(org, user string) (bool, error)
}

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
// HelpProvider defines the type for function that construct the PluginHelp for plugins.
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.ContributionFor(repo.Org, repo.Repo)
			var isConfigured bool
			var configInfoStrings []string

			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")

			if len(opts.Message) != 0 {
				isConfigured = true
			}

			configInfoStrings = append(configInfoStrings, "<li>message: "+opts.Message+"</li>")

			configInfoStrings = append(configInfoStrings, "</ul>")
			if isConfigured {
				configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
			}
		}

		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityContribution: []tiexternalplugins.TiCommunityContribution{
				{
					Repos:   []string{"ti-community-infra/test-dev"},
					Message: "These are tips for external contributors.",
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: fmt.Sprintf("The %s plugin will add %s or %s "+
				"labels to the PRs of external contributors.",
				PluginName, tiexternalplugins.ContributionLabel, tiexternalplugins.FirstTimeContributorLabel),
			Config:  configInfo,
			Snippet: yamlSnippet,
			Events:  []string{tiexternalplugins.PullRequestEvent},
		}

		return pluginHelp, nil
	}
}

func HandlePullRequestEvent(gc githubClient, pe *github.PullRequestEvent,
	config *tiexternalplugins.Configuration, log *logrus.Entry) error {
	if pe.Action != github.PullRequestActionOpened {
		log.Debug("Not a pull request opened or reopened action, skipping...")
		return nil
	}

	org := pe.Repo.Owner.Login
	repo := pe.Repo.Name
	num := pe.Number
	author := pe.PullRequest.User.Login

	var needsAddLabels []string

	isMember, err := gc.IsMember(org, author)
	if err != nil {
		return err
	}

	// If the author is not a member of the organization, we need to add a contribution label.
	if !isMember {
		needsAddLabels = append(needsAddLabels, tiexternalplugins.ContributionLabel)
	}

	isFirstTime := pe.PullRequest.AuthorAssociation == firstTimer ||
		pe.PullRequest.AuthorAssociation == firstTimeContributor
	// If it is the first contribution, you need to add the first first-time-contributor label.
	if isFirstTime {
		needsAddLabels = append(needsAddLabels, tiexternalplugins.FirstTimeContributorLabel)
	}

	if len(needsAddLabels) > 0 {
		log.Infof("Adding labels %v.", needsAddLabels)
		err = gc.AddLabels(org, repo, num, needsAddLabels...)
		if err != nil {
			return err
		}
	}

	opts := config.ContributionFor(org, repo)
	if len(needsAddLabels) > 0 && len(opts.Message) != 0 {
		return gc.CreateComment(org, repo, num, tiexternalplugins.FormatSimpleResponse(author, opts.Message))
	}

	return nil
}
