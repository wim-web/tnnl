package input

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

func MakeInputFile(skelton any, filepath string) {
	jsonData, err := json.Marshal(skelton)
	if err != nil {
		log.Fatalln("Error encoding JSON:", err)
	}

	file, err := os.Create(filepath)
	if err != nil {
		log.Fatalln("Error creating file:", err)
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		log.Fatalln("Error writing to file:", err)
	}

	fmt.Printf("made %s\n", filepath)
}

func ReadInputFile(v any, filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		log.Fatalln("Error opening file:", err)
	}
	defer file.Close()

	jsonData, err := io.ReadAll(file)
	if err != nil {
		log.Fatalln("Error reading file:", err)
	}

	err = json.Unmarshal(jsonData, &v)
	if err != nil {
		log.Fatalln("Error decoding JSON:", err)
	}
}
