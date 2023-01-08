package kube

import (
	"io"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesApiService interface {
	ExecuteCommand(podName string, containerName string, command []string, stdOut io.Writer) (int, error)
	CreatePod(podName string) error
	DeletePod(podName string) error
	GetPod(podName string) (*v1.Pod, error)
}
type KubernetesApiServiceImpl struct {
	clientset       *kubernetes.Clientset
	restConfig      *rest.Config
	targetNamespace string
}

func NewKubernetesApiServiceImpl() (k *KubernetesApiServiceImpl, err error) {
	k = &KubernetesApiServiceImpl{}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil)
	k.restConfig, err = kubeConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	k.restConfig.Timeout = 30 * time.Second
	k.clientset, err = kubernetes.NewForConfig(k.restConfig)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (k *KubernetesApiServiceImpl) ExecuteCommand(podName string, containerName string, command []string, stdOut io.Writer) (int, error) {
	return 0, nil
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
		Name:            "containerName",
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

	_, err := k.clientset.CoreV1().Pods("default").Create(&pod)
	if err != nil {
		return err
	}

	return nil
}

func (k *KubernetesApiServiceImpl) DeletePod(podName string) error {
	return nil
}

func (k *KubernetesApiServiceImpl) GetPod(podName string) (*v1.Pod, error) {
	return nil, nil
}
