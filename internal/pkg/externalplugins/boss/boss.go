package boss

import (
	"errors"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"k8s.io/test-infra/prow/plugins"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/ownersclient"
)

// 1. 新 pr 识别到匹配文件后自动 打 label
// 2. 识别 人员，分配  boss 人员 到 assignees
// 3. assignees approve 时, 去除标签。
// 2. 指定目录，指定人 满足后

// HelpProvider constructs the PluginHelp for this plugin that takes into account enabled repositories.
func HelpProvider(epa *tiexternalplugins.ConfigAgent) externalplugins.ExternalPluginHelpProvider {
	return func(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
		configInfo := map[string]string{}
		cfg := epa.Config()

		for _, repo := range enabledRepos {
			opts := cfg.BossFor(repo.Org, repo.Repo)
			if opts == nil {
				continue
			}

			var configInfoStrings []string
			configInfoStrings = append(configInfoStrings, "The plugin has these configurations:<ul>")
			// TODO(wuhuizuo): fill it.
			configInfoStrings = append(configInfoStrings, "</ul>")
			configInfo[repo.String()] = strings.Join(configInfoStrings, "\n")
		}

		yamlSnippet, err := plugins.CommentMap.GenYaml(&tiexternalplugins.Configuration{
			TiCommunityBoss: []tiexternalplugins.TiCommunityBoss{
				{
					Repos:     []string{"ti-community-infra/test-dev"},
					Approvers: []string{"boss-a", "boss-b"},
					Patterns:  []string{`^config/.*\.go`, `^var/.*\.go`},
				},
			},
		})
		if err != nil {
			logrus.WithError(err).Warnf("cannot generate comments for %s plugin", PluginName)
			return nil, err
		}

		pluginHelp := &pluginhelp.PluginHelp{
			Description: `The ti-community-boss plugin automatically requests approves to 
 			bosses when changed file hit patterns`,
			Config:  configInfo,
			Snippet: yamlSnippet,
			Events:  []string{tiexternalplugins.PullRequestEvent, tiexternalplugins.IssueCommentEvent},
		}

		return pluginHelp, nil
	}
}

// HandlePullRequestEvent handles a GitHub pull request event and requests review.
func HandlePullRequestEvent(
	githubClient,
	*github.PullRequestEvent,
	*tiexternalplugins.Configuration,
	ownersclient.OwnersLoader,
	*logrus.Entry,
) error {
	return errors.New("not implemented yet")
}

// HandleIssueCommentEvent handles a GitHub issue comment event and requests review.
func HandleIssueCommentEvent(
	githubClient,
	*github.IssueCommentEvent,
	*tiexternalplugins.Configuration,
	ownersclient.OwnersLoader,
	*logrus.Entry,
) error {
	return errors.New("not implemented yet")
}
