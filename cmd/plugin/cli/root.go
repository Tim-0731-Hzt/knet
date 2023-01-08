package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

var cmd = &cobra.Command{
	Use:   "knet",
	Short: "Perform network diagnose on pods running in a kubernetes cluster.",
	RunE: func(cmd *cobra.Command, args []string) error {
		err := errors.New("must also specify a subcommand like info or tcpdump")
		return err
	},
}

func RootCmd() *cobra.Command {
	cobra.OnInitialize(initConfig)
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.AutomaticEnv()
}
