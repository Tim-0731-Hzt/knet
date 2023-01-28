package cli

import (
	"github.com/Tim-0731-Hzt/knet/pkg/plugin"
	"github.com/spf13/cobra"
)

func init() {
	d := plugin.NewDeployService()
	// tcpdumpCmd represents the tcpdump command
	var deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "deploy kata containers on each node",
		RunE: func(cmd *cobra.Command, args []string) error {
			err := d.Complete(cmd, args)
			if err != nil {
				return err
			}
			err = d.Run()
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.AddCommand(deployCmd)
}
