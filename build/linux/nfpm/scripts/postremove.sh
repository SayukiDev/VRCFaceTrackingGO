#!/bin/sh

# Drop the removed .desktop file from the application menus.
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q /usr/share/applications
fi

exit 0
