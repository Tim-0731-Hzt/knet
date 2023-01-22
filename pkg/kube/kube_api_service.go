package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	apps_v1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	node_v1 "k8s.io/api/node/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/cmd/debug"
	"k8s.io/kubectl/pkg/scheme"
	"time"
)

var KubernetesConfigFlags = genericclioptions.NewConfigFlags(true)

type KubernetesApiService interface {
	ExecuteCommand(req ExecCommandRequest) (int, error)
	CreatePod(podName string) error
	DeletePod(podName string, ks KubernetesApiService) error
	GetPod(podName string, namespace string) (*v1.Pod, error)
	GenerateDebugContainer(podName string, namespace string, containerName string) (*v1.Pod, *v1.EphemeralContainer, error)
	DeployDaemonSet(d *apps_v1.DaemonSet) error
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
	Namespace string
	Container string
	Command   []string
	StdIn     io.Reader
	StdOut    io.Writer
	StdErr    io.Writer
}

type Writer struct {
	Output string
}

func NewKubernetesApiServiceImpl() (k *KubernetesApiServiceImpl, err error) {
	k = &KubernetesApiServiceImpl{}
	//rawConfig, err := KubernetesConfigFlags.ToRawKubeConfigLoader().RawConfig()
	//currentContext, exists := rawConfig.Contexts[rawConfig.CurrentContext]
	//if !exists {
	//	return nil, errors.New("context doesn't exist")
	//}
	//loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	//configOverrides := &clientcmd.ConfigOverrides{}
	//kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
	//k.restConfig, err = kubeConfig.ClientConfig()
	k.restConfig, err = KubernetesConfigFlags.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	k.restConfig.Timeout = 30 * time.Second
	// new client from rest config
	k.clientset, err = kubernetes.NewForConfig(k.restConfig)
	if err != nil {
		return nil, err
	}
	//k.resultingContext = currentContext.DeepCopy()
	//k.resultingContext.Namespace = UserSpecifiedNamespace
	k.applier, err = debug.NewProfileApplier(debug.ProfileLegacy)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (k *KubernetesApiServiceImpl) ExecuteCommand(req ExecCommandRequest) (int, error) {
	execRequest := k.clientset.CoreV1().RESTClient().Post().Resource("pods").Name(req.PodName).Namespace(req.Namespace).SubResource("exec")
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
		return 0, nil
	}
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdin:  req.StdIn,
		Stdout: req.StdOut,
		Tty:    false,
	})
	var exitCode = 0
	if err != nil {
		if exitErr, ok := err.(utilexec.ExitError); ok && exitErr.Exited() {
			exitCode = exitErr.ExitStatus()
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

func (k *KubernetesApiServiceImpl) GetPod(podName string, namespace string) (*v1.Pod, error) {
	return k.clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
}

func (k *KubernetesApiServiceImpl) GenerateDebugContainer(podName string, namespace string, containerName string) (*v1.Pod, *v1.EphemeralContainer, error) {
	pod, err := k.GetPod(podName, namespace)
	if err != nil {
		return nil, nil, err
	}
	ecc := v1.EphemeralContainerCommon{
		Name:            "debug4",
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

func (k *KubernetesApiServiceImpl) DeployDaemonSet(d *apps_v1.DaemonSet) error {
	_, err := k.clientset.AppsV1().DaemonSets("kube-system").Create(context.TODO(), d, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k *KubernetesApiServiceImpl) CreateRuntimeClass(d *node_v1.RuntimeClass) error {
	_, err := k.clientset.NodeV1().RuntimeClasses().Create(context.TODO(), d, metav1.CreateOptions{})
	if err != nil {
		return nil
	}
	return err
}

func (k *KubernetesApiServiceImpl) isPodRunning(podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		listOptions := metav1.ListOptions{
			LabelSelector: "name=kata-deploy",
		}
		pods, err := k.clientset.CoreV1().Pods(namespace).List(context.TODO(), listOptions)
		if err != nil {
			return false, err
		}
		for _, pod := range pods.Items {
			switch pod.Status.Phase {
			case v1.PodRunning:
				return true, nil
			case v1.PodFailed, v1.PodSucceeded:
				return false, errors.New("pod name is empty")
			}
		}
		return false, nil
	}
}

// WaitForPodRunning Poll up to timeout seconds for pod to enter running state.
// Returns an error if the pod never enters the running state.
func (k *KubernetesApiServiceImpl) WaitForPodRunning(namespace, podName string, timeout time.Duration) error {
	return wait.PollImmediate(time.Minute, timeout, k.isPodRunning(podName, namespace))
}
