package externalplugins

import (
	"os"
	"testing"
	"time"
)

func TestGetConfig(t *testing.T) {
	var testcases = []struct {
		lgtm                  *TiCommunityLgtm
		expectedPullOwnersURL string
	}{
		{
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-dev"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			expectedPullOwnersURL: "https://bots.tidb.io/ti-community-bot",
		},
		{
			lgtm: &TiCommunityLgtm{
				Repos:              []string{"ti-community-infra/test-live"},
				PullOwnersEndpoint: "https://bots.tidb.io/ti-community-bot",
			},
			expectedPullOwnersURL: "https://bots.tidb.io/ti-community-bot",
		},
	}
	for _, tc := range testcases {
		pa := ConfigAgent{configuration: &Configuration{TiCommunityLgtm: []TiCommunityLgtm{*tc.lgtm}}}

		config := pa.Config()
		for _, lgtm := range config.TiCommunityLgtm {
			if lgtm.PullOwnersEndpoint != tc.expectedPullOwnersURL {
				t.Errorf("Different URL: Got \"%s\" expected \"%s\"", lgtm.PullOwnersEndpoint, tc.expectedPullOwnersURL)
			}
		}
	}
}

func TestStartLoadConfig(t *testing.T) {
	pa := ConfigAgent{}

	// Create a tmp config file.
	tmp := "../../../test/testdata/config_tmp.yaml"
	_ = os.Remove(tmp)
	// Change pull config duration.
	pullDuration = 1 * time.Second

	// Test and update config.
	testConfigPath := "../../../test/testdata/config_test.yaml"
	updateConfigPath := "../../../test/testdata/config_update.yaml"

	// Start pull config.
	err := pa.Start(testConfigPath, false)
	if err != nil {
		t.Errorf("unexpected error: '%v'", err)
	}

	// Assert first time.
	expectLen := 1
	if len(pa.Config().TiCommunityLgtm) != expectLen {
		t.Errorf("Different TiCommunityLgtm len: Got \"%d\" expected \"%d\"",
			len(pa.Config().TiCommunityLgtm), expectLen)
	}

	if pa.Config().TiCommunityLgtm[expectLen-1].PullOwnersEndpoint != "https://test" {
		t.Errorf("Different PullOwnersEndpoint: Got \"%v\" expected \"%v\"",
			pa.Config().TiCommunityLgtm[expectLen-1].PullOwnersEndpoint, "https://test")
	}

	// Move test config into tmp.
	{
		testInput, err := os.ReadFile(testConfigPath)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}

		err = os.WriteFile(tmp, testInput, 0600)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}
	}

	{
		// Move update config into test config.
		updateInput, err := os.ReadFile(updateConfigPath)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}

		err = os.WriteFile(testConfigPath, updateInput, 0600)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}
	}

	// Wait a moment.
	time.Sleep(pullDuration * 2)
	if pa.Config().TiCommunityLgtm[expectLen-1].PullOwnersEndpoint != "https://test-updated" {
		t.Errorf("Different PullOwnersEndpoint: Got \"%v\" expected \"%v\"",
			pa.Config().TiCommunityLgtm[expectLen-1].PullOwnersEndpoint, "https://test-updated")
	}

	{
		// Move tmp config back to test config file.
		tmpInput, err := os.ReadFile(tmp)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}

		err = os.WriteFile(testConfigPath, tmpInput, 0600)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}
	}

	_ = os.Remove(tmp)
}

func TestStartLoadFailed(t *testing.T) {
	pa := ConfigAgent{}

	failedPath := "../../test/testdata/config_tmp.yaml"
	_ = os.Remove(failedPath)

	// Start pull config.
	err := pa.Start(failedPath, false)
	if err == nil {
		t.Errorf("expected error, but it is nil")
	}
}
