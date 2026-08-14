---
title: korrel8r web
---
<!-- Generated content, do not edit! -->
## korrel8r web

Start REST server. Listening address must be  provided via --http or --https.

```
korrel8r web [flags]
```

### Options

```
      --cert string                 TLS certificate file (PEM format) for https
  -h, --help                        help for web
      --http string                 host:port address for insecure http listener
      --https string                host:port address for secure https listener
      --key string                  Private key (PEM format) for https
      --mcp                         Enable MCP streamable HTTP protocol on /mcp (default true)
      --rest                        Enable HTTP REST server on /api/v1alpha1 (default true)
      --spec string                 Write OpenAPI specification to a file, '-' for stdout.
      --tls-cipher-suites strings   Comma-separated list of TLS cipher suites for https (IANA or OpenSSL names)
      --tls-curves strings          Comma-separated list of TLS curves for https (Go or OpenSSL names, e.g. CurveP256/prime256v1, X25519)
      --tls-min-version string      Minimum TLS version for https (e.g. VersionTLS12, VersionTLS13)
      --unsafe-shared-session       Allow unauthenticated requests to share a single session (UNSAFE: disables per-user isolation)
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

