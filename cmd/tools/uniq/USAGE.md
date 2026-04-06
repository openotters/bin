Filter adjacent matching lines.

    [OPTIONS]
    LINE1
    LINE2
    ...

First line may contain options. Remaining lines are the input.

Options:
    -c    prefix lines with occurrence count
    -d    only print duplicated lines
    -u    only print unique lines
    -i    case-insensitive comparison

Examples:
    apple
    apple
    banana

    -c
    hello
    hello
    world
