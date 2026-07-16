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
						{"metadata": k8s.Object{"name": "template-dv"}},
					},
					"template": k8s.Object{
						"spec": k8s.Object{
							"volumes": []k8s.Object{
								{"name": "rootdisk", "dataVolume": k8s.Object{"name": "preexisting-dv"}},
								{"name": "datadisk", "persistentVolumeClaim": k8s.Object{"claimName": "direct-pvc"}},
							},
						},
					},
				},
			}),
			want: []string{
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"template-dv"}`,
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"preexisting-dv"}`,
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"direct-pvc"}`,
			},
		},
		{
			rule: "VmToPVC",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "vm-name", k8s.Object{
				"spec": k8s.Object{
					"template": k8s.Object{
						"spec": k8s.Object{
							"volumes": []k8s.Object{
								{"name": "ephemeral-vol", "ephemeral": k8s.Object{"persistentVolumeClaim": k8s.Object{"claimName": "eph-pvc"}}},
								{"name": "memdump", "memoryDump": k8s.Object{"claimName": "dump-pvc"}},
							},
						},
					},
				},
			}),
			want: []string{
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"eph-pvc"}`,
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"dump-pvc"}`,
			},
		},
		{
			rule: "VmiToPVC",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "ephemeral-vol", "ephemeral": k8s.Object{"persistentVolumeClaim": k8s.Object{"claimName": "eph-pvc"}}},
						{"name": "memdump", "memoryDump": k8s.Object{"claimName": "dump-pvc"}},
					},
				},
			}),
			want: []string{
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"eph-pvc"}`,
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"dump-pvc"}`,
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
			rule: "VmiToPVC",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "rootdisk", "dataVolume": k8s.Object{"name": "my-dv"}},
						{"name": "datadisk", "persistentVolumeClaim": k8s.Object{"claimName": "my-pvc"}},
					},
				},
			}),
			want: []string{
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"my-dv"}`,
				`k8s:PersistentVolumeClaim.v1:{"namespace":"vm-ns","name":"my-pvc"}`,
			},
		},
		{
			rule: "VmiToDataVolume",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "rootdisk", "dataVolume": k8s.Object{"name": "my-dv"}},
						{"name": "datadisk", "persistentVolumeClaim": k8s.Object{"claimName": "my-pvc"}},
					},
				},
			}),
			want: []string{`k8s:DataVolume.v1beta1.cdi.kubevirt.io:{"namespace":"vm-ns","name":"my-dv"}`},
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
						{"metadata": k8s.Object{"name": "template-dv"}},
					},
					"template": k8s.Object{
						"spec": k8s.Object{
							"volumes": []k8s.Object{
								{"name": "extra", "dataVolume": k8s.Object{"name": "preexisting-dv"}},
							},
						},
					},
				},
			}),
			want: []string{
				`k8s:DataVolume.v1beta1.cdi.kubevirt.io:{"namespace":"vm-ns","name":"template-dv"}`,
				`k8s:DataVolume.v1beta1.cdi.kubevirt.io:{"namespace":"vm-ns","name":"preexisting-dv"}`,
			},
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
			rule: "VmToSecret",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"template": k8s.Object{
						"spec": k8s.Object{
							"volumes": []k8s.Object{
								{"name": "cloudinit", "cloudInitNoCloud": k8s.Object{"secretRef": k8s.Object{"name": "ci-secret"}}},
								{"name": "sysprep-vol", "sysprep": k8s.Object{"secret": k8s.Object{"name": "sysprep-secret"}}},
							},
							"accessCredentials": []k8s.Object{
								{"sshPublicKey": k8s.Object{"source": k8s.Object{"secret": k8s.Object{"secretName": "ssh-key"}}}},
							},
						},
					},
				},
			}),
			want: []string{
				`k8s:Secret.v1:{"namespace":"vm-ns","name":"ci-secret"}`,
				`k8s:Secret.v1:{"namespace":"vm-ns","name":"sysprep-secret"}`,
				`k8s:Secret.v1:{"namespace":"vm-ns","name":"ssh-key"}`,
			},
		},
		{
			rule: "VmiToSecret",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "secret-vol", "secret": k8s.Object{"secretName": "my-secret"}},
						{"name": "rootdisk", "dataVolume": k8s.Object{"name": "dv1"}},
					},
				},
			}),
			want: []string{`k8s:Secret.v1:{"namespace":"vm-ns","name":"my-secret"}`},
		},
		{
			rule: "VmToConfigMap",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"template": k8s.Object{
						"spec": k8s.Object{
							"volumes": []k8s.Object{
								{"name": "cm-vol", "configMap": k8s.Object{"name": "my-cm"}},
							},
						},
					},
				},
			}),
			want: []string{`k8s:ConfigMap.v1:{"namespace":"vm-ns","name":"my-cm"}`},
		},
		{
			rule: "VmiToConfigMap",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "sysprep-vol", "sysprep": k8s.Object{"configMap": k8s.Object{"name": "sysprep-cm"}}},
					},
				},
			}),
			want: []string{`k8s:ConfigMap.v1:{"namespace":"vm-ns","name":"sysprep-cm"}`},
		},
		{
			rule: "VmToServiceAccount",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"template": k8s.Object{
						"spec": k8s.Object{
							"volumes": []k8s.Object{
								{"name": "sa-vol", "serviceAccount": k8s.Object{"serviceAccountName": "my-sa"}},
							},
						},
					},
				},
			}),
			want: []string{`k8s:ServiceAccount.v1:{"namespace":"vm-ns","name":"my-sa"}`},
		},
		{
			rule: "VmiToServiceAccount",
			start: newK8s("VirtualMachineInstance.kubevirt.io", "vm-ns", "my-vmi", k8s.Object{
				"spec": k8s.Object{
					"volumes": []k8s.Object{
						{"name": "sa-vol", "serviceAccount": k8s.Object{"serviceAccountName": "my-sa"}},
					},
				},
			}),
			want: []string{`k8s:ServiceAccount.v1:{"namespace":"vm-ns","name":"my-sa"}`},
		},
		{
			rule: "VmToInstancetype",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"instancetype": k8s.Object{
						"name": "u1.medium",
						"kind": "VirtualMachineClusterInstancetype",
					},
				},
			}),
			want: []string{`k8s:VirtualMachineClusterInstancetype.v1beta1.instancetype.kubevirt.io:{"name":"u1.medium"}`},
		},
		{
			rule: "VmToInstancetype",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"instancetype": k8s.Object{
						"name": "custom-type",
						"kind": "VirtualMachineInstancetype",
					},
				},
			}),
			want: []string{`k8s:VirtualMachineInstancetype.v1beta1.instancetype.kubevirt.io:{"namespace":"vm-ns","name":"custom-type"}`},
		},
		{
			rule: "VmToPreference",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"preference": k8s.Object{
						"name": "rhel9",
					},
				},
			}),
			want: []string{`k8s:VirtualMachineClusterPreference.v1beta1.instancetype.kubevirt.io:{"name":"rhel9"}`},
		},
		{
			rule: "VmToPreference",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"preference": k8s.Object{
						"name": "custom-pref",
						"kind": "VirtualMachinePreference",
					},
				},
			}),
			want: []string{`k8s:VirtualMachinePreference.v1beta1.instancetype.kubevirt.io:{"namespace":"vm-ns","name":"custom-pref"}`},
		},
		{
			rule: "VmToNetAttachDef",
			start: newK8s("VirtualMachine.kubevirt.io", "vm-ns", "my-vm", k8s.Object{
				"spec": k8s.Object{
					"template": k8s.Object{
						"spec": k8s.Object{
							"networks": []k8s.Object{
								{"name": "default", "pod": k8s.Object{}},
								{"name": "net1", "multus": k8s.Object{"networkName": "my-nad"}},
							},
						},
					},
				},
			}),
			want: []string{`k8s:NetworkAttachmentDefinition.v1.k8s.cni.cncf.io:{"namespace":"vm-ns","name":"my-nad"}`},
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
