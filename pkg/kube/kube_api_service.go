package kube

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	apps_v1 "k8s.io/api/apps/v1"
	api_v1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	node_v1 "k8s.io/api/node/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/remotecommand"
	utilexec "k8s.io/client-go/util/exec"
	"k8s.io/kubectl/pkg/cmd/debug"
	"k8s.io/kubectl/pkg/scheme"
	"os"
	"time"
)

var KubernetesConfigFlags = genericclioptions.NewConfigFlags(true)

type KubernetesApiService interface {
	ExecuteCommand(req ExecCommandRequest) (int, error)
	CreatePod(podName string) error
	DeletePod(podName string, ks KubernetesApiService) error
	GetPod(podName string, namespace string) (*v1.Pod, error)
	GenerateDebugContainer(podName string, namespace string, containerName string, debugContainerName string) (*v1.Pod, *v1.EphemeralContainer, error)
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
		Stdout: req.StdOut,
		Tty:    false,
	})
	var exitCode = 0
	if err != nil {
		if exitErr, ok := err.(utilexec.ExitError); ok && exitErr.Exited() {
			exitCode = exitErr.ExitStatus()
			return 1, err
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

func (k *KubernetesApiServiceImpl) GenerateDebugContainer(podName string, namespace string, containerName string, debugContainerName string) (*v1.Pod, *v1.EphemeralContainer, error) {
	pod, err := k.GetPod(podName, namespace)
	if err != nil {
		return nil, nil, err
	}
	ecc := v1.EphemeralContainerCommon{
		Name:            debugContainerName,
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
	if err := k.applier.Apply(copied, debugContainerName, copied); err != nil {
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
	_, err = k.clientset.CoreV1().Pods(namespace).Patch(context.TODO(), copied.Name, types.StrategicMergePatchType, patch, opt, "ephemeralcontainers")
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

func (k *KubernetesApiServiceImpl) CreateRbac(sva *api_v1.ServiceAccount, cr *rbac.ClusterRole, crb *rbac.ClusterRoleBinding) error {
	_, err := k.clientset.CoreV1().ServiceAccounts("kube-system").Create(context.TODO(), sva, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = k.clientset.RbacV1().ClusterRoles().Create(context.TODO(), cr, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	_, err = k.clientset.RbacV1().ClusterRoleBindings().Create(context.TODO(), crb, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k *KubernetesApiServiceImpl) DeleteDaemonSet(d string) error {
	err := k.clientset.AppsV1().DaemonSets("kube-system").Delete(context.TODO(), d, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k *KubernetesApiServiceImpl) DeleteRuntimeClass() error {
	err := k.clientset.NodeV1().RuntimeClasses().Delete(context.TODO(), "kata-qemu", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = k.clientset.NodeV1().RuntimeClasses().Delete(context.TODO(), "kata-clh", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = k.clientset.NodeV1().RuntimeClasses().Delete(context.TODO(), "kata-fc", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = k.clientset.NodeV1().RuntimeClasses().Delete(context.TODO(), "kata-dragonball", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k *KubernetesApiServiceImpl) DeleteRbac() error {
	err := k.clientset.CoreV1().ServiceAccounts("kube-system").Delete(context.TODO(), "kata-label-node", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = k.clientset.RbacV1().ClusterRoles().Delete(context.TODO(), "node-labeler", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	err = k.clientset.RbacV1().ClusterRoleBindings().Delete(context.TODO(), "kata-label-node-rb", metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (k *KubernetesApiServiceImpl) GetKataDeployPod(p *v1.Pod) (*v1.Pod, error) {
	listOptions := metav1.ListOptions{
		LabelSelector: "name=kata-deploy",
	}
	pods, err := k.clientset.CoreV1().Pods("kube-system").List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}
	for _, pod := range pods.Items {
		if pod.Spec.NodeName == p.Spec.NodeName {
			return &pod, nil
		}
	}
	return nil, errors.New("deploy pod not found")
}

func (k *KubernetesApiServiceImpl) ExecuteCleanupCommand() error {
	listOptions := metav1.ListOptions{
		LabelSelector: "name=kubelet-kata-cleanup",
	}
	pods, err := k.clientset.CoreV1().Pods("kube-system").List(context.TODO(), listOptions)
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		executeCleanupRequest := ExecCommandRequest{
			PodName:   pod.Name,
			Namespace: pod.Namespace,
			Container: "kube-kata",
			Command:   []string{"bash", "-c", "/opt/kata-artifacts/scripts/kata-deploy.sh reset"},
			StdOut:    os.Stdout,
		}
		if _, err := k.ExecuteCommand(executeCleanupRequest); err != nil {
			return err
		}
	}
	return nil
}

func (k *KubernetesApiServiceImpl) ExecuteVMCommand(req ExecCommandRequest) (int, error) {
	execRequest := k.clientset.CoreV1().RESTClient().Post().Resource("pods").Name(req.PodName).Namespace(req.Namespace).SubResource("exec")
	execRequest.VersionedParams(&v1.PodExecOptions{
		Container: req.Container,
		Command:   req.Command,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", execRequest.URL())
	if err != nil {
		return 0, nil
	}
	if !terminal.IsTerminal(0) || !terminal.IsTerminal(1) {
		return 0, err
	}
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		return 1, err
	}
	defer func(fd int, oldState *terminal.State) error {
		err := terminal.Restore(fd, oldState)
		if err != nil {
			return err
		}
		return nil
	}(0, oldState)

	// 用IO读写替换 os stdout
	screen := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdin:  screen,
		Stdout: screen,
		Stderr: screen,
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
