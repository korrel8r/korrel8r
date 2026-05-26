---
title: Commands
description: Command-line interface
---
## korrel8r

Correlate observability data in a cluster

### Options

```
  -c, --config string        Configuration file (default "/etc/korrel8r/korrel8r.yaml")
  -h, --help                 help for korrel8r
  -o, --output string        One of [json json-pretty ndjson yaml] (default "yaml")
      --profile string       Enable profiling: One of [alloc block clock cpu goroutine heap mem mutex trace]
      --profilePath string   Output path for profile
  -v, --verbose int          Verbosity for logging (0: notice/error, 1: info/warn, 2: debug, 3: per-request, 4: per-rule, 5: per-query, 9: extra detail
```

### SEE ALSO

* [korrel8r describe](korrel8r_describe.md)	 - Documentation for DOMAIN or for all domains.
* [korrel8r goals](korrel8r_goals.md)	 - Execute QUERY, find all paths to GOAL classes.
* [korrel8r list](korrel8r_list.md)	 - List domains or classes in DOMAIN.
* [korrel8r mcp](korrel8r_mcp.md)	 - MCP stdio server
* [korrel8r neighbors](korrel8r_neighbors.md)	 - Get graph of nearest neighbors
* [korrel8r objects](korrel8r_objects.md)	 - Execute QUERY and print the results
* [korrel8r rules](korrel8r_rules.md)	 - List rules by start, goal or name
* [korrel8r stores](korrel8r_stores.md)	 - List the stores configured for the listed domains, or for all domains if none are listed.
* [korrel8r template](korrel8r_template.md)	 - Apply a Go template to the korrel8r engine.
* [korrel8r version](korrel8r_version.md)	 - Print the version of this command.
* [korrel8r web](korrel8r_web.md)	 - Start REST server. Listening address must be  provided via --http or --https.

