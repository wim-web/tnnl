package update

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

func checksumForAsset(manifest []byte, assetName string) ([sha256.Size]byte, error) {
	type match struct {
		digest string
		line   int
	}

	var matches []match
	scanner := bufio.NewScanner(bytes.NewReader(manifest))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			return [sha256.Size]byte{}, fmt.Errorf("checksum manifest line %d has %d fields; want exactly 2", lineNumber, len(fields))
		}

		filename := strings.TrimPrefix(fields[1], "*")
		if filename == assetName {
			matches = append(matches, match{digest: fields[0], line: lineNumber})
		}
	}
	if err := scanner.Err(); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("scan checksum manifest near line %d: %w", lineNumber+1, err)
	}

	if len(matches) != 1 {
		return [sha256.Size]byte{}, fmt.Errorf("checksum manifest has %d entries for asset %q; want exactly 1", len(matches), assetName)
	}

	entry := matches[0]
	expectedLength := hex.EncodedLen(sha256.Size)
	if len(entry.digest) != expectedLength {
		return [sha256.Size]byte{}, fmt.Errorf("checksum for asset %q on line %d has %d hex characters; want %d", assetName, entry.line, len(entry.digest), expectedLength)
	}

	var checksum [sha256.Size]byte
	if _, err := hex.Decode(checksum[:], []byte(entry.digest)); err != nil {
		return [sha256.Size]byte{}, fmt.Errorf("decode checksum for asset %q on line %d as hexadecimal: %w", assetName, entry.line, err)
	}

	return checksum, nil
}

func verifyFileSHA256(path string, want [sha256.Size]byte) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file for checksum %q: %w", path, err)
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return fmt.Errorf("hash file %q: %w", path, err)
	}

	if subtle.ConstantTimeCompare(hasher.Sum(nil), want[:]) != 1 {
		return fmt.Errorf("checksum mismatch for %s", path)
	}

	return nil
}
