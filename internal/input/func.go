package input

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func MakeInputFile(skeleton any, path string, force bool) error {
	jsonData, err := json.MarshalIndent(skeleton, "", "  ")
	if err != nil {
		return fmt.Errorf("encode input file %q: %w", path, err)
	}
	jsonData = append(jsonData, '\n')

	flags := os.O_CREATE | os.O_WRONLY
	if force {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_EXCL
	}

	file, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		return fmt.Errorf("create input file %q: %w", path, err)
	}

	written, err := file.Write(jsonData)
	if err == nil && written != len(jsonData) {
		err = io.ErrShortWrite
	}
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("write input file %q: %w", path, err)
	}

	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("sync input file %q: %w", path, err)
	}

	if err := file.Close(); err != nil {
		return fmt.Errorf("close input file %q: %w", path, err)
	}

	return nil
}

func ReadInputFile(v any, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open input file %q: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("decode input file %q: %w", path, err)
	}

	var extra any
	err = decoder.Decode(&extra)
	if err == nil {
		return fmt.Errorf("decode input file %q: expected exactly one JSON document", path)
	}
	if err != io.EOF {
		return fmt.Errorf("decode input file %q: %w", path, err)
	}

	return nil
}
