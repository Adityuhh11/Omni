package secrets

import (
	"errors"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	advapi     = windows.NewLazySystemDLL("advapi32.dll")
	credWrite  = advapi.NewProc("CredWriteW")
	credRead   = advapi.NewProc("CredReadW")
	credDelete = advapi.NewProc("CredDeleteW")
	credFree   = advapi.NewProc("CredFree")
)

const (
	credTypeGeneric      = 1
	credPersistLocal     = 2 // kept for this user on this computer, not roamed
	errorNotFound        = windows.Errno(1168)
	maxCredentialBytes   = 5 * 512
	credentialTargetForm = "%s/%s"
)

// credential is CREDENTIALW.
type credential struct {
	Flags, Type    uint32
	TargetName     *uint16
	Comment        *uint16
	LastWritten    windows.Filetime
	BlobSize       uint32
	Blob           *byte
	Persist        uint32
	AttributeCount uint32
	Attributes     uintptr
	TargetAlias    *uint16
	UserName       *uint16
}

func target(name string) (*uint16, error) {
	return windows.UTF16PtrFromString(fmt.Sprintf(credentialTargetForm, service, name))
}

// Get returns the secret stored under name, or ErrNotFound.
func Get(name string) (string, error) {
	t, err := target(name)
	if err != nil {
		return "", fmt.Errorf("secret name %q: %w", name, err)
	}
	var c *credential
	if r, _, err := credRead.Call(uintptr(unsafe.Pointer(t)), credTypeGeneric, 0, uintptr(unsafe.Pointer(&c))); r == 0 {
		if errors.Is(err, errorNotFound) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("read %q from the credential store: %w", name, err)
	}
	defer credFree.Call(uintptr(unsafe.Pointer(c)))
	return string(unsafe.Slice(c.Blob, c.BlobSize)), nil
}

// Set stores value under name, replacing what was there. An empty value deletes the entry.
func Set(name, value string) error {
	if value == "" {
		return Delete(name)
	}
	if len(value) > maxCredentialBytes {
		return fmt.Errorf("a secret may be at most %d bytes", maxCredentialBytes)
	}
	t, err := target(name)
	if err != nil {
		return fmt.Errorf("secret name %q: %w", name, err)
	}
	user, _ := windows.UTF16PtrFromString("Omni")
	blob := []byte(value)
	c := credential{Type: credTypeGeneric, TargetName: t, UserName: user, BlobSize: uint32(len(blob)), Blob: &blob[0], Persist: credPersistLocal}
	if r, _, err := credWrite.Call(uintptr(unsafe.Pointer(&c)), 0); r == 0 {
		return fmt.Errorf("write %q to the credential store: %w", name, err)
	}
	return nil
}

// Delete removes the secret under name; one that is not there is fine.
func Delete(name string) error {
	t, err := target(name)
	if err != nil {
		return fmt.Errorf("secret name %q: %w", name, err)
	}
	if r, _, err := credDelete.Call(uintptr(unsafe.Pointer(t)), credTypeGeneric, 0); r == 0 && !errors.Is(err, errorNotFound) {
		return fmt.Errorf("delete %q from the credential store: %w", name, err)
	}
	return nil
}
