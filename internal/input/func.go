package input

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func MakeInputFile(skelton any, filepath string) {
	jsonData, err := json.Marshal(skelton)
	if err != nil {
		panic(fmt.Errorf("Error encoding JSON: %w", err))
	}

	file, err := os.Create(filepath)
	if err != nil {
		panic(fmt.Errorf("Error creating file: %w", err))
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		panic(fmt.Errorf("Error writing to file: %w", err))
	}

	fmt.Printf("made %s\n", filepath)
}

func ReadInputFile(v any, filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		panic(fmt.Errorf("Error opening file: %w", err))
	}
	defer file.Close()

	jsonData, err := io.ReadAll(file)
	if err != nil {
		panic(fmt.Errorf("Error reading file: %w", err))
	}

	err = json.Unmarshal(jsonData, &v)
	if err != nil {
		panic(fmt.Errorf("Error decoding JSON: %w", err))
	}
}
