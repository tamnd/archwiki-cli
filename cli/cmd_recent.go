package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) recentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recent",
		Short: "List recent ArchWiki changes",
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(20)
			a.progressf("fetching %d recent changes...", n)
			changes, err := a.client.Recent(cmd.Context(), n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(changes, len(changes))
		},
	}
	return cmd
}
