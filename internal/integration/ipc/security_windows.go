//go:build windows

package ipc

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

func windowsRuntimeSecurity(user *windows.SID) (*windows.SECURITY_DESCRIPTOR, error) {
	// P prevents inherited grants from broadening the DACL. OI/CI propagate
	// these grants to socket files and child directories. Set the owner
	// explicitly so elevated and non-elevated launches use the same policy.
	return windows.SecurityDescriptorFromString("O:" + user.String() +
		"D:P(A;OICI;FA;;;" + user.String() + ")(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)")
}

func createWindowsRuntimeDir(path string, sd *windows.SECURITY_DESCRIPTOR) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	sa := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: sd,
	}
	err = windows.CreateDirectory(p, &sa)
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return nil // The caller must validate the existing object.
	}
	return err
}

// openWindowsRuntimeDir opens the directory itself, including junctions and
// symlinks, so validation never follows the final component. Omitting
// FILE_SHARE_DELETE prevents renaming/removing it while its children are used.
func openWindowsRuntimeDir(path string) (windows.Handle, error) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return windows.InvalidHandle, err
	}
	h, err := windows.CreateFile(p, windows.READ_CONTROL|windows.FILE_READ_ATTRIBUTES,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return windows.InvalidHandle, err
	}
	var info windows.ByHandleFileInformation
	err = windows.GetFileInformationByHandle(h, &info)
	if err == nil {
		switch {
		case info.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0:
			err = errors.New("runtime directory must not be a reparse point")
		case info.FileAttributes&windows.FILE_ATTRIBUTE_DIRECTORY == 0:
			err = errors.New("runtime path is not a directory")
		}
	}
	if err != nil {
		return windows.InvalidHandle, errors.Join(err, windows.CloseHandle(h))
	}
	return h, nil
}

// validateWindowsRuntimeSecurity conservatively accepts only ordinary allow
// entries for the current user, SYSTEM, and Administrators. Unknown ACE types
// are rejected rather than attempting to reproduce Windows access evaluation.
func validateWindowsRuntimeSecurity(sd *windows.SECURITY_DESCRIPTOR, user *windows.SID) error {
	defer runtime.KeepAlive(sd) // ACE and SID pointers below refer into sd.
	trusted := func(sid *windows.SID) bool {
		return sid != nil && (sid.Equals(user) || sid.IsWellKnown(windows.WinLocalSystemSid) ||
			sid.IsWellKnown(windows.WinBuiltinAdministratorsSid))
	}
	owner, _, err := sd.Owner()
	if err != nil {
		return fmt.Errorf("read owner: %w", err)
	}
	if !trusted(owner) {
		return errors.New("runtime directory has an untrusted owner")
	}
	dacl, _, err := sd.DACL()
	if err != nil {
		return fmt.Errorf("read DACL: %w", err)
	}
	if dacl == nil {
		return errors.New("runtime directory must have a restrictive DACL")
	}
	userAccess := false
	for i := uint16(0); i < dacl.AceCount; i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err := windows.GetAce(dacl, uint32(i), &ace); err != nil {
			return fmt.Errorf("read DACL entry: %w", err)
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return errors.New("runtime directory DACL contains an unsupported entry")
		}
		// In an ACCESS_ALLOWED_ACE, SidStart is the first DWORD of a SID
		// whose remaining bytes follow within this variable-sized entry.
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !trusted(sid) {
			return errors.New("runtime directory grants access to another user or group")
		}
		const inherit = windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE
		const readWrite = windows.FILE_GENERIC_READ | windows.FILE_GENERIC_WRITE
		if sid.Equals(user) && ace.Header.AceFlags&inherit == inherit &&
			ace.Header.AceFlags&(windows.INHERIT_ONLY_ACE|windows.NO_PROPAGATE_INHERIT_ACE) == 0 &&
			(uint32(ace.Mask)&readWrite == readWrite || uint32(ace.Mask)&windows.GENERIC_ALL != 0) {
			userAccess = true
		}
	}
	if !userAccess {
		return errors.New("runtime directory must grant the current user inheritable read/write access")
	}
	return nil
}
