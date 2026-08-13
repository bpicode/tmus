#!/bin/sh

set -e

# Unload the AppArmor profile when tmus is removed. Debian passes "remove"
# and RPM passes 0 when no package version remains installed. Other values
# describe upgrades or rollbacks.
case "${1:-}" in
	remove|0)
		if command -v aa-enabled >/dev/null 2>&1 \
			&& command -v apparmor_parser >/dev/null 2>&1 \
			&& aa-enabled --quiet; then
			apparmor_parser --remove /etc/apparmor.d/usr.bin.tmus \
				>/dev/null 2>&1 || true
		fi
		;;
esac

exit 0
