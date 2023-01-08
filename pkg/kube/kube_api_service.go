package kube

import (
	"io"
	"time"

	v1 "k8s.io/api/core/v1"
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
	return nil
}

func (k *KubernetesApiServiceImpl) DeletePod(podName string) error {
	return nil
}

func (k *KubernetesApiServiceImpl) GetPod(podName string) (*v1.Pod, error) {
	return nil, nil
}
