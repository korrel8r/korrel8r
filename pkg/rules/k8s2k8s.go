package rules

import (
	"fmt"

	"github.com/alanconway/korrel8/pkg/k8s"
	"github.com/alanconway/korrel8/pkg/korrel8"
	appsv1 "github.com/openshift/api/apps/v1"
	appv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func K8sToK8s() (rules []korrel8.Rule) {
	// need to add schemes for all API types mentioned in rules.
	appsv1.AddToScheme(scheme.Scheme)
	// Rules for resources with a pod selector.
	for _, o := range []client.Object{
		&appsv1.DeploymentConfig{},
		&appv1.DaemonSet{},
		&appv1.Deployment{},
		&appv1.ReplicaSet{},
		&appv1.StatefulSet{},
		&batchv1.Job{},
		&corev1.ReplicationController{},
		&corev1.Service{},
		&policyv1.PodDisruptionBudget{},
	} {
		rules = append(rules, newTemplate(fmt.Sprintf("%TPodSelector", o), k8s.ClassOf(o), k8s.ClassOf(&corev1.Pod{}),
			`/api/v1/namespaces/{{.ObjectMeta.Namespace}}/pods?labelSelector={{$s := ""}}{{range $k,$v := .Spec.Selector.MatchLabels}}{{urlquery $s}}{{$k}}{{urlquery "="}}{{$v}}{{$s = ","}}{{end}}`))
	}
	rules = append(rules,
		newTemplate("EventToPod", k8s.ClassOf(&eventsv1.Event{}), k8s.ClassOf(&corev1.Pod{}),
			// FIXME only applies if involvedObject.kind == Pod, need to check.
			`/api/v1/namespaces/{{.involvedObject.namespace}}/pods/{{.involvedObject.name}}`))
	return rules
}
