//go:build windows

package ipc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestWindowsRuntimeSecurity(t *testing.T) {
	const userSID = "S-1-5-21-1-2-3-1001"
	user, err := windows.StringToSid(userSID)
	require.NoError(t, err)
	const userGrant = "(A;OICI;FA;;;" + userSID + ")"
	tests := []struct {
		name string
		sddl string
		want string
	}{
		{name: "private", sddl: "O:" + userSID + "D:P" + userGrant},
		{name: "system and administrators", sddl: "O:" + userSID + "D:P" + userGrant + "(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)"},
		{name: "administrator owner", sddl: "O:BAD:P" + userGrant},
		{name: "inherited private grants", sddl: "O:" + userSID + "D:AI(A;OICIID;FA;;;" + userSID + ")"},
		{name: "foreign owner", sddl: "O:S-1-5-21-1-2-3-1002D:P" + userGrant, want: "untrusted owner"},
		{name: "null DACL", sddl: "O:" + userSID + "D:NO_ACCESS_CONTROL", want: "restrictive DACL"},
		{name: "missing DACL", sddl: "O:" + userSID, want: "read DACL"},
		{name: "empty DACL", sddl: "O:" + userSID + "D:P", want: "inheritable read/write"},
		{name: "everyone read", sddl: "O:" + userSID + "D:P" + userGrant + "(A;OICI;FR;;;WD)", want: "another user or group"},
		{name: "users write", sddl: "O:" + userSID + "D:P" + userGrant + "(A;OICI;FW;;;BU)", want: "another user or group"},
		{name: "deny entry", sddl: "O:" + userSID + "D:P(D;;FW;;;WD)" + userGrant, want: "unsupported entry"},
		{name: "no inheritance", sddl: "O:" + userSID + "D:P(A;;FA;;;" + userSID + ")", want: "inheritable read/write"},
		{name: "inherit only", sddl: "O:" + userSID + "D:P(A;OICIIO;FA;;;" + userSID + ")", want: "inheritable read/write"},
		{name: "no propagation", sddl: "O:" + userSID + "D:P(A;OICINP;FA;;;" + userSID + ")", want: "inheritable read/write"},
		{name: "read only", sddl: "O:" + userSID + "D:P(A;OICI;FR;;;" + userSID + ")", want: "inheritable read/write"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd, err := windows.SecurityDescriptorFromString(tt.sddl)
			require.NoError(t, err)
			err = validateWindowsRuntimeSecurity(sd, user)
			if tt.want == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.want)
			}
		})
	}
}

func TestPrepareWindowsRuntimeDir(t *testing.T) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	require.NoError(t, err)
	sd, err := windowsRuntimeSecurity(user.User.Sid)
	require.NoError(t, err)
	newBase := func(t *testing.T) string {
		t.Helper()
		base := filepath.Join(t.TempDir(), "base")
		require.NoError(t, createWindowsRuntimeDir(base, sd))
		return base
	}

	t.Run("creates private directories and inheritable socket permissions", func(t *testing.T) {
		base := newBase(t)
		dir := filepath.Join(base, "tmus", "run")
		require.NoError(t, prepareRuntimeDir(dir))
		require.NoError(t, prepareRuntimeDir(dir))
		// A regular file exercises the same file ACL inheritance as a new
		// socket, without needing AF_UNIX support on this Windows version.
		file := filepath.Join(dir, "inherited")
		require.NoError(t, os.WriteFile(file, nil, 0o600))
		for _, path := range []string{filepath.Dir(dir), dir, file} {
			actual, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
				windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
			require.NoError(t, err)
			owner, _, err := actual.Owner()
			require.NoError(t, err)
			if path != file {
				assert.True(t, owner.Equals(user.User.Sid))
				assert.NoError(t, validateWindowsRuntimeSecurity(actual, user.User.Sid))
			} else {
				// Windows may assign the Administrators owner when elevated.
				assert.True(t, owner.Equals(user.User.Sid) || owner.IsWellKnown(windows.WinBuiltinAdministratorsSid))
				acl, _, err := actual.DACL()
				require.NoError(t, err)
				require.NotNil(t, acl)
				assert.EqualValues(t, 3, acl.AceCount)
				for i := uint16(0); i < acl.AceCount; i++ {
					var ace *windows.ACCESS_ALLOWED_ACE
					require.NoError(t, windows.GetAce(acl, uint32(i), &ace))
					require.EqualValues(t, windows.ACCESS_ALLOWED_ACE_TYPE, ace.Header.AceType)
					sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
					assert.True(t, sid.Equals(user.User.Sid) || sid.IsWellKnown(windows.WinLocalSystemSid) ||
						sid.IsWellKnown(windows.WinBuiltinAdministratorsSid))
					assert.NotZero(t, ace.Header.AceFlags&windows.INHERITED_ACE)
				}
			}
		}
	})

	for _, component := range []string{"base", "tmus", "run"} {
		t.Run("rejects permissive "+component, func(t *testing.T) {
			base := newBase(t)
			dir := filepath.Join(base, "tmus", "run")
			require.NoError(t, prepareRuntimeDir(dir))
			path := map[string]string{"base": base, "tmus": filepath.Dir(dir), "run": dir}[component]
			broad, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")(A;OICI;FR;;;WD)")
			require.NoError(t, err)
			acl, _, err := broad.DACL()
			require.NoError(t, err)
			require.NoError(t, windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT,
				windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil))
			before, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
			require.NoError(t, err)
			assert.ErrorContains(t, prepareRuntimeDir(dir), "another user or group")
			after, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
			require.NoError(t, err)
			assert.Equal(t, before.String(), after.String(), "existing permissions must not be rewritten")
		})

		t.Run("rejects redirected "+component, func(t *testing.T) {
			base := newBase(t)
			dir := filepath.Join(base, "tmus", "run")
			path := filepath.Dir(dir)
			switch component {
			case "base":
				path = filepath.Join(t.TempDir(), "redirected")
				dir = filepath.Join(path, "tmus", "run")
			case "run":
				require.NoError(t, createWindowsRuntimeDir(filepath.Dir(dir), sd))
				path = dir
			}
			target := t.TempDir()
			err := os.Symlink(target, path)
			if errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD) {
				t.Skip("creating directory symlinks requires Windows Developer Mode or symlink privilege")
			}
			require.NoError(t, err)
			assert.ErrorContains(t, prepareRuntimeDir(dir), "reparse point")
			entries, err := os.ReadDir(target)
			require.NoError(t, err)
			assert.Empty(t, entries, "validation must not create children through the redirect")
		})
	}

	t.Run("rejects a regular file", func(t *testing.T) {
		base := newBase(t)
		require.NoError(t, os.WriteFile(filepath.Join(base, "tmus"), nil, 0o600))
		assert.ErrorContains(t, prepareRuntimeDir(filepath.Join(base, "tmus", "run")), "not a directory")
	})

	t.Run("directory cannot be replaced while open", func(t *testing.T) {
		base := newBase(t)
		h, err := openWindowsRuntimeDir(base)
		require.NoError(t, err)
		closed := false
		defer func() {
			if !closed {
				_ = windows.CloseHandle(h)
			}
		}()
		replacement := base + "-renamed"
		assert.Error(t, os.Rename(base, replacement))
		require.NoError(t, windows.CloseHandle(h))
		closed = true
		assert.NoError(t, os.Rename(base, replacement))
	})
}
