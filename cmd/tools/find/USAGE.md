Search for files in a directory hierarchy.

    [PATH] [OPTIONS...]

Options:
    -name PATTERN    match filename against a glob pattern
    -type f|d        filter by file (f) or directory (d)

Examples:
    . -name "*.json"
    /tmp -type d
    . -name "*.go" -type f
