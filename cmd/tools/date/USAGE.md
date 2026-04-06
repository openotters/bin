Print the current UTC date and time.

No input returns RFC3339 format.
Prefix with + to use POSIX-style format specifiers.

Format specifiers:
    %Y year (2006)    %m month (01)     %d day (02)
    %H hour (15)      %M minute (04)    %S second (05)
    %a weekday (Mon)  %A weekday (Monday)
    %b month (Jan)    %B month (January)
    %Z timezone (MST) %z offset (-0700)

Examples:
    (empty)         → 2026-04-05T22:30:00Z
    +%Y-%m-%d       → 2026-04-05
    +%H:%M          → 22:30
