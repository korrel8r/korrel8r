package openshift

import (
	"context"

	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	MonitoringNS     = "openshift-monitoring"
	AlertmanagerMain = "alertmanager-main"
)

func init() {
	runtime.Must(routev1.AddToScheme(scheme.Scheme))
}

func AlertmanagerMainHost(ctx context.Context, kc client.Client) (string, error) {
	r := routev1.Route{}
	nsName := client.ObjectKey{Name: AlertmanagerMain, Namespace: MonitoringNS}
	if err := kc.Get(context.Background(), nsName, &r); err != nil {
		return "", err
	}
	return r.Spec.Host, nil
}
