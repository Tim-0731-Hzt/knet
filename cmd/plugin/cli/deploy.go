package cli

import (
	"github.com/Tim-0731-Hzt/knet/pkg/plugin"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	d := plugin.NewDeployService()
	// tcpdumpCmd represents the tcpdump command
	var deployCmd = &cobra.Command{
		Use:   "deploy",
		Short: "deploy kata containers on each node",
		RunE: func(cmd *cobra.Command, args []string) error {
			log.Infof("deploy called")
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
