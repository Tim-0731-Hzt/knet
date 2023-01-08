package plugin

import (
	"os/exec"

	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	"github.com/spf13/cobra"
)

type TcpdumpService struct {
	kubeService *kube.KubernetesApiServiceImpl
	termshark   *exec.Cmd
	config      Tcpdump
}

type Tcpdump struct {
	UserSpecifiedPodName   string
	UserSpecifiedInterface string
	UserSpecifiedFilter    string
	UserSpecifiedContainer string
	UserSpecifiedNamespace string
}

func NewTcpdumpService() *TcpdumpService {
	return &TcpdumpService{}
}

func (t *TcpdumpService) Complete(cmd *cobra.Command, args []string) error {
	var err error
	t.kubeService, err = kube.NewKubernetesApiServiceImpl()
	if err != nil {
		return err
	}
	return nil
}
func (t *TcpdumpService) Validate() error {
	return nil
}
func (t *TcpdumpService) Run() error {
	err := t.kubeService.CreatePod("cdd")
	if err != nil {
		return err
	}
	return nil
}
