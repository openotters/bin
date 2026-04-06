Process JSON with jq expressions.

First line is the jq expression.
Remaining lines are the JSON input.

Examples:
    .name
    {"name": "Lyon", "country": "France"}

    .[] | select(.age > 30)
    [{"name":"Alice","age":25},{"name":"Bob","age":35}]
