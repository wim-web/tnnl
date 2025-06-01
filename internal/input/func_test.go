package input

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMakeInputFile(t *testing.T) {
	type TestData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	testData := TestData{Name: "Test User", Age: 30}
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_input.json")

	// Capture stdout to prevent "made ..." message from interfering with test output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	MakeInputFile(testData, filePath)

	w.Close()
	os.Stdout = oldStdout // Restore stdout

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("MakeInputFile did not create the file: %s", filePath)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Error reading created file: %v", err)
	}

	expectedContent := `{"name":"Test User","age":30}`
	if string(content) != expectedContent {
		t.Errorf("MakeInputFile created file with unexpected content.\nExpected: %s\nGot: %s", expectedContent, string(content))
	}
}

func TestReadInputFile(t *testing.T) {
	type TestData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	expectedData := TestData{Name: "Test User", Age: 30}
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_input.json")

	fileContent := `{"name":"Test User","age":30}`
	err := os.WriteFile(filePath, []byte(fileContent), 0644)
	if err != nil {
		t.Fatalf("Error writing temporary test file: %v", err)
	}

	var actualData TestData
	ReadInputFile(&actualData, filePath)

	if !reflect.DeepEqual(actualData, expectedData) {
		t.Errorf("ReadInputFile did not parse data correctly.\nExpected: %+v\nGot: %+v", expectedData, actualData)
	}
}

func TestReadInputFile_fileNotFound(t *testing.T) {
	type TestData struct {
		Name string `json:"name"`
	}
	var data TestData
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("ReadInputFile did not panic with non-existent file")
		}
	}()
	ReadInputFile(&data, "non_existent_file.json")
}

func TestMakeInputFile_jsonError(t *testing.T) {
	// Use a channel, which cannot be marshalled to JSON, to trigger an error
	testData := make(chan int)
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_input_error.json")

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("MakeInputFile did not panic with unmarshallable data")
		}
	}()

	// Capture stdout to prevent "made ..." message from interfering with test output
	oldStdout := os.Stdout
	_, w, _ := os.Pipe()
	os.Stdout = w

	MakeInputFile(testData, filePath)

	w.Close()
	os.Stdout = oldStdout // Restore stdout
}
