/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cli

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Tim-0731-Hzt/knet/pkg/plugin"
	"github.com/spf13/cobra"
)

func init() {
	c := plugin.NewTcpdumpConfig()
	t := plugin.NewTcpdumpService(c)
	// tcpdumpCmd represents the tcpdump command
	var tcpdumpCmd = &cobra.Command{
		Use:   "tcpdump",
		Short: "perform tcpdump on target pod",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Infof("tcpdump called")
			err := t.Complete(cmd, args)
			if err != nil {
				return err
			}
			err = t.Validate()
			if err != nil {
				return err
			}
			err = t.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}

	tcpdumpCmd.Flags().StringVarP(&t.Config.UserSpecifiedNamespace, "namespace", "n", "", "namespace (optional)")
	_ = viper.BindEnv("namespace", "KUBECTL_PLUGINS_CURRENT_NAMESPACE")
	_ = viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))

	tcpdumpCmd.Flags().StringVarP(&t.Config.UserSpecifiedPodName, "pod", "p", "", "pod (optional)")
	_ = viper.BindEnv("pod", "KUBECTL_PLUGINS_LOCAL_FLAG_POD")
	_ = viper.BindPFlag("pod", cmd.Flags().Lookup("pod"))

	cmd.AddCommand(tcpdumpCmd)
}
