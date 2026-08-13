#!/bin/sh

set -e

# Load the AppArmor profile when AppArmor is available and enabled.
profile=/etc/apparmor.d/usr.bin.tmus

profile_disabled() {
	[ -e /etc/apparmor.d/disable/usr.bin.tmus ] \
		|| [ -L /etc/apparmor.d/disable/usr.bin.tmus ] \
		|| [ -e /etc/apparmor.d/disabled/usr.bin.tmus ] \
		|| [ -L /etc/apparmor.d/disabled/usr.bin.tmus ]
}

if command -v aa-enabled >/dev/null 2>&1 \
	&& command -v apparmor_parser >/dev/null 2>&1 \
	&& aa-enabled --quiet \
	&& ! profile_disabled; then
	if ! apparmor_parser --replace "$profile"; then
		echo "tmus: warning: failed to load AppArmor profile $profile" >&2
	fi
fi

exit 0
