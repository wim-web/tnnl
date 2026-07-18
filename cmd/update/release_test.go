package update

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChecksumForAsset(t *testing.T) {
	t.Run("selects exact filename", func(t *testing.T) {
		const (
			arm64Asset = "tnnl_darwin_arm64.tar.gz"
			amd64Asset = "tnnl_darwin_amd64.tar.gz"
		)
		arm64Digest := strings.Repeat("11", sha256.Size)
		amd64Digest := strings.Repeat("22", sha256.Size)
		manifest := []byte(amd64Digest + "  " + amd64Asset + "\n" + arm64Digest + "  " + arm64Asset + "\n")

		got, err := checksumForAsset(manifest, arm64Asset)
		if err != nil {
			t.Fatalf("checksumForAsset() error = %v", err)
		}

		want := digestFromHex(t, arm64Digest)
		if got != want {
			t.Fatalf("checksumForAsset() = %x, want %x", got, want)
		}
	})

	t.Run("does not match filename that merely ends with asset name", func(t *testing.T) {
		const assetName = "tnnl_darwin_arm64.tar.gz"
		manifest := []byte(strings.Repeat("33", sha256.Size) + "  prefix-" + assetName + "\n")

		_, err := checksumForAsset(manifest, assetName)
		requireErrorContains(t, err, assetName, "0")
	})

	t.Run("rejects missing exact entry", func(t *testing.T) {
		const assetName = "tnnl_linux_arm64.tar.gz"
		manifest := []byte(strings.Repeat("44", sha256.Size) + "  tnnl_linux_amd64.tar.gz\n")

		_, err := checksumForAsset(manifest, assetName)
		requireErrorContains(t, err, assetName, "0")
	})

	t.Run("rejects duplicate exact entries", func(t *testing.T) {
		const assetName = "tnnl_linux_arm64.tar.gz"
		manifest := []byte(
			strings.Repeat("55", sha256.Size) + "  " + assetName + "\n" +
				strings.Repeat("66", sha256.Size) + "  " + assetName + "\n",
		)

		_, err := checksumForAsset(manifest, assetName)
		requireErrorContains(t, err, assetName, "2")
	})

	t.Run("rejects malformed nonblank line", func(t *testing.T) {
		const assetName = "tnnl_linux_arm64.tar.gz"
		manifest := []byte(
			strings.Repeat("77", sha256.Size) + "  " + assetName + "\n" +
				"unexpected extra fields here\n",
		)

		_, err := checksumForAsset(manifest, assetName)
		requireErrorContains(t, err, "line 2", "4")
	})

	t.Run("rejects matching digest with wrong length", func(t *testing.T) {
		const assetName = "tnnl_linux_arm64.tar.gz"
		manifest := []byte(strings.Repeat("88", sha256.Size-1) + "  " + assetName + "\n")

		_, err := checksumForAsset(manifest, assetName)
		requireErrorContains(t, err, assetName, "line 1", "want 64")
	})

	t.Run("rejects matching nonhex digest", func(t *testing.T) {
		const assetName = "tnnl_linux_arm64.tar.gz"
		manifest := []byte(strings.Repeat("z", hex.EncodedLen(sha256.Size)) + "  " + assetName + "\n")

		_, err := checksumForAsset(manifest, assetName)
		requireErrorContains(t, err, assetName, "line 1", "hex")
	})

	t.Run("allows blank lines", func(t *testing.T) {
		const assetName = "tnnl_linux_arm64.tar.gz"
		digest := strings.Repeat("99", sha256.Size)
		manifest := []byte("\n \t\n" + digest + "  " + assetName + "\n\n")

		got, err := checksumForAsset(manifest, assetName)
		if err != nil {
			t.Fatalf("checksumForAsset() error = %v", err)
		}

		want := digestFromHex(t, digest)
		if got != want {
			t.Fatalf("checksumForAsset() = %x, want %x", got, want)
		}
	})

	t.Run("accepts GNU binary marker for exact filename", func(t *testing.T) {
		const assetName = "tnnl_linux_arm64.tar.gz"
		digest := strings.Repeat("aa", sha256.Size)
		manifest := []byte(digest + " *" + assetName + "\n")

		got, err := checksumForAsset(manifest, assetName)
		if err != nil {
			t.Fatalf("checksumForAsset() error = %v", err)
		}

		want := digestFromHex(t, digest)
		if got != want {
			t.Fatalf("checksumForAsset() = %x, want %x", got, want)
		}
	})
}

func TestVerifyFileSHA256(t *testing.T) {
	t.Run("accepts matching file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "archive.tar.gz")
		contents := []byte("release archive contents")
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		want := sha256.Sum256(contents)

		if err := verifyFileSHA256(path, want); err != nil {
			t.Fatalf("verifyFileSHA256() error = %v", err)
		}
	})

	t.Run("rejects one byte difference", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "archive.tar.gz")
		if err := os.WriteFile(path, []byte("release archive contents"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		want := sha256.Sum256([]byte("release archive contentt"))

		err := verifyFileSHA256(path, want)
		requireErrorContains(t, err, "checksum mismatch", path)
	})

	t.Run("reports open error with path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.tar.gz")

		err := verifyFileSHA256(path, sha256.Sum256(nil))
		requireErrorContains(t, err, "open", path)
	})
}

func digestFromHex(t *testing.T, value string) [sha256.Size]byte {
	t.Helper()

	decoded, err := hex.DecodeString(value)
	if err != nil {
		t.Fatalf("DecodeString(%q) error = %v", value, err)
	}

	var digest [sha256.Size]byte
	copy(digest[:], decoded)
	return digest
}

func requireErrorContains(t *testing.T, err error, parts ...string) {
	t.Helper()

	if err == nil {
		t.Fatal("error = nil, want non-nil")
	}
	for _, part := range parts {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error = %q, want it to contain %q", err, part)
		}
	}
}
