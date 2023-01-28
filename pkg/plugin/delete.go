package plugin

import (
	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
		return err
	}
	log.Infof("wait kata-deploy")
	if err := d.kubeService.Wait("name=kata-deploy"); err != nil {
		return err
	}
	log.Infof("deploy kubelet-kata-cleanup")
	if err := d.kubeService.DeployDaemonSet(daemonSetCleanDeployment); err != nil {
		return err
	}
	log.Infof("exec cleanup")
	if err := d.kubeService.ExecuteCleanupCommand(); err != nil {
		return err
	}
	log.Infof("delete kubelet-kata-cleanup")
	if err := d.kubeService.DeleteDaemonSet("kubelet-kata-cleanup"); err != nil {
		return err
	}
	log.Infof("wait kubelet-kata-cleanup stop")
	if err := d.kubeService.Wait("kubelet-kata-cleanup"); err != nil {
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
