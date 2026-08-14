---
title: korrel8r template
---
<!-- Generated content, do not edit! -->
## korrel8r template

Apply a Go template to the korrel8r engine.

### Synopsis

Apply a Go template to the korrel8r engine.
Reads stdin by default if neither --file nor --template is provided.
Useful for testing rule and store templates.

```
korrel8r template [--file FILE|--template STRING] [flags]
```

### Options

```
  -f, --file string       read template from file
  -h, --help              help for template
  -t, --template string   use template string
```

### Options inherited from parent commands

```
      --blockprofile file       Write block profile to file
  -c, --config string           Configuration file (default "/etc/korrel8r/korrel8r.yaml")
      --cpuprofile file         Write CPU profile to file
      --httpprofile             Enable pprof HTTP endpoints
      --memprofile file         Write memory profile to file
      --metric-file string      Write metrics to the given file at the end of execution
      --metric-http             Expose an HTTP endpoint for prometheus to collect metrics.
      --mutexprofile file       Write mutex profile to file
      --otel-collector string   URL of OTLP collector endpoint for pushing metrics (e.g. http://localhost:4318/v1/metrics)
  -o, --output string           One of [json json-pretty ndjson yaml] (default "yaml")
      --trace file              Write execution trace to file
  -v, --verbose int             Verbosity for logging (0: notice/error, 1: info/warn, 2: debug, 3: per-request, 4: per-rule, 5: per-query, 9: extra detail
```

