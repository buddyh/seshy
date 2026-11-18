package main

import (
	"os"

	"github.com/buddyh/seshy/internal/plugin"
	"github.com/spf13/cobra"
)

// newMCPCommand runs the read-only plugin transport without the TUI lifecycle.
func newMCPCommand() *cobra.Command {
	return &cobra.Command{
		Use: "mcp", Short: "Serve local conversation recovery tools over MCP stdio", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return plugin.NewServer(plugin.NewService()).Serve(os.Stdin, os.Stdout)
		},
	}
}
