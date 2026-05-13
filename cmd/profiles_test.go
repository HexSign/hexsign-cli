package cmd

import (
	"strings"
	"testing"
)

func TestValidateProfileDownloadArgs(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		bundleID string
		teamID   string
		filename string

		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "single id, no bulk flags",
			args: []string{"profile-uuid"},
		},
		{
			name:     "single id with --filename — allowed",
			args:     []string{"profile-uuid"},
			filename: "named",
		},
		{
			name:     "bulk: --bundle-id alone",
			args:     []string{},
			bundleID: "com.example.app",
		},
		{
			name:     "bulk: --bundle-id + --team-id",
			args:     []string{},
			bundleID: "com.example.app",
			teamID:   "ABCDE12345",
		},
		{
			name: "neither id nor bundle-id",
			args: []string{},

			wantErr:       true,
			wantErrSubstr: "provide exactly one of <id> or --bundle-id",
		},
		{
			name:     "id and bundle-id together",
			args:     []string{"profile-uuid"},
			bundleID: "com.example.app",

			wantErr:       true,
			wantErrSubstr: "provide exactly one of <id> or --bundle-id",
		},
		{
			name:     "bundle-id with --filename — disallowed (multiple files written)",
			args:     []string{},
			bundleID: "com.example.app",
			filename: "named",

			wantErr:       true,
			wantErrSubstr: "--filename cannot be used with --bundle-id",
		},
		{
			name:   "team-id without bundle-id",
			args:   []string{"profile-uuid"},
			teamID: "ABCDE12345",

			wantErr:       true,
			wantErrSubstr: "--team-id requires --bundle-id",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateProfileDownloadArgs(tc.args, tc.bundleID, tc.teamID, tc.filename)
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

func TestTrimExt(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"hexsign.p12", "hexsign"},
		{"dir/file.txt", "dir/file"},
		{"no-extension", "no-extension"},
		{"", ""},
		{"only.one.dot.txt", "only.one.dot"},
		{"/abs/path.with/dots.p12", "/abs/path.with/dots"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := trimExt(tc.in); got != tc.want {
				t.Errorf("trimExt(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
