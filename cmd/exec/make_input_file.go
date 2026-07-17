package exec

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/internal/input"
)

var MakeInputFileCmd = &cobra.Command{
	Use:   "make-input-file",
	Short: "make input file skelton for exec",
	Run: func(cmd *cobra.Command, args []string) {
		if err := input.MakeInputFile(input.ExecInput{}, "exec-input.json", false); err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	ExecCmd.AddCommand(MakeInputFileCmd)
}
