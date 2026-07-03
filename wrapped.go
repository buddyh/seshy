package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// wrappedCmd delegates `seshy wrapped` to the seshy-wrapped npm package,
// which shares seshy's on-disk session knowledge and renders the cards.
// It is a thin shim on purpose: the card generator ships over npx so it
// works on machines that have never installed seshy.
var wrappedCmd = &cobra.Command{
	Use:                "wrapped [flags]",
	Short:              "Your AI-coding life, on a shareable card (runs npx seshy-wrapped)",
	DisableFlagParsing: true, // pass everything through untouched
	RunE: func(cmd *cobra.Command, args []string) error {
		npx, err := exec.LookPath("npx")
		if err != nil {
			fmt.Fprintln(os.Stderr, "seshy wrapped needs Node.js (npx) to render cards.")
			fmt.Fprintln(os.Stderr, "Install Node 20+ from https://nodejs.org, then run:")
			fmt.Fprintln(os.Stderr, "  npx seshy-wrapped")
			return fmt.Errorf("npx not found")
		}
		c := exec.Command(npx, append([]string{"--yes", "seshy-wrapped"}, args...)...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}
