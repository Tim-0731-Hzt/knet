package plugin

import (
	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	"github.com/spf13/cobra"
	apps_v1 "k8s.io/api/apps/v1"
	api_v1 "k8s.io/api/core/v1"
	node_v1 "k8s.io/api/node/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var (
	privileged                = func() *bool { b := true; return &b }
	hostPathDirectoryOrCreate = api_v1.HostPathDirectoryOrCreate
)

type DeployService struct {
	kubeService *kube.KubernetesApiServiceImpl
}

func NewDeployService() *DeployService {
	return &DeployService{}
}
func (d *DeployService) Complete(cmd *cobra.Command, args []string) error {
	var err error
	d.kubeService, err = kube.NewKubernetesApiServiceImpl()
	if err != nil {
		return err
	}
	return nil
}
func (d *DeployService) Validate() error {
	return nil
}
func (d *DeployService) Run() error {
	daemonSetDeployment := &apps_v1.DaemonSet{
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
					},
				},
			},
		},
	}
	err := d.kubeService.DeployDaemonSet(daemonSetDeployment)
	if err != nil {
		return err
	}
	if err := d.kubeService.WaitForPodRunning("kube-system", "kata-deploy", time.Duration(10)*time.Minute); err != nil {
		return err
	}
	QemuRuntimeClass := &node_v1.RuntimeClass{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "RuntimeClass",
			APIVersion: "node.k8s.io/v1",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "kata-qemu",
		},
		Handler: "kata-qemu",
		Overhead: &node_v1.Overhead{
			PodFixed: map[api_v1.ResourceName]resource.Quantity{
				api_v1.ResourceCPU:    resource.MustParse("250m"),
				api_v1.ResourceMemory: resource.MustParse("160Mi"),
			},
		},
		Scheduling: &node_v1.Scheduling{
			NodeSelector: map[string]string{
				"katacontainers.io/kata-runtime": "true",
			},
		},
	}
	err = d.kubeService.CreateRuntimeClass(QemuRuntimeClass)
	if err != nil {
		return err
	}
	ClhRuntimeClass := &node_v1.RuntimeClass{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "RuntimeClass",
			APIVersion: "node.k8s.io/v1",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "kata-clh",
		},
		Handler: "kata-clh",
		Overhead: &node_v1.Overhead{
			PodFixed: map[api_v1.ResourceName]resource.Quantity{
				api_v1.ResourceCPU:    resource.MustParse("250m"),
				api_v1.ResourceMemory: resource.MustParse("160Mi"),
			},
		},
		Scheduling: &node_v1.Scheduling{
			NodeSelector: map[string]string{
				"katacontainers.io/kata-runtime": "true",
			},
		},
	}
	err = d.kubeService.CreateRuntimeClass(ClhRuntimeClass)
	if err != nil {
		return err
	}
	FcRuntimeClass := &node_v1.RuntimeClass{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "RuntimeClass",
			APIVersion: "node.k8s.io/v1",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "kata-fc",
		},
		Handler: "kata-fc",
		Overhead: &node_v1.Overhead{
			PodFixed: map[api_v1.ResourceName]resource.Quantity{
				api_v1.ResourceCPU:    resource.MustParse("250m"),
				api_v1.ResourceMemory: resource.MustParse("160Mi"),
			},
		},
		Scheduling: &node_v1.Scheduling{
			NodeSelector: map[string]string{
				"katacontainers.io/kata-runtime": "true",
			},
		},
	}
	err = d.kubeService.CreateRuntimeClass(FcRuntimeClass)
	if err != nil {
		return err
	}
	DragonballRuntimeClass := &node_v1.RuntimeClass{
		TypeMeta: meta_v1.TypeMeta{
			Kind:       "RuntimeClass",
			APIVersion: "node.k8s.io/v1",
		},
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "kata-dragonball",
		},
		Handler: "kata-dragonball",
		Overhead: &node_v1.Overhead{
			PodFixed: map[api_v1.ResourceName]resource.Quantity{
				api_v1.ResourceCPU:    resource.MustParse("250m"),
				api_v1.ResourceMemory: resource.MustParse("160Mi"),
			},
		},
		Scheduling: &node_v1.Scheduling{
			NodeSelector: map[string]string{
				"katacontainers.io/kata-runtime": "true",
			},
		},
	}
	err = d.kubeService.CreateRuntimeClass(DragonballRuntimeClass)
	if err != nil {
		return err
	}
	return nil
}

func (d *DeployService) cleanup() error {
	return nil
}
