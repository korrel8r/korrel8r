package k8s

import (
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	securityv1 "github.com/openshift/api/security/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// Scheme fro k8s types known to korrel8r.
var Scheme = runtime.NewScheme()

func init() {
	util.Must(clientgoscheme.AddToScheme(Scheme))
	util.Must(routev1.AddToScheme(Scheme))
	util.Must(oauthv1.AddToScheme(Scheme))
	util.Must(securityv1.AddToScheme(Scheme))
	util.Must(appsv1.AddToScheme(Scheme))
}
