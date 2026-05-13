package cmd

import (
	"strings"
	"testing"
)

func TestValidateCertDownloadArgs(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		certType string
		teamID   string
		filename string

		wantErr        bool
		wantErrSubstr  string
	}{
		{
			name: "single id, no bulk flags",
			args: []string{"cert-uuid"},
		},
		{
			name:     "bulk: type + team-id",
			args:     []string{},
			certType: "IOS_DISTRIBUTION",
			teamID:   "ABCDE12345",
		},
		{
			name: "neither id nor bulk flags",
			args: []string{},

			wantErr:       true,
			wantErrSubstr: "provide either <id> or both --type and --team-id",
		},
		{
			name:     "only --type without --team-id",
			args:     []string{},
			certType: "IOS_DISTRIBUTION",

			wantErr:       true,
			wantErrSubstr: "--type and --team-id must be provided together",
		},
		{
			name:   "only --team-id without --type",
			args:   []string{},
			teamID: "ABCDE12345",

			wantErr:       true,
			wantErrSubstr: "--type and --team-id must be provided together",
		},
		{
			name:     "id combined with --type",
			args:     []string{"cert-uuid"},
			certType: "IOS_DISTRIBUTION",

			wantErr:       true,
			wantErrSubstr: "cannot be combined with <id>",
		},
		{
			name:     "id combined with --team-id",
			args:     []string{"cert-uuid"},
			teamID:   "ABCDE12345",

			wantErr:       true,
			wantErrSubstr: "cannot be combined with <id>",
		},
		{
			name:     "bulk with --filename",
			args:     []string{},
			certType: "IOS_DISTRIBUTION",
			teamID:   "ABCDE12345",
			filename: "custom",

			wantErr:       true,
			wantErrSubstr: "--filename cannot be used with --type/--team-id",
		},
		{
			name:     "single id with --filename — allowed",
			args:     []string{"cert-uuid"},
			filename: "custom",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateCertDownloadArgs(tc.args, tc.certType, tc.teamID, tc.filename)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Errorf("error = %q, want it to contain %q", err.Error(), tc.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
