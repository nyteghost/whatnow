package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "secrets.json")
	if err := UseFile(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(UseKeychain)

	if _, err := Get("anthropic"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get before Set: err = %v, want ErrNotFound", err)
	}
	if err := Set("anthropic", "sk-1"); err != nil {
		t.Fatal(err)
	}
	if err := Set("integration/jira/token", "tok"); err != nil {
		t.Fatal(err)
	}
	if v, err := Get("anthropic"); err != nil || v != "sk-1" {
		t.Fatalf("Get = %q, %v; want sk-1", v, err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file mode = %o, want 600", mode)
	}

	// A fresh store on the same path sees what the first one saved.
	if err := UseFile(path); err != nil {
		t.Fatal(err)
	}
	if v, err := Get("integration/jira/token"); err != nil || v != "tok" {
		t.Fatalf("Get after reopen = %q, %v; want tok", v, err)
	}

	if err := Delete("anthropic"); err != nil {
		t.Fatal(err)
	}
	if err := Delete("anthropic"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete: err = %v, want ErrNotFound", err)
	}
	if _, err := Get("anthropic"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after Delete: err = %v, want ErrNotFound", err)
	}
}

func TestFileRejectsCorruptFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UseFile(path); err == nil {
		UseKeychain()
		t.Fatal("UseFile on a corrupt file succeeded")
	}
}

func TestKeychainIsDefault(t *testing.T) {
	keyring.MockInit()
	if err := Set("openai", "sk-2"); err != nil {
		t.Fatal(err)
	}
	if v, err := keyring.Get(Service, "openai"); err != nil || v != "sk-2" {
		t.Fatalf("keyring.Get = %q, %v; want sk-2", v, err)
	}
}

func TestFromEnv(t *testing.T) {
	t.Setenv(FileEnv, "")
	if where, err := FromEnv(); err != nil || where != "the OS keychain" {
		t.Fatalf("FromEnv without env = %q, %v", where, err)
	}
	path := filepath.Join(t.TempDir(), "s.json")
	t.Setenv(FileEnv, path)
	t.Cleanup(UseKeychain)
	if where, err := FromEnv(); err != nil || where != path {
		t.Fatalf("FromEnv = %q, %v; want %q", where, err, path)
	}
}
