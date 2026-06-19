package cli

import (
	"github.com/spf13/cobra"
)

func (a *App) suggestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "suggest <prefix>",
		Short: "Autocomplete suggestions for an ArchWiki title prefix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n := a.effectiveLimit(10)
			a.progressf("fetching suggestions for %q...", args[0])
			suggestions, err := a.client.Suggest(cmd.Context(), args[0], n)
			if err != nil {
				return mapFetchErr(err)
			}
			return a.renderOrEmpty(suggestions, len(suggestions))
		},
	}
	return cmd
}
