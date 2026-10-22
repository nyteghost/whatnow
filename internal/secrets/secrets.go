// Package secrets keeps API keys and integration secrets out of the database. They go in the OS
// keychain by default. Where there is no keychain, as in a container or on a headless server,
// UseFile puts them in a file only the owner can read instead.
package secrets

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/zalando/go-keyring"
)

// Service is the keychain service every whatnow secret is stored under.
const Service = "whatnow"

// ErrNotFound means the secret isn't set. It is keyring's own error, so callers can check either.
var ErrNotFound = keyring.ErrNotFound

// FileEnv names the environment variable that switches storage to a file at its path.
const FileEnv = "WHATNOW_SECRETS_FILE"

type backend interface {
	Get(account string) (string, error)
	Set(account, value string) error
	Delete(account string) error
}

// keychain calls keyring on each use, so keyring.MockInit in tests still takes effect.
type keychain struct{}

func (keychain) Get(account string) (string, error) { return keyring.Get(Service, account) }
func (keychain) Set(account, value string) error    { return keyring.Set(Service, account, value) }
func (keychain) Delete(account string) error        { return keyring.Delete(Service, account) }

var (
	mu      sync.RWMutex
	current backend = keychain{}
)

// Get returns a secret, or ErrNotFound when it isn't set.
func Get(account string) (string, error) {
	mu.RLock()
	defer mu.RUnlock()
	return current.Get(account)
}

// Set saves a secret, replacing any it had.
func Set(account, value string) error {
	mu.RLock()
	defer mu.RUnlock()
	return current.Set(account, value)
}

// Delete removes a secret, or returns ErrNotFound when it isn't set.
func Delete(account string) error {
	mu.RLock()
	defer mu.RUnlock()
	return current.Delete(account)
}

// UseFile stores secrets in a JSON file at path from now on, creating it and its directory as
// needed. The file is the owner's alone (0600), but it is not encrypted: protect it as you would
// an .env file.
func UseFile(path string) error {
	f := &file{path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("secrets file: %w", err)
	}
	if _, err := f.read(); err != nil {
		return err
	}
	mu.Lock()
	current = f
	mu.Unlock()
	return nil
}

// UseKeychain goes back to the OS keychain.
func UseKeychain() {
	mu.Lock()
	current = keychain{}
	mu.Unlock()
}

// FromEnv switches to a file when FileEnv is set, and reports where secrets are kept.
func FromEnv() (string, error) {
	path := os.Getenv(FileEnv)
	if path == "" {
		return "the OS keychain", nil
	}
	if err := UseFile(path); err != nil {
		return "", err
	}
	return path, nil
}

type file struct {
	mu   sync.Mutex
	path string
}

func (f *file) read() (map[string]string, error) {
	body, err := os.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("secrets file: %w", err)
	}
	m := map[string]string{}
	if len(body) == 0 {
		return m, nil
	}
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("secrets file %s: %w", f.path, err)
	}
	return m, nil
}

// write replaces the file in one step, so a crash leaves the old secrets or the new ones.
func (f *file) write(m map[string]string) error {
	body, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(f.path), ".secrets-*")
	if err != nil {
		return fmt.Errorf("secrets file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return fmt.Errorf("secrets file: %w", err)
	}
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		return fmt.Errorf("secrets file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("secrets file: %w", err)
	}
	if err := os.Rename(tmp.Name(), f.path); err != nil {
		return fmt.Errorf("secrets file: %w", err)
	}
	return nil
}

func (f *file) Get(account string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return "", err
	}
	v, ok := m[account]
	if !ok {
		return "", ErrNotFound
	}
	return v, nil
}

func (f *file) Set(account, value string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return err
	}
	m[account] = value
	return f.write(m)
}

func (f *file) Delete(account string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, err := f.read()
	if err != nil {
		return err
	}
	if _, ok := m[account]; !ok {
		return ErrNotFound
	}
	delete(m, account)
	return f.write(m)
}
