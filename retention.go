package main

import (
	"fmt"
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

	setCmd := &cobra.Command{
		Use:   "set <agent> <days|off>",
		Short: "Change an agent's session retention (claude, gemini)",
		Long: "Write the retention setting of an agent that auto-deletes sessions. " +
			"claude: days or off (~/.claude/settings.json cleanupPeriodDays). " +
			"gemini: a duration like 365d/52w/12m or off (~/.gemini/settings.json " +
			"general.sessionRetention). The file is backed up to <file>.bak first.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ch, err := retention.Set(store.Home, args[0], args[1])
			if err != nil {
				return err
			}
			if err := ch.Apply(); err != nil {
				return err
			}
			fmt.Printf("set %s %s: %s -> %s\n", ch.Agent, ch.Key, ch.Before, ch.After)
			if ch.NewFile {
				fmt.Printf("  %s (created)\n", ch.File)
			} else {
				fmt.Printf("  %s (backup: %s.bak)\n", ch.File, ch.File)
			}
			return nil
		},
	}

	cmd.AddCommand(setCmd)
	return cmd
}
