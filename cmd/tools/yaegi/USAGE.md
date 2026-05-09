Run Go source under the embedded yaegi interpreter.

Source is passed as a SINGLE argv string (use `\n` literals for line
breaks) or piped on stdin. Stdout / stderr / exit code from the
executed program flow back as the tool's response.

Snippets without `package main` are auto-wrapped, with these stdlib
packages already in scope: `fmt`, `os`, `strings`, `strconv`, `sort`,
`math`, `encoding/json`, `regexp`. So one-liners work directly:

    fmt.Println("hello")

    for i := 0; i < 3; i++ { fmt.Println(i) }

    var data map[string]any
    json.Unmarshal([]byte(`{"a":1,"b":2}`), &data)
    fmt.Println(data["a"].(float64) + data["b"].(float64))

Full programs work too — anything starting with `package main` is
passed through verbatim:

    package main

    import "fmt"

    func main() {
        fmt.Println("from a full program")
    }

Limits: stdlib only (no module fetches, no cgo). Each invocation runs
in a fresh interpreter — no state persists between calls.
