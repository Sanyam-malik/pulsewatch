---
sidebar_position: 3
---

# HTTP response conditions

HTTP monitors can define up to 20 response conditions in their JSON `config`.
`condition_operator` combines the rules: `and` (default) requires every rule to
pass; `or` requires at least one. The monitor is checked only after its accepted
HTTP status-code ranges pass.

```json
{
  "url": "https://api.example.com/health",
  "method": "GET",
  "encoding": "json",
  "accepted_statuscodes": ["2XX"],
  "authMethod": "none",
  "condition_operator": "and",
  "conditions": [
    { "type": "status", "operator": "==", "expected_value": "200" },
    { "type": "response_time", "operator": "<", "expected_value": "500" },
    {
      "type": "json",
      "json_query": "data.ready",
      "operator": "==",
      "expected_value": "true"
    }
  ]
}
```

Supported `type` values are:

- `status`: compare the HTTP response status code.
- `response_time`: compare request time to response headers, in milliseconds.
- `json`: evaluate a GJSON path using the comparison operators. `exists` and
  `not_exists` test a path without an expected value. An empty path with `==`
  or `!=` compares the complete JSON value; object key order and whitespace
  do not affect comparison.

Comparison operators are `==`, `!=`, `>`, `<`, `>=`, and `<=`. Numeric values
are compared numerically; non-numeric JSON values use lexical comparison.
Missing paths fail comparisons and `exists`, and pass `not_exists`.

Existing monitors using `json_query`, `json_condition`, and `expected_value`
continue to use their original single-condition behavior when `conditions` is
absent. If an explicit `conditions` list is present, it takes precedence over
those legacy fields. The monitor form imports a saved legacy check as one rule
when it is edited and saved.
