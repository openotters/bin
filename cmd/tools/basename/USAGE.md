Return the filename portion of a path, optionally removing a suffix.

    NAME           → basename of the path
    NAME SUFFIX    → basename with SUFFIX stripped

Examples:
    /path/to/file.txt       → file.txt
    /path/to/file.txt .txt  → file
    .h .h                   → .h
