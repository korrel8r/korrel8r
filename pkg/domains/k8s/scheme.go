// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package k8s

import (
	appsv1 "github.com/openshift/api/apps/v1"
	oauthv1 "github.com/openshift/api/oauth/v1"
	routev1 "github.com/openshift/api/route/v1"
	securityv1 "github.com/openshift/api/security/v1"
	userv1 "github.com/openshift/api/user/v1"
	operators "github.com/operator-framework/api/pkg/operators/v1alpha1"
	storagev1 "k8s.io/api/storage/v1"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

var Scheme = apiruntime.NewScheme()

// TODO extend this set, consider using discover or dynamic objects?
func init() {
	runtime.Must(scheme.AddToScheme(Scheme))
	runtime.Must(appsv1.AddToScheme(Scheme))
	runtime.Must(oauthv1.AddToScheme(Scheme))
	runtime.Must(routev1.AddToScheme(Scheme))
	runtime.Must(securityv1.AddToScheme(Scheme))
	runtime.Must(storagev1.AddToScheme(Scheme))
	runtime.Must(userv1.AddToScheme(Scheme))
	runtime.Must(operators.AddToScheme(Scheme))
}
