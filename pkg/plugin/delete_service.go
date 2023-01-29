package plugin

import (
	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"os/exec"
)

type DeleteService struct {
	kubeService *kube.KubernetesApiServiceImpl
}

func NewDeleteService() *DeleteService {
	return &DeleteService{}
}
func (d *DeleteService) Complete(cmd *cobra.Command, args []string) error {
	var err error
	d.kubeService, err = kube.NewKubernetesApiServiceImpl()
	if err != nil {
		return err
	}
	return nil
}

func (d *DeleteService) Run() error {

	log.Infof("delete kata-deploy")
	if err := d.kubeService.DeleteDaemonSet("kata-deploy"); err != nil {
		//return err
	}
	cmd := exec.Command("kubectl", "-n", "kube-system", "wait", "--timeout=10m", "--for=delete", "-l", "name=kata-deploy", "pod")
	if err := cmd.Run(); err != nil {
		log.WithError(err).Errorf("failed to execute kubectl wait")
		//return err
	}

	log.Infof("deploy kubelet-kata-cleanup")
	if err := d.kubeService.DeployDaemonSet(daemonSetCleanDeployment); err != nil {
		//return err
	}

	cmd = exec.Command("kubectl", "-n", "kube-system", "wait", "--timeout=10m", "--for=condition=Ready", "-l", "name=kubelet-kata-cleanup", "pod")
	if err := cmd.Run(); err != nil {
		log.WithError(err).Errorf("failed to execute kubectl wait")
		return err
	}

	log.Infof("exec cleanup")
	if err := d.kubeService.ExecuteCleanupCommand(); err != nil {
		log.WithError(err).Errorf("failed to execute reset command")
		return err
	}
	log.Infof("delete kubelet-kata-cleanup")
	if err := d.kubeService.DeleteDaemonSet("kubelet-kata-cleanup"); err != nil {
		return err
	}
	cmd = exec.Command("kubectl", "-n", "kube-system", "wait", "--timeout=10m", "--for=delete", "-l", "name=kubelet-kata-cleanup", "pod")
	if err := cmd.Run(); err != nil {
		log.WithError(err).Errorf("failed to execute kubectl wait")
		return err
	}
	log.Infof("delete rbac")
	if err := d.kubeService.DeleteRbac(); err != nil {
		return err
	}

	log.Infof("delete runtimeclass")
	if err := d.kubeService.DeleteRuntimeClass(); err != nil {
		return err
	}

	return nil
}
