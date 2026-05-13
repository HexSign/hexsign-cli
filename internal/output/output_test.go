package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in      string
		want    Format
		wantErr bool
	}{
		{"", FormatTable, false},
		{"table", FormatTable, false},
		{"json", FormatJSON, false},
		{"yaml", "", true},
		{"TABLE", "", true}, // case-sensitive — be strict
	}
	for _, tc := range cases {
		t.Run("parse "+tc.in, func(t *testing.T) {
			got, err := Parse(tc.in)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTable_RendersHeadersAndRows(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable(&buf, "ID", "NAME", "STATUS")
	tbl.Append("c-1", "Distribution", "valid")
	tbl.Append("c-2", "Development", "expiring_soon")
	tbl.Render()

	rendered := buf.String()
	for _, want := range []string{"ID", "NAME", "STATUS", "c-1", "Distribution", "valid", "c-2", "expiring_soon"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("table output missing %q\n%s", want, rendered)
		}
	}
}

func TestPrint_RoutesByFormat(t *testing.T) {
	// JSON format must not call the table renderer.
	renderCalled := false
	render := func() error {
		renderCalled = true
		return nil
	}
	if err := Print(FormatJSON, struct {
		Name string `json:"name"`
	}{"x"}, render); err != nil {
		t.Fatalf("Print json: %v", err)
	}
	if renderCalled {
		t.Error("table renderer must not be called when format is JSON")
	}

	// Table format must call the renderer instead of serializing JSON.
	renderCalled = false
	if err := Print(FormatTable, nil, render); err != nil {
		t.Fatalf("Print table: %v", err)
	}
	if !renderCalled {
		t.Error("table renderer should be called when format is table")
	}
}
