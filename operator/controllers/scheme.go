// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package controllers

import (
	"errors"

	korrel8rv1alpha1 "github.com/korrel8r/korrel8r/operator/apis/korrel8r/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

// AddToScheme adds all types needed by controllers.
func AddToScheme(s *runtime.Scheme) error {
	return errors.Join(
		scheme.AddToScheme(s),
		korrel8rv1alpha1.AddToScheme(s),
		routev1.AddToScheme(s),
	)
	//+kubebuilder:scaffold:scheme
}
