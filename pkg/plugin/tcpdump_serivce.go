package plugin

import (
	"fmt"
	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"os/exec"
	"strings"
)

type TcpdumpService struct {
	kubeService *kube.KubernetesApiServiceImpl
	wireshark   *exec.Cmd
	config      Tcpdump
}

type Tcpdump struct {
	UserSpecifiedPodName     string
	DetectedPodNodeName      string
	UserSpecifiedInterface   string
	UserSpecifiedFilter      string
	UserSpecifiedContainer   string
	UserSpecifiedNamespace   string
	DetectedContainerRuntime string
	DetectedContainerId      string
}

func NewTcpdumpService() *TcpdumpService {
	return &TcpdumpService{}
}

func (t *TcpdumpService) Complete(cmd *cobra.Command, args []string) error {
	t.config.UserSpecifiedNamespace = "default"
	if t.config.UserSpecifiedNamespace == "" {
		return errors.New("namespace value is empty should be custom or default")
	}
	var err error
	t.kubeService, err = kube.NewKubernetesApiServiceImpl(t.config.UserSpecifiedNamespace)
	if err != nil {
		return err
	}
	return nil
}
func (t *TcpdumpService) Validate() error {
	pod, err := t.kubeService.GetPod(t.config.UserSpecifiedPodName)
	if err != nil {
		return err
	}

	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return errors.Errorf("cannot sniff on a container in a completed pod; current phase is %s", pod.Status.Phase)
	}
	t.config.DetectedPodNodeName = pod.Spec.NodeName

	if len(pod.Spec.Containers) < 1 {
		return errors.New("no containers in specified pod")
	}

	if t.config.UserSpecifiedContainer == "" {
		t.config.UserSpecifiedContainer = pod.Spec.Containers[0].Name
	}

	if err := t.findContainerId(pod); err != nil {
		return err
	}

	return nil
}
func (t *TcpdumpService) Run() error {
	pod, err := t.kubeService.GetPod("nginx")
	if err != nil {
		return err
	}
	_, _, err = t.kubeService.GenerateDebugContainer(pod, "nginx")
	if err != nil {
		return err
	}
	t.wireshark = exec.Command("wireshark", "-k", "-i", "-")
	stdinWriter, err := t.wireshark.StdinPipe()
	if err != nil {
		return err
	}

	executeTcpdumpRequest := kube.ExecCommandRequest{
		PodName:   "nginx",
		Container: "debug",
		Command:   []string{"/usr/bin/tcpdump", "-w", "-"},
		StdOut:    stdinWriter,
	}

	go func() {
		_, err = t.kubeService.ExecuteCommand(executeTcpdumpRequest)
		if err != nil {
			_ = t.wireshark.Process.Kill()
		}
	}()
	err = t.wireshark.Run()
	if err != nil {
		fmt.Println("hello")
	}
	return err
}

func (t *TcpdumpService) findContainerId(pod *v1.Pod) error {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if t.config.UserSpecifiedContainer == containerStatus.Name {
			result := strings.Split(containerStatus.ContainerID, "://")
			if len(result) != 2 {
				break
			}
			t.config.DetectedContainerRuntime = result[0]
			t.config.DetectedContainerId = result[1]
			return nil
		}
	}

	return errors.Errorf("couldn't find container: '%s' in pod: '%s'", t.config.UserSpecifiedContainer, t.config.UserSpecifiedPodName)
}
