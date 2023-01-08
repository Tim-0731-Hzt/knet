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

func (t *TcpdumpService) Complete(cmd *cobra.Command, args []string) error {

	return nil
}
func (t *TcpdumpService) Validate() error {
	return nil
}
func (t *TcpdumpService) Run() error {
	return nil
}
