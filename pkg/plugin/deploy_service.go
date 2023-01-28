package plugin

import (
	"fmt"
	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os"
	"os/exec"
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

	pod, err := d.kubeService.GetKataDeployPod()
	if err != nil {
		return err
	}

	executeKataRuntimeRequest := kube.ExecCommandRequest{
		PodName:   pod.Name,
		Namespace: pod.Namespace,
		Container: pod.Spec.Containers[0].Name,
		Command:   []string{"/opt/kata/bin/kata-runtime", "help"},
		StdOut:    os.Stdout,
		StdErr:    os.Stderr,
	}
	fmt.Println(executeKataRuntimeRequest)
	log.Infof("exec called")
	_, err = d.kubeService.ExecuteCommand(executeKataRuntimeRequest)
	if err != nil {
		return err
	}
	return nil
}
func (d *DeployService) Run() error {
	if err := d.kubeService.CreateRbac(serviceAccount, clusterRole, clusterRoleBinding); err != nil {
		return err
	}
	if err := d.kubeService.DeployDaemonSet(daemonSetDeployment); err != nil {
		return err
	}
	cmd := exec.Command("kubectl", "-n", "kube-system", "wait", "--timeout=10m", "--for=condition=Ready", "-l", "name=kata-deploy", "pod")
	if err := cmd.Run(); err != nil {
		log.WithError(err).Errorf("failed to execute kubectl wait")
		return err
	}
	QemuRuntimeClass := runtimeClass("kata-qemu", "250m", "160Mi")
	if err := d.kubeService.CreateRuntimeClass(QemuRuntimeClass); err != nil {
		return err
	}
	ClhRuntimeClass := runtimeClass("kata-clh", "250m", "130Mi")
	if err := d.kubeService.CreateRuntimeClass(ClhRuntimeClass); err != nil {
		return err
	}
	FcRuntimeClass := runtimeClass("kata-fc", "250m", "130Mi")
	if err := d.kubeService.CreateRuntimeClass(FcRuntimeClass); err != nil {
		return err
	}
	DragonballRuntimeClass := runtimeClass("kata-dragonball", "250m", "130Mi")
	if err := d.kubeService.CreateRuntimeClass(DragonballRuntimeClass); err != nil {
		return err
	}
	return nil
}

func (d *DeployService) cleanup() error {
	return nil
}
