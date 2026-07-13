// Copyright: This file is part of korrel8r, released under https://github.com/korrel8r/korrel8r/blob/main/LICENSE

package rules_test

import (
	"testing"

	"github.com/korrel8r/korrel8r/pkg/domains/alert"
	"github.com/korrel8r/korrel8r/pkg/domains/k8s"
)

func TestKubevirtRules(t *testing.T) {
	for _, x := range []ruleTest{
		{
			rule:  "VmToVmi",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "vm-name", nil),
			want:  []string{`k8s:VirtualMachineInstance.v1.kubevirt.io:{"namespace":"vm-ns","name":"vm-name"}`},
		},
		{
			rule:  "VmiToPod",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "vm-name", nil),
			want:  []string{`k8s:Pod.v1:{"namespace":"vm-ns","labels":{"kubevirt.io":"virt-launcher","vm.kubevirt.io/name":"vm-name"}}`},
		},
		{
			rule: "VmiToNode",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "vm-name", k8s.Object{
				"status": k8s.Object{
					"nodeName": "worker-1",
				},
			}),
			want: []string{`k8s:Node.v1:{"name":"worker-1"}`},
		},
		{
			rule: "VmToPVC",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "vm-name", k8s.Object{
				"spec": k8s.Object{
					"dataVolumeTemplates": []k8s.Object{
						{"metadata": k8s.Object{"name": "dv1"}},
						{"metadata": k8s.Object{"name": "dv2"}},
					},
				},
			}),
			want: []string{
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"dv1"}`,
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"dv2"}`,
			},
		},
		{
			rule:  "AlertToVM",
			start: &alert.Object{Labels: map[string]string{"namespace": "vm-ns", "name": "my-vm"}},
			want:  []string{`k8s:VirtualMachine.v1.kubevirt.io:{"namespace":"vm-ns","name":"my-vm"}`},
		},
		{
			rule:  "VmToAlert",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", nil),
			want:  []string{`alert:alert:{"namespace":"vm-ns","name":"my-vm"}`},
		},
		{
			rule:  "AlertToVMI",
			start: &alert.Object{Labels: map[string]string{"namespace": "vm-ns", "name": "my-vmi"}},
			want:  []string{`k8s:VirtualMachineInstance.v1.kubevirt.io:{"namespace":"vm-ns","name":"my-vmi"}`},
		},
		{
			rule:  "VmiToAlert",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", nil),
			want:  []string{`alert:alert:{"namespace":"vm-ns","name":"my-vmi"}`},
		},
		{
			rule:  "AlertToVmim",
			start: &alert.Object{Labels: map[string]string{"namespace": "vm-ns", "vmim": "my-migration"}},
			want:  []string{`k8s:VirtualMachineInstanceMigration.v1.kubevirt.io:{"namespace":"vm-ns","name":"my-migration"}`},
		},
		{
			rule:  "VmimToAlert",
			start: newK8s("VirtualMachineInstanceMigration.kubevirt.io", "vm-ns", "my-migration", nil),
			want:  []string{`alert:alert:{"namespace":"vm-ns","vmim":"my-migration"}`},
		},
		{
			rule:  "VmToMetric",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", nil),
			want:  []string{`metric:metric:{namespace="vm-ns",name="my-vm"}`},
		},
		{
			rule:  "VmiToMetric",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", nil),
			want:  []string{`metric:metric:{namespace="vm-ns",name="my-vmi"}`},
		},
		{
			rule:  "VmiToLogs",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", nil),
			want:  []string{`log:application:{"namespace":"vm-ns","labels":{"kubevirt.io":"virt-launcher","vm.kubevirt.io/name":"my-vmi"}}`},
		},
		{
			rule: "VmimToVmi",
			start: newK8s("VirtualMachineInstanceMigration.kubevirt.io", "vm-ns", "mig-1", k8s.Object{
				"spec": k8s.Object{
					"vmiName": "my-vmi",
				},
			}),
			want: []string{`k8s:VirtualMachineInstance.v1.kubevirt.io:{"namespace":"vm-ns","name":"my-vmi"}`},
		},
		{
			rule:  "VmiToVmim",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", nil),
			want:  []string{`k8s:VirtualMachineInstanceMigration.v1.kubevirt.io:{"namespace":"vm-ns","labels":{"kubevirt.io/vmi-name":"my-vmi"}}`},
		},
		{
			rule:  "DataVolumeToPVC",
			start: newK8s("DataVolume.cdi.kubevirt.io", "vm-ns", "my-dv", nil),
			want:  []string{`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"my-dv"}`},
		},
		{
			rule: "VmToDataVolume",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"dataVolumeTemplates": []k8s.Object{
						{"metadata": k8s.Object{"name": "dv1"}},
					},
				},
			}),
			want: []string{`k8s:DataVolume.v1beta1.cdi.kubevirt.io:{"namespace":"vm-ns","name":"dv1"}`},
		},
		{
			rule: "VmSnapshotToVm",
			start: newK8s("VirtualMachineSnapshot.snapshot.kubevirt.io", "vm-ns", "snap-1", k8s.Object{
				"spec": k8s.Object{
					"source": k8s.Object{"name": "my-vm"},
				},
			}),
			want: []string{`k8s:VirtualMachine.v1.kubevirt.io:{"namespace":"vm-ns","name":"my-vm"}`},
		},
		{
			rule:  "VmToVmSnapshot",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", nil),
			want:  []string{`k8s:VirtualMachineSnapshot.v1beta1.snapshot.kubevirt.io:{"namespace":"vm-ns","labels":{"vm.kubevirt.io/name":"my-vm"}}`},
		},
		{
			rule: "VmRestoreToVm",
			start: newK8s("VirtualMachineRestore.snapshot.kubevirt.io", "vm-ns", "restore-1", k8s.Object{
				"spec": k8s.Object{
					"target": k8s.Object{"name": "my-vm"},
				},
			}),
			want: []string{`k8s:VirtualMachine.v1.kubevirt.io:{"namespace":"vm-ns","name":"my-vm"}`},
		},
		{
			rule:  "VmToVmRestore",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", nil),
			want:  []string{`k8s:VirtualMachineRestore.v1beta1.snapshot.kubevirt.io:{"namespace":"vm-ns","labels":{"vm.kubevirt.io/name":"my-vm"}}`},
		},
		{
			rule: "VmRestoreToVmSnapshot",
			start: newK8s("VirtualMachineRestore.snapshot.kubevirt.io", "vm-ns", "restore-1", k8s.Object{
				"spec": k8s.Object{
					"virtualMachineSnapshotName": "snap-1",
				},
			}),
			want: []string{`k8s:VirtualMachineSnapshot.v1beta1.snapshot.kubevirt.io:{"namespace":"vm-ns","name":"snap-1"}`},
		},
		{
			rule: "VmExportToVm",
			start: newK8s("VirtualMachineExport.export.kubevirt.io", "vm-ns", "export-1", k8s.Object{
				"spec": k8s.Object{
					"source": k8s.Object{"kind": "VirtualMachine", "name": "my-vm"},
				},
			}),
			want: []string{`k8s:VirtualMachine.v1.kubevirt.io:{"namespace":"vm-ns","name":"my-vm"}`},
		},
		{
			rule: "VmExportToVmSnapshot",
			start: newK8s("VirtualMachineExport.export.kubevirt.io", "vm-ns", "export-1", k8s.Object{
				"spec": k8s.Object{
					"source": k8s.Object{"kind": "VirtualMachineSnapshot", "name": "snap-1"},
				},
			}),
			want: []string{`k8s:VirtualMachineSnapshot.v1beta1.snapshot.kubevirt.io:{"namespace":"vm-ns","name":"snap-1"}`},
		},
		{
			rule: "VmExportToPVC",
			start: newK8s("VirtualMachineExport.export.kubevirt.io", "vm-ns", "export-1", k8s.Object{
				"spec": k8s.Object{
					"source": k8s.Object{"kind": "PersistentVolumeClaim", "name": "my-pvc"},
				},
			}),
			want: []string{`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"my-pvc"}`},
		},
		{
			rule:  "NodeToVmi",
			start: newK8s("Node", "", "worker-1", nil),
			want:  []string{`k8s:VirtualMachineInstance.v1.kubevirt.io:{"labels":{"kubevirt.io/nodeName":"worker-1"}}`},
		},
		{
			rule:  "DataVolumeToImporterPod",
			start: newK8s("DataVolume.cdi.kubevirt.io", "vm-ns", "my-dv", nil),
			want:  []string{`k8s:Pod.v1:{"namespace":"vm-ns","labels":{"cdi.kubevirt.io/storage.import.importPvcName":"my-dv"}}`},
		},
		{
			rule: "VmiToNetAttachDef",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", k8s.Object{
				"spec": k8s.Object{
					"networks": []k8s.Object{
						{"name": "default", "pod": k8s.Object{}},
						{"name": "net1", "multus": k8s.Object{"networkName": "my-nad"}},
					},
				},
			}),
			want: []string{`k8s:NetworkAttachmentDefinition.v1.k8s.cni.cncf.io:{"namespace":"vm-ns","name":"my-nad"}`},
		},
	} {
		x.Run(t)
	}
}
