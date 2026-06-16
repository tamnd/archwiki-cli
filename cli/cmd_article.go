package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func (a *App) articleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "article <title>",
		Short: "Show the introduction of an ArchWiki article",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching article %q...", args[0])
			ext, err := a.client.Article(cmd.Context(), args[0])
			if err != nil {
				return mapFetchErr(err)
			}
			// plain text by default; structured output when format is set
			if a.output == string(FormatTable) || a.output == string(FormatRaw) {
				_, _ = fmt.Fprintln(os.Stdout, ext.Extract)
				return nil
			}
			return a.render([]any{ext})
		},
	}
	return cmd
}
