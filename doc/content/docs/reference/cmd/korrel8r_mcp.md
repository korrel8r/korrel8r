---
title: korrel8r mcp
---
<!-- Generated content, do not edit! -->
## korrel8r mcp

MCP stdio server

### Synopsis

Run korrel8r as an MCP server communicating via stdin/stdout.
Allows korrel8r to be run as a sub-process by an MCP tool.
For a HTTP streaming server use the 'web' command with the '--mcp' flag.


```
korrel8r mcp [flags]
```

### Options

```
  -h, --help   help for mcp
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

