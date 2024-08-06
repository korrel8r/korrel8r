// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

// package openshift provides tools for interacting with OpenShift clusters.
//
// - Convert between OpenShift console URL and korrel8r Query.
// - Locate default observability services deployed in an OpenShift cluster.
package openshift

import (
	"context"
	"fmt"
	"net"

	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Observability namespaces
const (
	OpenshiftLogging    = "openshift-logging"
	LoggingLoki         = "logging-loki"
	OpenshiftConsole    = "openshift-console"
	OpenshiftMonitoring = "openshift-monitoring"
	ThanosQuerier       = "thanos-querier"
	AlertmanagerMain    = "alertmanager-main"
)

// Observability service namespace-names
var (
	LokiStackNSName        = NamespacedName(OpenshiftLogging, LoggingLoki)
	ConsoleNSName          = NamespacedName(OpenshiftConsole, "console")
	ThanosQuerierNSName    = NamespacedName(OpenshiftMonitoring, ThanosQuerier)
	PrometheusK8sName      = NamespacedName(OpenshiftMonitoring, "prometheus-k8s")
	AlertmanagerMainNSName = NamespacedName(OpenshiftMonitoring, AlertmanagerMain)
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

func ServiceHost(ctx context.Context, c client.Client, nn types.NamespacedName) (string, error) {
	host := fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace)
	_, err := net.DefaultResolver.LookupHost(ctx, host)
	return host, err
}
