package secrets

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// notFound is the exit status of `security` when an item does not exist.
const notFound = 44

// The `security` program is the Keychain's own command line. A secret is never an argument (any process could read it there):
// it goes in on standard input, as hexadecimal so that no character needs quoting.

// Get returns the secret stored under name, or ErrNotFound.
func Get(name string) (string, error) {
	out, err := exec.Command("security", "find-generic-password", "-s", service, "-a", name, "-w").Output()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit) && exit.ExitCode() == notFound:
		return "", ErrNotFound
	case err != nil:
		return "", fmt.Errorf("read %q from the Keychain: %w", name, err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// Set stores value under name, replacing what was there. An empty value deletes the entry.
func Set(name, value string) error {
	if value == "" {
		return Delete(name)
	}
	cmd := exec.Command("security", "-i")
	cmd.Stdin = strings.NewReader(fmt.Sprintf("add-generic-password -U -s %s -a %s -X %s\n", service, name, hex.EncodeToString([]byte(value))))
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("write %q to the Keychain: %w: %s", name, err, strings.TrimSpace(out.String()))
	}
	// `security -i` reports a refused command on its output, not in its exit status, so what was stored is read back.
	if got, err := Get(name); err != nil || got != value {
		return fmt.Errorf("write %q to the Keychain: not stored: %s", name, strings.TrimSpace(out.String()))
	}
	return nil
}

// Delete removes the secret under name; one that is not there is fine.
func Delete(name string) error {
	err := exec.Command("security", "delete-generic-password", "-s", service, "-a", name).Run()
	var exit *exec.ExitError
	if err != nil && !(errors.As(err, &exit) && exit.ExitCode() == notFound) {
		return fmt.Errorf("delete %q from the Keychain: %w", name, err)
	}
	return nil
}
