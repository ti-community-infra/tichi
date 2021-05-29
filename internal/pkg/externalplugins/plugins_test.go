package externalplugins

import (
	"io/ioutil"
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
	config := pa.Config()
	expectLen := 1
	if len(config.TiCommunityLgtm) != expectLen {
		t.Errorf("Different TiCommunityLgtm len: Got \"%d\" expected \"%d\"",
			len(config.TiCommunityLgtm), expectLen)
	}

	// Move test config into tmp.
	{
		testInput, err := ioutil.ReadFile(testConfigPath)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}

		err = ioutil.WriteFile(tmp, testInput, 0600)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}
	}

	{
		// Move update config into test config.
		updateInput, err := ioutil.ReadFile(updateConfigPath)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}

		err = ioutil.WriteFile(testConfigPath, updateInput, 0600)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}
	}

	// Wait a moment.
	time.Sleep(pullDuration + 1)

	{
		// Move tmp config back to test config file.
		tmpInput, err := ioutil.ReadFile(tmp)
		if err != nil {
			t.Errorf("unexpected error: '%v'", err)
		}

		err = ioutil.WriteFile(testConfigPath, tmpInput, 0600)
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
