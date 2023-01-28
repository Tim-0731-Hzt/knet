package plugin

import (
	apps_v1 "k8s.io/api/apps/v1"
	api_v1 "k8s.io/api/core/v1"
	node_v1 "k8s.io/api/node/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	serviceAccount = &api_v1.ServiceAccount{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "kata-label-node",
			Namespace: "kube-system",
		},
	}
	clusterRole = &rbac.ClusterRole{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "ClusterRole",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "node-labeler",
		},
		Rules: []rbac.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "patch"},
			},
		},
	}
	clusterRoleBinding = &rbac.ClusterRoleBinding{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "ClusterRoleBinding",
			APIVersion: "rbac.authorization.k8s.io/v1",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "kata-label-node-rb",
		},
		RoleRef: rbac.RoleRef{
			APIGroup: rbac.GroupName,
			Kind:     "ClusterRole",
			Name:     "node-labeler",
		},
		Subjects: []rbac.Subject{
			{
				Kind:      rbac.ServiceAccountKind,
				Namespace: "kube-system",
				Name:      "kata-label-node",
			},
		},
	}
	privileged                = func() *bool { b := true; return &b }
	hostPathDirectoryOrCreate = api_v1.HostPathDirectoryOrCreate
	daemonSetDeployment       = &apps_v1.DaemonSet{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "kata-deploy",
			Namespace: "kube-system",
		},
		Spec: apps_v1.DaemonSetSpec{
			Selector: &meta_v1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "kata-deploy",
				},
			},
			UpdateStrategy: apps_v1.DaemonSetUpdateStrategy{
				Type:          apps_v1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: nil,
			},
			Template: api_v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{
					Labels: map[string]string{
						"name": "kata-deploy",
					},
				},
				Spec: api_v1.PodSpec{
					ServiceAccountName: "kata-label-node",
					Containers: []api_v1.Container{
						{
							Name:    "kube-kata",
							Image:   "quay.io/kata-containers/kata-deploy:stable",
							Command: []string{"bash", "-c", "/opt/kata-artifacts/scripts/kata-deploy.sh install"},
							Env: []api_v1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &api_v1.EnvVarSource{
										FieldRef: &api_v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							VolumeMounts: []api_v1.VolumeMount{
								{
									Name:      "crio-conf",
									MountPath: "/etc/crio/",
								},
								{
									Name:      "containerd-conf",
									MountPath: "/etc/containerd",
								},
								{
									Name:      "kata-artifacts",
									MountPath: "/opt/kata/",
								},
								{
									Name:      "dbus",
									MountPath: "/var/run/dbus",
								},
								{
									Name:      "run",
									MountPath: "/run",
								},
								{
									Name:      "local-bin",
									MountPath: "/usr/local/bin/",
								},
								{
									Name:      "dev",
									MountPath: "/dev/",
								},
							},
							Lifecycle: &api_v1.Lifecycle{
								PreStop: &api_v1.LifecycleHandler{
									Exec: &api_v1.ExecAction{
										Command: []string{"sleep", "6m"},
									},
								},
							},
							ImagePullPolicy: api_v1.PullAlways,
							SecurityContext: &api_v1.SecurityContext{
								Privileged: privileged(),
							},
						},
					},
					Volumes: []api_v1.Volume{
						{
							Name: "crio-conf",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/etc/crio/",
								},
							},
						},
						{
							Name: "containerd-conf",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/etc/containerd/",
								},
							},
						},
						{
							Name: "kata-artifacts",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/opt/kata/",
									Type: &hostPathDirectoryOrCreate,
								},
							},
						},
						{
							Name: "dbus",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/var/run/dbus",
								},
							},
						},
						{
							Name: "run",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/run",
								},
							},
						},
						{
							Name: "local-bin",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/usr/local/bin/",
								},
							},
						},
						{
							Name: "dev",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/dev/",
								},
							},
						},
					},
				},
			},
		},
	}
	runtimeClass = func(name string, cpu string, memory string) *node_v1.RuntimeClass {
		return &node_v1.RuntimeClass{
			TypeMeta: meta_v1.TypeMeta{
				Kind:       "RuntimeClass",
				APIVersion: "node.k8s.io/v1",
			},
			ObjectMeta: meta_v1.ObjectMeta{
				Name: name,
			},
			Handler: name,
			Overhead: &node_v1.Overhead{
				PodFixed: map[api_v1.ResourceName]resource.Quantity{
					api_v1.ResourceCPU:    resource.MustParse(cpu),
					api_v1.ResourceMemory: resource.MustParse(memory),
				},
			},
			Scheduling: &node_v1.Scheduling{
				NodeSelector: map[string]string{
					"katacontainers.io/kata-runtime": "true",
				},
			},
		}
	}
	daemonSetCleanDeployment = &apps_v1.DaemonSet{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "kubelet-kata-cleanup",
			Namespace: "kube-system",
		},
		Spec: apps_v1.DaemonSetSpec{
			Selector: &meta_v1.LabelSelector{
				MatchLabels: map[string]string{
					"name": "kubelet-kata-cleanup",
				},
			},
			UpdateStrategy: apps_v1.DaemonSetUpdateStrategy{
				Type: apps_v1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &apps_v1.RollingUpdateDaemonSet{
					MaxUnavailable: &intstr.IntOrString{
						Type:   0,
						IntVal: 1,
					},
				},
			},
			Template: api_v1.PodTemplateSpec{
				ObjectMeta: meta_v1.ObjectMeta{
					Labels: map[string]string{
						"name": "kubelet-kata-cleanup",
					},
				},
				Spec: api_v1.PodSpec{
					ServiceAccountName: "kata-label-node",
					Containers: []api_v1.Container{
						{
							Name:    "kube-kata",
							Image:   "quay.io/kata-containers/kata-deploy:stable",
							Command: []string{"bash", "-c", "/opt/kata-artifacts/scripts/kata-deploy.sh reset"},
							Env: []api_v1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &api_v1.EnvVarSource{
										FieldRef: &api_v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
							},
							VolumeMounts: []api_v1.VolumeMount{
								{
									Name:      "dbus",
									MountPath: "/var/run/dbus",
								},
								{
									Name:      "systemd",
									MountPath: "/run/systemd",
								},
							},
							Lifecycle: &api_v1.Lifecycle{
								PreStop: &api_v1.LifecycleHandler{
									Exec: &api_v1.ExecAction{
										Command: []string{"bash", "-c", "/opt/kata-artifacts/scripts/kata-deploy.sh cleanup"},
									},
								},
							},
							ImagePullPolicy: api_v1.PullAlways,
							SecurityContext: &api_v1.SecurityContext{
								Privileged: privileged(),
							},
						},
					},
					Volumes: []api_v1.Volume{
						{
							Name: "dbus",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/var/run/dbus",
								},
							},
						},
						{
							Name: "systemd",
							VolumeSource: api_v1.VolumeSource{
								HostPath: &api_v1.HostPathVolumeSource{
									Path: "/run/systemd",
								},
							},
						},
					},
				},
			},
		},
	}
)
