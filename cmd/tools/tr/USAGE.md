Translate or delete characters.

To translate: three sections separated by newlines.
    Line 1: characters to replace (SET1)
    Line 2: replacement characters (SET2)
    Line 3+: input text

To delete: use -d flag.
    Line 1: -d
    Line 2: characters to delete (SET1)
    Line 3+: input text

Examples (translate):
    abc
    ABC
    a lazy cat

Examples (delete):
    -d
    aeiou
    hello world
