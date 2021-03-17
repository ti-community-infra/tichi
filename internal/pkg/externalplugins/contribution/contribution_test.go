package contribution

import (
	"strings"
	"testing"

	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/config"
)

func TestHelpProvider(t *testing.T) {
	enabledRepos := []config.OrgRepo{
		{Org: "org1", Repo: "repo"},
		{Org: "org2", Repo: "repo"},
	}
	cases := []struct {
		name               string
		config             *externalplugins.Configuration
		enabledRepos       []config.OrgRepo
		err                bool
		configInfoIncludes []string
		configInfoExcludes []string
	}{
		{
			name:               "Empty config",
			config:             &externalplugins.Configuration{},
			enabledRepos:       enabledRepos,
			configInfoExcludes: []string{"message: "},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityContribution: []externalplugins.TiCommunityContribution{
					{
						Repos:   []string{"org2/repo"},
						Message: "Message",
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{"message: "},
		},
	}
	for _, testcase := range cases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			epa := &externalplugins.ConfigAgent{}
			epa.Set(tc.config)

			helpProvider := HelpProvider(epa)
			pluginHelp, err := helpProvider(tc.enabledRepos)
			if err != nil && !tc.err {
				t.Fatalf("helpProvider error: %v", err)
			}
			for _, msg := range tc.configInfoExcludes {
				if strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: got %v, but didn't want it", msg)
				}
			}
			for _, msg := range tc.configInfoIncludes {
				if !strings.Contains(pluginHelp.Config["org2/repo"], msg) {
					t.Fatalf("helpProvider.Config error mismatch: didn't get %v, but wanted it", msg)
				}
			}
		})
	}
}
