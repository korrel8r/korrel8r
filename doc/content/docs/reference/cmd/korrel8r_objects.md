---
title: korrel8r objects
---
<!-- Generated content, do not edit! -->
## korrel8r objects

Execute QUERY and print the results

```
korrel8r objects QUERY [flags]
```

### Options

```
  -h, --help                    help for objects
      --limit int               Limit number of results per query.
      --query-limit int         Limit number of queries per class during traversal.
      --since duration          Only get results since this long ago.
      --timeout duration        Timeout for store requests.
      --total-limit int         Limit unique results retained across the traversal.
      --total-query-limit int   Limit unique queries accepted across the traversal.
      --until duration          Only get results until this long ago.
```

### Options inherited from parent commands

```
      --blockprofile file       Write block profile to file
  -c, --config string           Configuration file (default "/etc/korrel8r/korrel8r.yaml")
      --cpuprofile file         Write CPU profile to file
      --httpprofile             Enable pprof HTTP endpoints
      --memprofile file         Write memory profile to file
      --metric-file string      Write metrics to the given file at the end of execution
      --mutexprofile file       Write mutex profile to file
      --otel-collector string   URL of OTLP collector endpoint for pushing metrics (e.g. http://localhost:4318/v1/metrics)
  -o, --output string           One of [json json-pretty ndjson yaml] (default "yaml")
      --trace file              Write execution trace to file
  -v, --verbose int             Verbosity for logging (0: notice/error, 1: info/warn, 2: debug, 3: per-request, 4: per-rule, 5: per-query, 9: extra detail
```

