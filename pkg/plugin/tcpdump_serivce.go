package plugin

import (
	"github.com/Tim-0731-Hzt/knet/pkg/kube"
	"github.com/goombaio/namegenerator"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"time"
)

type TcpdumpService struct {
	kubeService *kube.KubernetesApiServiceImpl
	Config      *Tcpdump
}

type Tcpdump struct {
	UserSpecifiedNamespace string
	UserSpecifiedPodsName  []string
	UserSpecifiedPods      map[string]*v1.Pod
}

func NewTcpdumpConfig() *Tcpdump {
	return &Tcpdump{}
}

func NewTcpdumpService(tcpdump *Tcpdump) *TcpdumpService {
	return &TcpdumpService{Config: tcpdump}
}

func (t *TcpdumpService) Complete(cmd *cobra.Command, args []string) error {
	if t.Config.UserSpecifiedNamespace == "" {
		t.Config.UserSpecifiedNamespace = "default"
	}
	if len(t.Config.UserSpecifiedPodsName) == 0 {
		return errors.New("pod name is empty")
	}
	var err error
	t.kubeService, err = kube.NewKubernetesApiServiceImpl()
	if err != nil {
		return err
	}
	t.Config.UserSpecifiedPods = make(map[string]*v1.Pod)
	return nil
}
func (t *TcpdumpService) Validate() error {
	log.Infof("validate pod")
	for _, p := range t.Config.UserSpecifiedPodsName {
		pod, err := t.kubeService.GetPod(p, t.Config.UserSpecifiedNamespace)
		if err != nil {
			return err
		}
		if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
			return errors.Errorf("cannot tcpdump on a container in a completed pod; current phase is %s", pod.Status.Phase)
		}
		t.Config.UserSpecifiedPods[p] = pod
	}
	return nil
}
func (t *TcpdumpService) Run() error {
	if len(t.Config.UserSpecifiedPodsName) > 1 {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		errs := make(chan error, 1)
		go func() {
			sigchan := make(chan os.Signal, 1)
			signal.Notify(sigchan, os.Interrupt)
			<-sigchan
			log.Println("stop capture")
			var args = []string{"-w", dir + "/merge.pcap"}
			for _, p := range t.Config.UserSpecifiedPodsName {
				args = append(args, dir+"/"+p+".pcap")
			}
			cmd := exec.Command("mergecap", args...)
			err = cmd.Run()
			if err != nil {
				errs <- err
			}
			log.Println("pcap file will be stored in " + dir + "/merge.pcap")
			os.Exit(0)
		}()
		var wg sync.WaitGroup
		wg.Add(len(t.Config.UserSpecifiedPodsName))
		for _, p := range t.Config.UserSpecifiedPodsName {
			log.Infof("creating ephemeral container inside pod %s", p)
			debugContainerName := namegenerator.NewNameGenerator(time.Now().UTC().UnixNano()).Generate()
			_, _, err := t.kubeService.GenerateDebugContainer(p, "default", t.Config.UserSpecifiedPods[p].Spec.Containers[0].Name, debugContainerName)
			if err != nil {
				log.WithError(err).Errorf("failed to create debug container")
				return err
			}
			f, err := os.OpenFile(dir+"/"+p+".pcap", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
			if err != nil {
				panic(err)
			}
			defer f.Close()
			executeTcpdumpRequest := kube.ExecCommandRequest{
				PodName:   p,
				Namespace: t.Config.UserSpecifiedNamespace,
				Container: debugContainerName,
				Command:   []string{"/usr/bin/tcpdump", "-w", "-"},
				StdOut:    f,
			}
			log.Infof("start capture")
			go func() {
				defer wg.Done()
				_, err = t.kubeService.ExecuteCommand(executeTcpdumpRequest)
				if err != nil {
					log.WithError(err).Errorf("failed to execute tcpdump")
				}
			}()
		}
		if err, open := <-errs; open {
			return err
		}
		wg.Wait()
	} else {
		podName := t.Config.UserSpecifiedPodsName[0]
		debugContainerName := namegenerator.NewNameGenerator(time.Now().UTC().UnixNano()).Generate()
		_, _, err := t.kubeService.GenerateDebugContainer(podName, t.Config.UserSpecifiedNamespace, t.Config.UserSpecifiedPods[podName].Spec.Containers[0].Name, debugContainerName)
		if err != nil {
			log.WithError(err).Errorf("failed to create debug container")
			return err
		}
		executeTcpdumpRequest := kube.ExecCommandRequest{
			PodName:   podName,
			Namespace: t.Config.UserSpecifiedNamespace,
			Container: debugContainerName,
			Command:   []string{"/usr/bin/tcpdump", "-w", "-"},
			StdOut:    os.Stdout,
		}
		log.Infof("spawning termshark!")
		_, err = t.kubeService.ExecuteCommand(executeTcpdumpRequest)
		if err != nil {
			log.WithError(err).Errorf("failed to execute tcpdump")
			return err
		}
	}
	return nil
}

func (t *TcpdumpService) cleanup() error {
	return nil
}
