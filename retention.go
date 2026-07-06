package main

import (
	"fmt"
	"os"
	"strings"

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

	var protectDays int
	var protectYes bool
	protectCmd := &cobra.Command{
		Use:   "protect",
		Short: "Raise every auto-deleting agent to a safe retention in one go",
		Long: "Plan and apply retention changes for every agent that auto-deletes " +
			"sessions (Claude Code and Gemini CLI), so at least --days of history " +
			"survives. Shows the exact before/after edits and asks before writing; " +
			"each config file is backed up to <file>.bak first.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if protectDays < 1 {
				return fmt.Errorf("--days must be at least 1, got %d", protectDays)
			}
			changes, notes := retention.ProtectPlan(store.Home, protectDays)
			fmt.Printf("seshy retention protect — keep at least %d days of sessions\n\n", protectDays)
			for _, ch := range changes {
				fmt.Printf("  %-8s %s\n", ch.Agent, ch.File)
				fmt.Printf("           %s   %s -> %s\n", ch.Key, ch.Before, ch.After)
			}
			for _, n := range notes {
				fmt.Println("  skipped:", n)
			}
			if len(changes) == 0 {
				fmt.Printf("\nnothing to do — every auto-deleting agent already keeps >= %dd\n", protectDays)
				return nil
			}
			if !protectYes {
				if !isTTY() {
					return fmt.Errorf("refusing to modify agent configs non-interactively; pass --yes")
				}
				fmt.Printf("\nEach existing file is backed up to <file>.bak before writing.\n")
				fmt.Printf("Apply %d change%s? [y/N] ", len(changes), pluralS(len(changes)))
				var answer string
				fmt.Fscanln(os.Stdin, &answer)
				switch strings.ToLower(strings.TrimSpace(answer)) {
				case "y", "yes":
				default:
					fmt.Println("aborted — nothing written")
					return nil
				}
			}
			for _, ch := range changes {
				if err := ch.Apply(); err != nil {
					return fmt.Errorf("%s: %w", ch.Agent, err)
				}
				fmt.Printf("  applied %s %s = %s\n", ch.Agent, ch.Key, ch.After)
			}
			return nil
		},
	}
	protectCmd.Flags().IntVar(&protectDays, "days", 365, "minimum days of history to keep")
	protectCmd.Flags().BoolVar(&protectYes, "yes", false, "apply without prompting")

	cmd.AddCommand(setCmd, protectCmd)
	return cmd
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
