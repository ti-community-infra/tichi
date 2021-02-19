package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestOptions(t *testing.T) {
	testcases := []struct {
		name string
		args []string

		expectedError  string
		expectedOption *options
	}{
		{
			name: "no external config path",
			args: []string{},

			expectedError:  "invalid options: required flag --external-plugin-config-path was unset",
			expectedOption: nil,
		},
		{
			name: "has external config path",
			args: []string{
				"--external-plugin-config-path=/etc/external_plugin_config.yaml",
			},

			expectedError: "",
			expectedOption: &options{
				externalPluginConfigPath: "/etc/external_plugin_config.yaml",
			},
		},
	}

	for _, testcase := range testcases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			flags := flag.NewFlagSet(tc.name, flag.ContinueOnError)
			var actualOptions options

			err := actualOptions.gatherOptions(flags, tc.args)

			if err != nil {
				if err.Error() != tc.expectedError {
					t.Errorf("expected error %#v but got %#v", tc.expectedError, err.Error())
				}
			} else {
				if !reflect.DeepEqual(&actualOptions, tc.expectedOption) {
					t.Errorf("expected options %#v but got %#v", tc.expectedOption, actualOptions)
				}
			}
		})
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name string
		opts options
	}{
		{
			name: "combined config",
			opts: options{
				externalPluginConfigPath: "testdata/external_plugins_config.yaml",
			},
		},
	}

	for _, testcase := range testCases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			if err := validate(tc.opts); err != nil {
				t.Fatalf("validation failed: %v", err)
			}
		})
	}
}
