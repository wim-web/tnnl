package makeinputfile

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/wim-web/tnnl/cmd/exec"
	inputfile "github.com/wim-web/tnnl/internal/input-file"
)

var MakeInputFileCmd = &cobra.Command{
	Use:   "make-input-file",
	Short: "make input file skelton for exec",
	Run: func(cmd *cobra.Command, args []string) {
		skelton := inputfile.ExecInputFile{}

		jsonData, err := json.Marshal(skelton)
		if err != nil {
			log.Fatalln("Error encoding JSON:", err)
		}

		file, err := os.Create("exec-input.json")
		if err != nil {
			log.Fatalln("Error creating file:", err)
		}
		defer file.Close()

		_, err = file.Write(jsonData)
		if err != nil {
			log.Fatalln("Error writing to file:", err)
		}

		fmt.Println("made exec-input.json")
	},
}

func init() {
	exec.ExecCmd.AddCommand(MakeInputFileCmd)
}
