package main

import (
	"testing"

	docopt "github.com/docopt/docopt-go"
	"github.com/stretchr/testify/require"
)

var usageTestTable = []struct {
	description string   // A description of the test
	argv        []string // Given command line args
	validArgs   bool     // Are the given arguments valid
	wantErr     bool     // Do we expect an error
	opts        docopt.Opts
}{
	// Good cases
	{
		description: "Providing --mkv overrides the default of mp4",
		argv:        []string{"--mkv", "testfile1.ogg"},
		validArgs:   true,
		wantErr:     false,
		opts: docopt.Opts{
			"--mp4":  false,
			"--mkv":  true,
			"<file>": []string{"testfile1.ogg"},
		},
	},

	{
		"Providing no container format is valid",
		[]string{"testfile1.ogg", "testfile2.ogg"},
		true,
		false,
		docopt.Opts{
			"--mp4":  false,
			"--mkv":  false,
			"<file>": []string{"testfile1.ogg", "testfile2.ogg"},
		},
	},

	{
		"Providing multiple files is valid",
		[]string{"--mp4", "testfile1.ogg", "testfile2.ogg"},
		true,
		false,
		docopt.Opts{
			"--mp4":  true,
			"--mkv":  false,
			"<file>": []string{"testfile1.ogg", "testfile2.ogg"},
		},
	},

	// Bad Cases
	{
		"Providing both --mp4 and --mkv is invalid",
		[]string{"--mp4", "--mkv", "testfile.ogg"},
		false,
		true,
		docopt.Opts{
			"--mp4":  true,
			"--mkv":  true,
			"<file>": []string{"testfile.ogg"},
		},
	},
}

func TestUsage(t *testing.T) {
	for _, tt := range usageTestTable {
		t.Run(tt.description, func(t *testing.T) {
			validArgs := true
			parser := &docopt.Parser{
				HelpHandler: func(err error, usage string) {
					if err != nil {
						validArgs = false // Triggered usage, args were invalid.
					}
				},
			}
			opts, err := parser.ParseArgs(usage, tt.argv, "")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.validArgs, validArgs)
				require.Equal(t, tt.opts, opts)
			}
		})
	}
}
