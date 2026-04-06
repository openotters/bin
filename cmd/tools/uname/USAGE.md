Print system information.

    [OPTIONS]

Options:
    -a    print all information
    -s    kernel name (default)
    -n    network node hostname
    -r    kernel release / Go version
    -m    machine architecture
    -p    processor type (same as -m)

Examples:
    (empty)  → darwin
    -a       → darwin myhost go1.26.1 arm64
    -snm     → darwin myhost arm64
