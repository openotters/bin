Create hard or symbolic links.

    [-sf] TARGET [LINK]

Options:
    -s    create a symbolic link instead of a hard link
    -f    remove existing destination files

If LINK is omitted, a link is created in the current directory
with the same name as TARGET.

Examples:
    -s /usr/bin/python3 python
    -sf ../config.yaml config.yaml
    target.txt link.txt
