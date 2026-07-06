package main

import (
	"os"

	"github.com/buddyh/seshy/internal/render"
	"github.com/buddyh/seshy/internal/retention"
	"github.com/buddyh/seshy/internal/store"
	"github.com/spf13/cobra"
)

// newRetentionCmd builds `seshy retention`: a per-agent report of where
// sessions live, how much disk they use, and which agents auto-delete them.
func newRetentionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "retention",
		Short: "Where each agent keeps your sessions — and which ones auto-delete them",
		Long: "Report every supported agent's session store: location, disk usage, " +
			"session count, oldest session, and the retention policy in effect. " +
			"Claude Code and Gemini CLI delete old sessions by default (30 days); " +
			"use 'seshy retention set' or 'seshy retention protect' to change that.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			rows := retention.Report(store.Home)
			if flagAgent != "" {
				var f []retention.Row
				for _, r := range rows {
					if r.Key == flagAgent {
						f = append(f, r)
					}
				}
				rows = f
			}
			render.Retention(os.Stdout, rows, store.Home, pickFormat())
			return nil
		},
	}
	return cmd
}
