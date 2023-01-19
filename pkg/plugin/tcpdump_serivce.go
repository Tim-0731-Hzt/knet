package plugin

import (
	"fmt"
	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"os/exec"
	"strings"
)

type TcpdumpService struct {
	kubeService *kube.KubernetesApiServiceImpl
	Wireshark   *exec.Cmd
	Config      *Tcpdump
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

func NewTcpdumpConfig() *Tcpdump {
	return &Tcpdump{}
}

func NewTcpdumpService(tcpdump *Tcpdump) *TcpdumpService {
	return &TcpdumpService{Config: tcpdump}
}

func (t *TcpdumpService) Complete(cmd *cobra.Command, args []string) error {
	if t.Config.UserSpecifiedNamespace == "" {
		return errors.New("namespace value is empty should be custom or default")
	}
	if t.Config.UserSpecifiedPodName == "" {
		return errors.New("pod name is empty")
	}
	var err error
	t.kubeService, err = kube.NewKubernetesApiServiceImpl(t.Config.UserSpecifiedNamespace)
	if err != nil {
		return err
	}
	return nil
}
func (t *TcpdumpService) Validate() error {
	log.Infof("validate pod")
	pod, err := t.kubeService.GetPod(t.Config.UserSpecifiedPodName, t.Config.UserSpecifiedNamespace)
	if err != nil {
		return err
	}
	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		return errors.Errorf("cannot sniff on a container in a completed pod; current phase is %s", pod.Status.Phase)
	}
	t.Config.DetectedPodNodeName = pod.Spec.NodeName

	if len(pod.Spec.Containers) < 1 {
		return errors.New("no containers in specified pod")
	}

	if t.Config.UserSpecifiedContainer == "" {
		t.Config.UserSpecifiedContainer = pod.Spec.Containers[0].Name
	}
	if err := t.findContainerId(pod); err != nil {
		return err
	}
	return nil
}
func (t *TcpdumpService) Run() error {
	log.Infof("start tcpdump on pod %s", t.Config.UserSpecifiedPodName)
	pod, err := t.kubeService.GetPod(t.Config.UserSpecifiedPodName, t.Config.UserSpecifiedNamespace)
	if err != nil {
		return err
	}
	log.Infof("creating ephemeral container")
	_, _, err = t.kubeService.GenerateDebugContainer(pod, t.Config.UserSpecifiedContainer)
	if err != nil {
		return err
	}
	t.Wireshark = exec.Command("wireshark", "-k", "-i", "-")
	stdinWriter, err := t.Wireshark.StdinPipe()
	if err != nil {
		return err
	}

	executeTcpdumpRequest := kube.ExecCommandRequest{
		PodName:   t.Config.UserSpecifiedPodName,
		Container: "debug",
		Command:   []string{"/usr/bin/tcpdump", "-w", "-"},
		StdOut:    stdinWriter,
	}

	go func() {
		_, err = t.kubeService.ExecuteCommand(executeTcpdumpRequest)
		if err != nil {
			_ = t.Wireshark.Process.Kill()
		}
	}()
	err = t.Wireshark.Run()
	if err != nil {
		fmt.Println("hello")
	}
	return err
}

func (t *TcpdumpService) findContainerId(pod *v1.Pod) error {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if t.Config.UserSpecifiedContainer == containerStatus.Name {
			result := strings.Split(containerStatus.ContainerID, "://")
			if len(result) != 2 {
				break
			}
			t.Config.DetectedContainerRuntime = result[0]
			t.Config.DetectedContainerId = result[1]
			return nil
		}
	}

	return errors.Errorf("couldn't find container: '%s' in pod: '%s'", t.Config.UserSpecifiedContainer, t.Config.UserSpecifiedPodName)
}
