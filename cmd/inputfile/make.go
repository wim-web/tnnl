package inputfile

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/internal/input"
)

func New(parent, defaultPath string, skeleton any) *cobra.Command {
	var output string
	var force bool
	c := &cobra.Command{
		Use:   "make-input-file",
		Short: fmt.Sprintf("Create an input file template for %s", parent),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := input.MakeInputFile(skeleton, output, force); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "made %s\n", output)
			return err
		},
	}
	c.Flags().StringVarP(&output, "output", "o", defaultPath, "output path")
	c.Flags().BoolVarP(&force, "force", "f", false, "replace an existing file")
	return c
}
