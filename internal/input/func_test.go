package input

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMakeInputFile(t *testing.T) {
	tests := []struct {
		name     string
		skelton  any
		filename string
		want     string
	}{
		{
			name: "simple struct",
			skelton: struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}{
				Name:  "test",
				Value: 123,
			},
			filename: "test_simple.json",
			want:     `{"name":"test","value":123}`,
		},
		{
			name: "empty struct",
			skelton: struct {
				Field1 string `json:"field1"`
				Field2 int    `json:"field2"`
			}{},
			filename: "test_empty.json",
			want:     `{"field1":"","field2":0}`,
		},
		{
			name: "nested struct",
			skelton: struct {
				Name   string `json:"name"`
				Config struct {
					Enabled bool   `json:"enabled"`
					Port    int    `json:"port"`
					Host    string `json:"host"`
				} `json:"config"`
			}{
				Name: "nested-test",
				Config: struct {
					Enabled bool   `json:"enabled"`
					Port    int    `json:"port"`
					Host    string `json:"host"`
				}{
					Enabled: true,
					Port:    8080,
					Host:    "localhost",
				},
			},
			filename: "test_nested.json",
			want:     `{"name":"nested-test","config":{"enabled":true,"port":8080,"host":"localhost"}}`,
		},
		{
			name: "slice field",
			skelton: struct {
				Items []string `json:"items"`
				Count int      `json:"count"`
			}{
				Items: []string{"item1", "item2", "item3"},
				Count: 3,
			},
			filename: "test_slice.json",
			want:     `{"items":["item1","item2","item3"],"count":3}`,
		},
	}

	// Create temporary directory for test files
	tempDir := t.TempDir()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare file path
			testFilePath := filepath.Join(tempDir, tt.filename)

			// Call MakeInputFile
			MakeInputFile(tt.skelton, testFilePath)

			// Check if file exists
			if _, err := os.Stat(testFilePath); os.IsNotExist(err) {
				t.Fatalf("File %s was not created", testFilePath)
			}

			// Read and verify file content
			content, err := os.ReadFile(testFilePath)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			if string(content) != tt.want {
				t.Errorf("File content mismatch\ngot:  %s\nwant: %s", string(content), tt.want)
			}

			// Clean up
			os.Remove(testFilePath)
		})
	}
}

func TestMakeInputFile_WithReadInputFile(t *testing.T) {
	// Integration test with ReadInputFile
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "integration_test.json")

	// Define test struct
	type TestStruct struct {
		Name     string            `json:"name"`
		Value    int               `json:"value"`
		Enabled  bool              `json:"enabled"`
		Tags     []string          `json:"tags"`
		Metadata map[string]string `json:"metadata"`
	}

	// Create test data
	original := TestStruct{
		Name:    "integration-test",
		Value:   42,
		Enabled: true,
		Tags:    []string{"test", "integration"},
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	// Write using MakeInputFile
	MakeInputFile(original, testFilePath)

	// Read using ReadInputFile
	var result TestStruct
	ReadInputFile(&result, testFilePath)

	// Compare original and result
	if result.Name != original.Name {
		t.Errorf("Name mismatch: got %s, want %s", result.Name, original.Name)
	}
	if result.Value != original.Value {
		t.Errorf("Value mismatch: got %d, want %d", result.Value, original.Value)
	}
	if result.Enabled != original.Enabled {
		t.Errorf("Enabled mismatch: got %v, want %v", result.Enabled, original.Enabled)
	}
	if len(result.Tags) != len(original.Tags) {
		t.Errorf("Tags length mismatch: got %d, want %d", len(result.Tags), len(original.Tags))
	} else {
		for i, tag := range result.Tags {
			if tag != original.Tags[i] {
				t.Errorf("Tag[%d] mismatch: got %s, want %s", i, tag, original.Tags[i])
			}
		}
	}
	if len(result.Metadata) != len(original.Metadata) {
		t.Errorf("Metadata length mismatch: got %d, want %d", len(result.Metadata), len(original.Metadata))
	} else {
		for k, v := range result.Metadata {
			if result.Metadata[k] != v {
				t.Errorf("Metadata[%s] mismatch: got %s, want %s", k, result.Metadata[k], v)
			}
		}
	}
}

func TestMakeInputFile_SpecialCharacters(t *testing.T) {
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "special_chars.json")

	// Test with special characters and unicode
	data := struct {
		Text     string `json:"text"`
		Japanese string `json:"japanese"`
		Emoji    string `json:"emoji"`
	}{
		Text: `Special "quotes" and \backslashes\ and newlines
here`,
		Japanese: "こんにちは世界",
		Emoji:    "😀🎉",
	}

	MakeInputFile(data, testFilePath)

	// Verify the file was created and can be read back
	var result struct {
		Text     string `json:"text"`
		Japanese string `json:"japanese"`
		Emoji    string `json:"emoji"`
	}
	ReadInputFile(&result, testFilePath)

	if result.Text != data.Text {
		t.Errorf("Text with special characters mismatch")
	}
	if result.Japanese != data.Japanese {
		t.Errorf("Japanese text mismatch")
	}
	if result.Emoji != data.Emoji {
		t.Errorf("Emoji mismatch")
	}
}
