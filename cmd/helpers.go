package cmd

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

func newOpCtx(cmd *cobra.Command, timeout time.Duration) (context.Context, context.CancelFunc) {
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	if timeout == 0 {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, timeout)
}

type pageFlags struct {
	page  int
	limit int
}

func (p *pageFlags) bind(cmd *cobra.Command) {
	cmd.Flags().IntVar(&p.page, "page", 1, "page number (1-based)")
	cmd.Flags().IntVar(&p.limit, "limit", 50, "page size")
}

func (p *pageFlags) values() url.Values {
	q := url.Values{}
	if p.page > 0 {
		q.Set("page", strconv.Itoa(p.page))
	}
	if p.limit > 0 {
		q.Set("limit", strconv.Itoa(p.limit))
	}
	return q
}
