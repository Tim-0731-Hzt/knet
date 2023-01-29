package plugin

import (
	"fmt"
	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"strings"
)

type ExecService struct {
	kubeService            *kube.KubernetesApiServiceImpl
	UserSpecifiedNamespace string
	UserSpecifiedPodName   string
	deployPod              *v1.Pod
	pod                    *v1.Pod
}

func NewExecService() *ExecService {
	return &ExecService{}
}

func (e *ExecService) Complete(cmd *cobra.Command, args []string) error {
	var err error
	if e.UserSpecifiedNamespace == "" {
		e.UserSpecifiedNamespace = "default"
	}
	if e.UserSpecifiedPodName == "" {
		return errors.New("pod name is empty")
	}
	e.kubeService, err = kube.NewKubernetesApiServiceImpl()
	if err != nil {
		return err
	}

	return nil
}
func (e *ExecService) Validate() error {
	log.Infof("validate pod")
	pod, err := e.kubeService.GetPod(e.UserSpecifiedPodName, e.UserSpecifiedNamespace)
	if err != nil {
		return err
	}
	e.pod = pod
	deployPod, err := e.kubeService.GetKataDeployPod(pod)
	if err != nil {
		return err
	}
	e.deployPod = deployPod
	return nil
}
func (e *ExecService) Run() error {
	log.Infof("Run")
	fmt.Println(strings.Replace(e.pod.Status.ContainerStatuses[0].ContainerID, "containerd://", "", -1))
	s := strings.Replace(e.pod.Status.ContainerStatuses[0].ContainerID, "containerd://", "", -1)
	shellScript := fmt.Sprintf("kata-runtime exec $(echo $(crictl --runtime-endpoint unix:///var/run/containerd/containerd.sock inspect %s | grep sandboxID) | awk '{print $2}' | sed 's/^.//;s/.$//' | sed 's/.$//')", s)
	executeGetPidRequests := kube.ExecCommandRequest{
		PodName:   e.deployPod.Name,
		Namespace: e.deployPod.Namespace,
		Container: "kube-kata",
		Command:   []string{"bash", "-c", shellScript},
	}
	if _, err := e.kubeService.ExecuteVMCommand(executeGetPidRequests); err != nil {
		return err
	}
	return nil
}

func (e *ExecService) cleanup() error {
	return nil
}
