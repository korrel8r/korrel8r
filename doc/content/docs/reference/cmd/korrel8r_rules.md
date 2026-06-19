---
title: korrel8r rules
---
<!-- Generated content, do not edit! -->
## korrel8r rules

List rules by start, goal or name

```
korrel8r rules [flags]
```

### Options

```
  -g, --goal string    show rules with this goal class
      --graph          write rule graph in graphviz format
  -h, --help           help for rules
      --long           show rule start and goal classes
  -n, --name string    show rules with name matching this regexp
  -s, --start string   show rules with this start class
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

