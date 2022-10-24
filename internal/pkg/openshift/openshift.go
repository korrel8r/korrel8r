// package openshift provides contants and functions for accessing an openshift cluster.
package openshift

import (
	"context"

	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OpenshiftLogging = "openshift-logging"
	LoggingLoki      = "logging-loki"
)

var (
	LokiStackNSName = NamespacedName(OpenshiftLogging, LoggingLoki)
)

func init() {
	runtime.Must(routev1.AddToScheme(scheme.Scheme))
}

// NamespacedName constructs a namespaced name
func NamespacedName(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}

// RouteHost gets the host from a route.
func RouteHost(ctx context.Context, c client.Client, nn types.NamespacedName) (string, error) {
	r := &routev1.Route{}
	err := c.Get(ctx, nn, r)
	return r.Spec.Host, err
}
