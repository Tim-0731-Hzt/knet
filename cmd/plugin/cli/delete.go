package cli

import (
	"github.com/Tim-0731-Hzt/knet/pkg/plugin"
	"github.com/spf13/cobra"
)

func init() {
	d := plugin.NewDeleteService()
	var deleteCmd = &cobra.Command{
		Use:   "delete",
		Short: "delete kata containers on each node",
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

	cmd.AddCommand(deleteCmd)
}
