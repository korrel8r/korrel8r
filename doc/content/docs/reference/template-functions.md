---
title: Template Functions
description: Functions available in rule templates
weight: 10
---
<!-- Generated content, do not edit! -->
### Template Functions

These functions are available in all rule templates, in addition to the standard [Go template functions](<https://pkg.go.dev/text/template#hdr-Functions>) and the [Sprig](<http://masterminds.github.io/sprig/>) library. Some domains provide additional domain\-specific functions, documented in the domain reference.

```
assert [message] value
    Fails the template if value is empty.
    An optional message string is included in the error.
    A value is "empty" if it is nil, false, zero, or a zero-length string, slice, or map.
    Can also be used with piped syntax: {{ value | assert "message" }}.

required [message] value
    Like assert, but returns the value if it is not empty.
    Fails the template if the value is empty.
    Can also be used with piped syntax: {{ value | required "message" }}.

query queryString
    Executes its argument as a korrel8r query string, returns the results as []any.
    May return an error.

k8sRouteHost namespace name
    Returns the spec.host of the named OpenShift route.
    Returns an error if the route is not found.
```

### Asserting Multiple Values

Use the [Sprig](<http://masterminds.github.io/sprig/>) function "all" to assert that several values are non\-empty. The "all" function returns true only when every argument is non\-empty.

Examples:

```
{{assert (all .metadata.namespace .metadata.name)}}
{{assert "need namespace, name and labels" (all .metadata.namespace .metadata.name .metadata.labels)}}
```

Use "required" to pass through a value while also checking that related values are present:

```
{{required "need namespace" .metadata.namespace}}
```

