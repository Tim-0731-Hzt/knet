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
	"fmt"
	"github.com/Tim-0731-Hzt/knet/pkg/plugin"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

func init() {
	e := plugin.NewExecService()
	var execCmd = &cobra.Command{
		Use: "exec",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("exec called")
			err := e.Complete(cmd, args)
			if err != nil {
				return err
			}
			err = e.Validate()
			if err != nil {
				return err
			}
			err = e.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}
	execCmd.Flags().StringVarP(&e.UserSpecifiedNamespace, "namespace", "n", "", "namespace (optional)")
	_ = viper.BindEnv("namespace", "KUBECTL_PLUGINS_CURRENT_NAMESPACE")
	_ = viper.BindPFlag("namespace", cmd.Flags().Lookup("namespace"))

	execCmd.Flags().StringVarP(&e.UserSpecifiedPodName, "pod", "p", "", "pod (optional)")
	_ = viper.BindEnv("pod", "KUBECTL_PLUGINS_LOCAL_FLAG_POD")
	_ = viper.BindPFlag("pod", cmd.Flags().Lookup("pod"))

	cmd.AddCommand(execCmd)
}
