# Korrel8r Rules

This directory contains YAML configuration files that define additional correlation rules loaded at runtime.
They use Go `text/template` syntax and can be added or changed without rebuilding korrel8r.
Include new files in `all.yaml` to get them picked up by default configurations.

Most rules are compiled into korrel8r as using [quicktemplate](https://github.com/valyala/quicktemplate) functions.
They are faster and type-safe, but require rebuilding the executable.

For rule syntax and examples, see [Writing Rules](../../../doc/content/docs/writing-rules.md)
See `_samples` for samples of old template rule that have been converted to quickrules.
