package label

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/github/fakegithub"
)

func formatLabels(labels ...string) []string {
	var r []string
	for _, l := range labels {
		r = append(r, fmt.Sprintf("%s/%s#%d:%s", "org", "repo", 1, l))
	}
	if len(r) == 0 {
		return nil
	}
	return r
}

func TestLabelIssueComment(t *testing.T) {
	type testCase struct {
		name             string
		body             string
		additionalLabels []string
		prefixes         []string
		excludeLabels    []string

		expectedNewLabels     []string
		expectedRemovedLabels []string
		expectedBotComment    bool
		repoLabels            []string
		issueLabels           []string
	}
	testcases := []testCase{
		{
			name:                  "Irrelevant comment",
			body:                  "irrelelvant",
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			repoLabels:            []string{},
			issueLabels:           []string{},
		},
		{
			name:                  "Empty Component",
			body:                  "/component",
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			repoLabels:            []string{"component/infra"},
			issueLabels:           []string{"component/infra"},
		},
		{
			name:                  "Add Single Component Label",
			body:                  "/component infra",
			repoLabels:            []string{"component/infra"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("component/infra"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Single Component Label when already present on Issue",
			body:                  "/component infra",
			repoLabels:            []string{"component/infra"},
			issueLabels:           []string{"component/infra"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Single Priority Label",
			body:                  "/priority critical",
			repoLabels:            []string{"component/infra", "priority/critical"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("priority/critical"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Single Type Label",
			body:                  "/type bug",
			repoLabels:            []string{"component/infra", "priority/critical", "type/bug"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("type/bug"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Single Status Label",
			body:                  "/status needs-information",
			repoLabels:            []string{"component/infra", "status/needs-information"},
			issueLabels:           []string{"component/infra"},
			expectedNewLabels:     formatLabels("status/needs-information"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Adding Labels is Case Insensitive",
			body:                  "/type BuG",
			repoLabels:            []string{"component/infra", "priority/critical", "type/bug"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("type/bug"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Adding Labels is Case Insensitive",
			body:                  "/type bug",
			repoLabels:            []string{"component/infra", "priority/critical", "type/BUG"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("type/BUG"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Can't Add Non Existent Label",
			body:                  "/priority critical",
			repoLabels:            []string{"component/infra"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Non Org Member Can't Add",
			body:                  "/component infra",
			repoLabels:            []string{"component/infra", "priority/critical", "type/bug"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("component/infra"),
			expectedRemovedLabels: []string{},
		},
		{
			name: "Command must start at the beginning of the line",
			body: "  /component infra",
			repoLabels: []string{"component/infra", "component/api", "priority/critical",
				"priority/urgent", "priority/important", "type/bug"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Can't Add Labels Non Existing Labels",
			body:                  "/component lgtm",
			repoLabels:            []string{"component/infra", "component/api", "priority/critical"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Multiple Component Labels",
			body:                  "/component api infra",
			repoLabels:            []string{"component/infra", "component/api", "priority/critical", "priority/urgent"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("component/api", "component/infra"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Multiple Component Labels one already present on Issue",
			body:                  "/component api infra",
			repoLabels:            []string{"component/infra", "component/api", "priority/critical", "priority/urgent"},
			issueLabels:           []string{"component/api"},
			expectedNewLabels:     formatLabels("component/infra"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Multiple Priority Labels",
			body:                  "/priority critical important",
			repoLabels:            []string{"priority/critical", "priority/important"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("priority/critical", "priority/important"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Label Prefix Must Match Command (Component-Priority Mismatch)",
			body:                  "/component urgent",
			repoLabels:            []string{"component/infra", "component/api", "priority/critical", "priority/urgent"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Label Prefix Must Match Command (Priority-Component Mismatch)",
			body:                  "/priority infra",
			repoLabels:            []string{"component/infra", "component/api", "priority/critical", "priority/urgent"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels(),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Multiple Component Labels (Some Valid)",
			body:                  "/component lgtm infra",
			repoLabels:            []string{"component/infra", "component/api"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("component/infra"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Multiple Committee Labels (Some Valid)",
			body:                  "/committee steering calamity",
			prefixes:              []string{"committee"},
			repoLabels:            []string{"committee/conduct", "committee/steering"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("committee/steering"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add Multiple Types of Labels Different Lines",
			body:                  "/priority urgent\n/component infra",
			repoLabels:            []string{"component/infra", "priority/urgent"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("priority/urgent", "component/infra"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Remove Component Label when no such Label on Repo",
			body:                  "/remove-component infra",
			repoLabels:            []string{},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			expectedBotComment:    true,
		},
		{
			name:                  "Remove Component Label when no such Label on Issue",
			body:                  "/remove-component infra",
			repoLabels:            []string{"component/infra"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			expectedBotComment:    true,
		},
		{
			name:                  "Remove Component Label",
			body:                  "/remove-component infra",
			repoLabels:            []string{"component/infra"},
			issueLabels:           []string{"component/infra"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("component/infra"),
		},
		{
			name:                  "Remove Committee Label",
			body:                  "/remove-committee infinite-monkeys",
			prefixes:              []string{"committee"},
			repoLabels:            []string{"component/infra", "sig/testing", "committee/infinite-monkeys"},
			issueLabels:           []string{"component/infra", "sig/testing", "committee/infinite-monkeys"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("committee/infinite-monkeys"),
		},
		{
			name:                  "Remove Type Label",
			body:                  "/remove-type bug",
			repoLabels:            []string{"component/infra", "priority/high", "type/bug"},
			issueLabels:           []string{"component/infra", "priority/high", "type/bug"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("type/bug"),
		},
		{
			name:                  "Remove Priority Label",
			body:                  "/remove-priority high",
			repoLabels:            []string{"component/infra", "priority/high"},
			issueLabels:           []string{"component/infra", "priority/high"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("priority/high"),
		},
		{
			name:                  "Remove SIG Label",
			body:                  "/remove-sig testing",
			prefixes:              []string{"sig"},
			repoLabels:            []string{"component/infra", "sig/testing"},
			issueLabels:           []string{"component/infra", "sig/testing"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("sig/testing"),
		},
		{
			name:                  "Remove WG Policy",
			body:                  "/remove-wg policy",
			prefixes:              []string{"wg"},
			repoLabels:            []string{"component/infra", "wg/policy"},
			issueLabels:           []string{"component/infra", "wg/policy"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("wg/policy"),
		},
		{
			name:                  "Remove Status Label",
			body:                  "/remove-status needs-information",
			repoLabels:            []string{"component/infra", "status/needs-information"},
			issueLabels:           []string{"component/infra", "status/needs-information"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("status/needs-information"),
		},
		{
			name:                  "Remove Multiple Labels",
			body:                  "/remove-priority low high\n/remove-type bug\n/remove-component  infra",
			repoLabels:            []string{"component/infra", "priority/high", "priority/low", "type/bug"},
			issueLabels:           []string{"component/infra", "priority/high", "priority/low", "type/bug"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("priority/low", "priority/high", "type/bug", "component/infra"),
			expectedBotComment:    true,
		},
		{
			name:                  "Add and Remove Label at the same time",
			body:                  "/remove-component infra\n/component test",
			repoLabels:            []string{"component/infra", "component/test"},
			issueLabels:           []string{"component/infra"},
			expectedNewLabels:     formatLabels("component/test"),
			expectedRemovedLabels: formatLabels("component/infra"),
		},
		{
			name:                  "Add and Remove the same Label",
			body:                  "/remove-component infra\n/component infra",
			repoLabels:            []string{"component/infra"},
			issueLabels:           []string{"component/infra"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("component/infra"),
		},
		{
			name: "Multiple Add and Delete Labels",
			body: "/remove-component ruby\n/remove-type srv\n/remove-priority l m\n/component go\n/type cli\n/priority h",
			repoLabels: []string{"component/go", "component/ruby",
				"type/cli", "type/srv", "priority/h", "priority/m", "priority/l"},
			issueLabels:           []string{"component/ruby", "type/srv", "priority/l", "priority/m"},
			expectedNewLabels:     formatLabels("component/go", "type/cli", "priority/h"),
			expectedRemovedLabels: formatLabels("component/ruby", "type/srv", "priority/l", "priority/m"),
		},
		{
			name:                  "Do nothing with empty /label command",
			body:                  "/label",
			additionalLabels:      []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Do nothing with empty /remove-label command",
			body:                  "/remove-label",
			additionalLabels:      []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add custom label",
			body:                  "/label orchestrator/foo",
			additionalLabels:      []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("orchestrator/foo"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Cannot add missing custom label",
			body:                  "/label orchestrator/foo",
			additionalLabels:      []string{"orchestrator/jar", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Remove custom label",
			body:                  "/remove-label orchestrator/foo",
			additionalLabels:      []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{"orchestrator/foo"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("orchestrator/foo"),
		},
		{
			name:                  "Cannot remove missing custom label",
			body:                  "/remove-label orchestrator/jar",
			additionalLabels:      []string{"orchestrator/foo", "orchestrator/bar"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{"orchestrator/foo"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Do nothing with empty custom prefixed label",
			body:                  "/orchestrator",
			prefixes:              []string{"orchestrator"},
			additionalLabels:      []string{},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Do nothing with empty remove custom prefixed label",
			body:                  "/remove-orchestrator",
			prefixes:              []string{"orchestrator"},
			additionalLabels:      []string{},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add custom prefixed label",
			body:                  "/orchestrator foo",
			prefixes:              []string{"orchestrator"},
			additionalLabels:      []string{},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     formatLabels("orchestrator/foo"),
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add already existing custom prefixed label",
			body:                  "/orchestrator foo",
			prefixes:              []string{"orchestrator"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{"orchestrator/foo"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Add non-existent custom prefixed label",
			body:                  "/orchestrator bar",
			prefixes:              []string{"orchestrator"},
			repoLabels:            []string{"orchestrator/foo"},
			issueLabels:           []string{},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
		},
		{
			name:                  "Remove custom prefixed label",
			body:                  "/remove-orchestrator foo",
			prefixes:              []string{"orchestrator"},
			repoLabels:            []string{"orchestrator/foo", "orchestrator/bar"},
			issueLabels:           []string{"orchestrator/foo", "orchestrator/bar"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: formatLabels("orchestrator/foo"),
		},
		{
			name:                  "Remove already missing custom prefixed label",
			body:                  "/remove-orchestrator foo",
			prefixes:              []string{"orchestrator"},
			repoLabels:            []string{"orchestrator/foo", "orchestrator/bar"},
			issueLabels:           []string{"orchestrator/bar"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			expectedBotComment:    true,
		},
		{
			name:                  "Remove non-existent custom prefixed label",
			body:                  "/remove-orchestrator jar",
			prefixes:              []string{"orchestrator"},
			repoLabels:            []string{"orchestrator/foo", "orchestrator/bar"},
			issueLabels:           []string{"orchestrator/bar"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			expectedBotComment:    true,
		},
		{
			name:                  "Add exclude label",
			body:                  "/orchestrator foo",
			prefixes:              []string{"orchestrator"},
			repoLabels:            []string{"orchestrator/foo", "orchestrator/bar"},
			issueLabels:           []string{"orchestrator/bar"},
			excludeLabels:         []string{"orchestrator/foo"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			expectedBotComment:    false,
		},
		{
			name:                  "Remove exclude label",
			body:                  "/remove-orchestrator foo",
			prefixes:              []string{"orchestrator"},
			repoLabels:            []string{"orchestrator/foo", "orchestrator/bar"},
			issueLabels:           []string{"orchestrator/bar", "orchestrator/foo"},
			excludeLabels:         []string{"orchestrator/foo"},
			expectedNewLabels:     []string{},
			expectedRemovedLabels: []string{},
			expectedBotComment:    false,
		},
	}

	for _, tc := range testcases {
		t.Logf("Running scenario %q", tc.name)
		sort.Strings(tc.expectedNewLabels)

		issue := github.Issue{
			Body:   "Test",
			Number: 1,
			User:   github.User{Login: "Alice"},
		}
		issueComment := github.IssueComment{
			Body: tc.body,
			User: github.User{Login: "Alice"},
		}

		fakeClient := &fakegithub.FakeClient{
			Issues: map[int]*github.Issue{
				1: &issue,
			},
			IssueComments:      make(map[int][]github.IssueComment),
			RepoLabelsExisting: tc.repoLabels,
			IssueLabelsAdded:   []string{},
			IssueLabelsRemoved: []string{},
		}
		// Add initial labels
		for _, label := range tc.issueLabels {
			_ = fakeClient.AddLabel("org", "repo", 1, label)
		}
		e := &github.IssueCommentEvent{
			Action:  github.IssueCommentActionCreated,
			Issue:   issue,
			Comment: issueComment,
			Repo:    github.Repo{Owner: github.User{Login: "org"}, Name: "repo"},
		}
		// Add default prefixes if none present
		if tc.prefixes == nil {
			tc.prefixes = []string{"component", "type", "priority", "status"}
		}
		cfg := &externalplugins.Configuration{
			TiCommunityLabel: []externalplugins.TiCommunityLabel{{
				Repos:            []string{"org/repo"},
				AdditionalLabels: tc.additionalLabels,
				Prefixes:         tc.prefixes,
				ExcludeLabels:    tc.excludeLabels,
			}},
		}
		err := HandleIssueCommentEvent(fakeClient, e, cfg, logrus.WithField("plugin", PluginName))
		if err != nil {
			t.Errorf("didn't expect error from label test: %v", err)
			continue
		}

		// Check that all the correct labels (and only the correct labels) were added.
		expectLabels := append(formatLabels(tc.issueLabels...), tc.expectedNewLabels...)
		if expectLabels == nil {
			expectLabels = []string{}
		}
		sort.Strings(expectLabels)
		sort.Strings(fakeClient.IssueLabelsAdded)
		if !reflect.DeepEqual(expectLabels, fakeClient.IssueLabelsAdded) {
			t.Errorf("expected the labels %q to be added, but %q were added.", expectLabels, fakeClient.IssueLabelsAdded)
		}

		sort.Strings(tc.expectedRemovedLabels)
		sort.Strings(fakeClient.IssueLabelsRemoved)
		if !reflect.DeepEqual(tc.expectedRemovedLabels, fakeClient.IssueLabelsRemoved) {
			t.Errorf("expected the labels %q to be removed, but %q were removed.",
				tc.expectedRemovedLabels, fakeClient.IssueLabelsRemoved)
		}
		if len(fakeClient.IssueCommentsAdded) > 0 && !tc.expectedBotComment {
			t.Errorf("unexpected bot comments: %#v", fakeClient.IssueCommentsAdded)
		}
		if len(fakeClient.IssueCommentsAdded) == 0 && tc.expectedBotComment {
			t.Error("expected a bot comment but got none")
		}
	}
}

func TestHelpProvider(t *testing.T) {
	configInfoHasPrefixesPrefix := "The label plugin also includes commands based on"
	configInfoHasAdditionalLabelsSuffix := "labels can be used with the `/[remove-]label` command."
	configInfoHasExcludeLabelsSuffix := "labels cannot be added by command."
	enabledRepos := []config.OrgRepo{
		{Org: "org1", Repo: "repo"},
		{Org: "org2", Repo: "repo"},
	}
	testcases := []struct {
		name               string
		config             *externalplugins.Configuration
		enabledRepos       []config.OrgRepo
		err                bool
		configInfoIncludes []string
		configInfoExcludes []string
	}{
		{
			name:         "Empty config",
			config:       &externalplugins.Configuration{},
			enabledRepos: enabledRepos,
			configInfoExcludes: []string{configInfoHasPrefixesPrefix,
				configInfoHasAdditionalLabelsSuffix, configInfoHasExcludeLabelsSuffix},
		},
		{
			name: "Prefixes added",
			config: &externalplugins.Configuration{
				TiCommunityLabel: []externalplugins.TiCommunityLabel{
					{
						Repos:    []string{"org2/repo"},
						Prefixes: []string{"test"},
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configInfoHasPrefixesPrefix},
			configInfoExcludes: []string{configInfoHasAdditionalLabelsSuffix, configInfoHasExcludeLabelsSuffix},
		},
		{
			name: "Additional labels added",
			config: &externalplugins.Configuration{
				TiCommunityLabel: []externalplugins.TiCommunityLabel{
					{
						Repos:            []string{"org2/repo"},
						AdditionalLabels: []string{"test/a"},
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configInfoHasAdditionalLabelsSuffix},
			configInfoExcludes: []string{configInfoHasPrefixesPrefix, configInfoHasExcludeLabelsSuffix},
		},
		{
			name: "Exclude labels added",
			config: &externalplugins.Configuration{
				TiCommunityLabel: []externalplugins.TiCommunityLabel{
					{
						Repos:            []string{"org2/repo"},
						AdditionalLabels: []string{"test/a"},
						ExcludeLabels:    []string{"test/b"},
					},
				},
			},
			enabledRepos:       enabledRepos,
			configInfoIncludes: []string{configInfoHasAdditionalLabelsSuffix, configInfoHasExcludeLabelsSuffix},
			configInfoExcludes: []string{configInfoHasPrefixesPrefix},
		},
		{
			name: "All configs enabled",
			config: &externalplugins.Configuration{
				TiCommunityLabel: []externalplugins.TiCommunityLabel{
					{
						Repos:            []string{"org2/repo"},
						Prefixes:         []string{"test"},
						AdditionalLabels: []string{"test/a"},
						ExcludeLabels:    []string{"test/b"},
					},
				},
			},
			enabledRepos: enabledRepos,
			configInfoIncludes: []string{configInfoHasPrefixesPrefix,
				configInfoHasAdditionalLabelsSuffix, configInfoHasAdditionalLabelsSuffix},
		},
	}
	for _, testcase := range testcases {
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
