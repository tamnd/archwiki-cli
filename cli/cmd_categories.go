package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/archwiki-cli/archwiki"
)

func (a *App) categoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "List main ArchWiki content categories",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cats := archwiki.MainCategories
			return a.renderOrEmpty(cats, len(cats))
		},
	}
	return cmd
}
