package externalplugins

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestGetConfig(t *testing.T) {
	var testcases = []struct {
		lgtm                     *TiCommunityLgtm
		expectedPullReviewersURL string
	}{
		{
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-dev"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullReviewersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expectedPullReviewersURL: "https://bots.tidb.io/ti-community-bot",
		},
		{
			lgtm: &TiCommunityLgtm{
				Repos:            []string{"tidb-community-bots/test-live"},
				ReviewActsAsLgtm: true,
				StoreTreeHash:    true,
				StickyLgtmTeam:   "tidb-community-bots/bots-test",
				PullReviewersURL: "https://bots.tidb.io/ti-community-bot",
			},
			expectedPullReviewersURL: "https://bots.tidb.io/ti-community-bot",
		},
	}
	for _, tc := range testcases {
		pa := ConfigAgent{configuration: &Configuration{TiCommunityLgtm: []TiCommunityLgtm{*tc.lgtm}}}

		config := pa.Config()
		for _, lgtm := range config.TiCommunityLgtm {
			if lgtm.PullReviewersURL != tc.expectedPullReviewersURL {
				t.Errorf("Different URL: Got \"%s\" expected \"%s\"", lgtm.PullReviewersURL, tc.expectedPullReviewersURL)
			}
		}
	}
}

func TestStartLoadConfig(t *testing.T) {
	pa := ConfigAgent{}

	// Create a tmp config file.
	tmp := "../../test/testdata/config_tmp.yaml"
	_ = os.Remove(tmp)
	// Change pull config duration.
	pullDuration = 1 * time.Second

	// Test and update config.
	testConfigPath := "../../test/testdata/config_test.yaml"
	updateConfigPath := "../../test/testdata/config_update.yaml"

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
	if config.TiCommunityLgtm[expectLen-1].StoreTreeHash != true {
		t.Errorf("Different StoreTreeHash: Got \"%v\" expected \"%v\"",
			config.TiCommunityLgtm[expectLen-1].StoreTreeHash, true)
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
	if config.TiCommunityLgtm[expectLen-1].StoreTreeHash == false {
		t.Errorf("Different StoreTreeHash: Got \"%v\" expected \"%v\"",
			config.TiCommunityLgtm[expectLen-1].StoreTreeHash, false)
	}

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
