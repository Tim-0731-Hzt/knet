package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/cmd/debug"
	"k8s.io/kubectl/pkg/scheme"
	"time"
)

type KubernetesApiService interface {
	ExecuteCommand(req ExecCommandRequest) (int, error)
	CreatePod(podName string) error
	DeletePod(podName string, ks KubernetesApiService) error
	GetPod(podName string) (*v1.Pod, error)
	GenerateDebugContainer(pod *v1.Pod, containerName string) (*v1.Pod, *v1.EphemeralContainer, error)
}
type KubernetesApiServiceImpl struct {
	clientset        *kubernetes.Clientset
	restConfig       *rest.Config
	resultingContext *api.Context
	targetNamespace  string
	applier          debug.ProfileApplier
}

type ExecCommandRequest struct {
	PodName   string
	Container string
	Command   []string
	StdIn     io.Reader
	StdOut    io.Writer
	StdErr    io.Writer
}

type Writer struct {
	Output string
}

func NewKubernetesApiServiceImpl(UserSpecifiedNamespace string) (k *KubernetesApiServiceImpl, err error) {
	k = &KubernetesApiServiceImpl{}
	configFlags := genericclioptions.NewConfigFlags(true)
	rawConfig, err := configFlags.ToRawKubeConfigLoader().RawConfig()
	currentContext, exists := rawConfig.Contexts[rawConfig.CurrentContext]
	if !exists {
		return nil, errors.New("context doesn't exist")
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	k.restConfig, err = kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	k.restConfig.Timeout = 30 * time.Second
	k.clientset, err = kubernetes.NewForConfig(k.restConfig)
	if err != nil {
		return nil, err
	}
	k.resultingContext = currentContext.DeepCopy()
	if UserSpecifiedNamespace != "" {
		k.resultingContext.Namespace = UserSpecifiedNamespace
	}
	k.applier, err = debug.NewProfileApplier(debug.ProfileLegacy)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (k *KubernetesApiServiceImpl) ExecuteCommand(req ExecCommandRequest) (int, error) {
	execRequest := k.clientset.CoreV1().RESTClient().Post().Resource("pods").Name(req.PodName).Namespace("default").SubResource("exec")
	execRequest.VersionedParams(&v1.PodExecOptions{
		Container: req.Container,
		Command:   req.Command,
		Stdin:     req.StdIn != nil,
		Stdout:    req.StdOut != nil,
		Stderr:    false,
		TTY:       false,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", execRequest.URL())
	if err != nil {
		fmt.Println("hello")
		return 0, nil
	}
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  req.StdIn,
		Stdout: req.StdOut,
		Tty:    false,
	})
	var exitCode = 0
	if err != nil {
		if exitErr, ok := err.(utilexec.ExitError); ok && exitErr.Exited() {
			exitCode = exitErr.ExitStatus()
			fmt.Println("hello")
			return 1, nil
		}
	}
	return exitCode, nil
}

func (k *KubernetesApiServiceImpl) CreatePod(podName string) error {
	typeMetadata := metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}

	objectMetadata := metav1.ObjectMeta{
		GenerateName: "ksniff-",
		Namespace:    k.targetNamespace,
		Labels: map[string]string{
			"app":                    "ksniff",
			"app.kubernetes.io/name": "ksniff",
		},
	}

	privilegedContainer := v1.Container{
		Name:            "container-name",
		Image:           "busybox",
		ImagePullPolicy: "IfNotPresent",
		Command:         []string{"sh", "-c", "sleep 10000000"},
	}
	podSpecs := v1.PodSpec{
		NodeName:      "ap-southeast-1.10.0.0.86",
		RestartPolicy: v1.RestartPolicyNever,
		HostPID:       true,
		Containers:    []v1.Container{privilegedContainer},
	}

	pod := v1.Pod{
		TypeMeta:   typeMetadata,
		ObjectMeta: objectMetadata,
		Spec:       podSpecs,
	}

	opt := metav1.CreateOptions{}

	_, err := k.clientset.CoreV1().Pods("default").Create(context.TODO(), &pod, opt)
	if err != nil {
		return err
	}

	return nil
}

func (k *KubernetesApiServiceImpl) DeletePod(podName string, ks KubernetesApiService) error {
	switch ks.(type) {
	case *KubernetesApiServiceImpl:
		fmt.Println("hello")
	default:
		fmt.Println("i stored")
	}
	return nil
}

func (k *KubernetesApiServiceImpl) GetPod(podName string) (*v1.Pod, error) {
	return k.clientset.CoreV1().Pods("default").Get(context.TODO(), podName, metav1.GetOptions{})
}

func (k *KubernetesApiServiceImpl) GenerateDebugContainer(pod *v1.Pod, containerName string) (*v1.Pod, *v1.EphemeralContainer, error) {
	ecc := v1.EphemeralContainerCommon{
		Name:            "debug",
		Image:           "nicolaka/netshoot",
		ImagePullPolicy: v1.PullIfNotPresent,
		Args:            []string{"sleep", "3600"},
	}
	ec := &v1.EphemeralContainer{
		EphemeralContainerCommon: ecc,
		TargetContainerName:      containerName,
	}

	copied := pod.DeepCopy()
	copied.Spec.EphemeralContainers = append(copied.Spec.EphemeralContainers, *ec)
	if err := k.applier.Apply(copied, "debug", copied); err != nil {
		return nil, nil, err
	}

	podJS, err := json.Marshal(pod)

	debugJS, err := json.Marshal(copied)
	if err != nil {
		return nil, nil, err
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(podJS, debugJS, pod)
	if err != nil {
		return nil, nil, err
	}
	opt := metav1.PatchOptions{}
	_, err = k.clientset.CoreV1().Pods("default").Patch(context.TODO(), copied.Name, types.StrategicMergePatchType, patch, opt, "ephemeralcontainers")
	if err != nil {
		return nil, nil, err
	}
	return copied, ec, nil
}
