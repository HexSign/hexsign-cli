package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
)

func Parse(s string) (Format, error) {
	switch s {
	case "", "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	}
	return "", fmt.Errorf("unknown output format %q (want table|json)", s)
}

func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

type Table struct {
	w *tablewriter.Table
}

func NewTable(out io.Writer, headers ...string) *Table {
	if out == nil {
		out = os.Stdout
	}
	t := tablewriter.NewWriter(out)
	t.Configure(func(cfg *tablewriter.Config) {
		cfg.Header.Formatting.AutoFormat = 1
		cfg.Header.Formatting.AutoWrap = 0
	})
	t.Header(headers)
	return &Table{w: t}
}

func (t *Table) Append(row ...string) { t.w.Append(row) }
func (t *Table) Render()              { t.w.Render() }

func Print(f Format, items any, renderTable func() error) error {
	if f == FormatJSON {
		return PrintJSON(items)
	}
	return renderTable()
}
