package korrel8r

// StoreConfig is a JSON map of name:value pairs to configure a store.
// Some have standard meanings, others are defined on a per-store basis, see [StoreConfigKey]
type StoreConfig map[StoreKey]string

type StoreKey string

const (
	StoreKeyContext StoreKey = "context" // StoreKeyContext is the kube context name for connecting to a cluster
)

// FIXME document, clean up
