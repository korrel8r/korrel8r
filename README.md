# Correlation tools for k8s and openshift clusters.

FIXME need explanation: for now see the [source comments](./pkg/korrel8/korrel8.go)

# TODO

- [X] Object: type (type of original), original, attributes
- [X] Object from k8s resource - use JSON
- [X] Object from alert
- [X] Rules & backward chaining
- [X] Initial k8s & prom stores.
- [ ] Need k8s store to take REST URI as query - to express selectors etc.
- [ ] Example of template rule e.g. Service -> Pods
- [ ] Example of full chain: alert -> logs

NOTES:
- Drive backwards from desired result type: Want logs, look for rules giving k8s pods/containers...
- Rules: output and input classes (resources, metrics etc.) output is query!
- Chaining rules: have alert, want logs - ask log-store.correlate(alert)
  -> wants pods so asks k8s-store.correlate(alert) then correlate pods/containers
  -> caching in stores?
- union & intersection of results? Object identity, depends on type or normalized as "namespace/name"...

