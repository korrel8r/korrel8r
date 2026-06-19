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
  -h, --help               help for objects
      --limit int          Limit total number of results.
      --since duration     Only get results since this long ago.
      --timeout duration   Timeout for store requests.
      --until duration     Only get results until this long ago.
```

### Options inherited from parent commands

```
      --blockprofile file   Write block profile to file
  -c, --config string       Configuration file (default "/etc/korrel8r/korrel8r.yaml")
      --cpuprofile file     Write CPU profile to file
      --httpprofile         Enable pprof HTTP endpoints
      --memprofile file     Write memory profile to file
      --mutexprofile file   Write mutex profile to file
  -o, --output string       One of [json json-pretty ndjson yaml] (default "yaml")
      --trace file          Write execution trace to file
  -v, --verbose int         Verbosity for logging (0: notice/error, 1: info/warn, 2: debug, 3: per-request, 4: per-rule, 5: per-query, 9: extra detail
```

