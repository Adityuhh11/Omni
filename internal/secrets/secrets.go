// Package secrets keeps API keys in the system credential store (Windows Credential Manager, the macOS Keychain), never in
// config.toml or any other file. A key is found by a short name of Omni's own; the store ties it to the signed-in user.
package secrets

import "errors"

// service groups Omni's entries in the store.
const service = "Omni"

// ErrNotFound is returned by Get when nothing is stored under the name.
var ErrNotFound = errors.New("no such secret")
