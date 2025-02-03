// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/netflow"
)

func Test_NetflowToK8S(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "NetflowToSrcK8s",
			start: netflow.Object{"SrcK8S_Type": "Pod", "SrcK8S_Namespace": "foo", "SrcK8S_Name": "bar"},
			query: `k8s:Pod.v1:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "NetflowToSrcK8sOwner",
			start: netflow.Object{"SrcK8S_OwnerType": "Deployment", "SrcK8S_Namespace": "foo", "SrcK8S_OwnerName": "bar"},
			query: `k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "NetflowToDstK8s",
			start: netflow.Object{"DstK8S_Type": "Pod", "DstK8S_Namespace": "foo", "DstK8S_Name": "bar"},
			query: `k8s:Pod.v1:{"namespace":"foo","name":"bar"}`,
		},
		{
			rule:  "NetflowToDstK8sOwner",
			start: netflow.Object{"DstK8S_OwnerType": "Deployment", "DstK8S_Namespace": "foo", "DstK8S_OwnerName": "bar"},
			query: `k8s:Deployment.v1.apps:{"namespace":"foo","name":"bar"}`,
		},
	} {
		x.Run(t)
	}
}

func Test_NetflowFromK8S(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "K8sSrcToNetflow",
			start: newK8s("Pod", "bar", "foo"),
			query: `netflow:network:{SrcK8S_Type="Pod", SrcK8S_Namespace="bar"} | json | SrcK8S_Name="foo"`,
		},
		{
			rule:  "K8sSrcOwnerToNetflow",
			start: newK8s("Deployment.app", "bar", "foo"),
			query: `netflow:network:{SrcK8S_Namespace="bar", SrcK8S_OwnerName="foo"}`,
		},
		{
			rule:  "K8sDstToNetflow",
			start: newK8s("Pod", "bar", "foo"),
			query: `netflow:network:{DstK8S_Type="Pod", DstK8S_Namespace="bar"} | json | DstK8S_Name="foo"`,
		},
		{
			rule:  "K8sDstOwnerToNetflow",
			start: newK8s("Deployment.app", "bar", "foo"),
			query: `netflow:network:{DstK8S_Namespace="bar", DstK8S_OwnerName="foo"}`,
		},
	} {
		x.Run(t)
	}
}
