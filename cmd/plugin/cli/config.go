package cli

import (
	"github.com/Tim-0731-Hzt/knet/pkg/plugin"
	"github.com/spf13/cobra"
)

func init() {
	c := plugin.NewConfigService()
	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "configure config.toml for kata-containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := c.Complete(cmd, args); err != nil {
				return err
			}
			if err := c.Validate(); err != nil {
				return err
			}
			if err := c.Run(); err != nil {
				return err
			}
			return nil
		},
	}
	configCmd.Flags().BoolVar(&c.DebugConsole, "debug_console", false, "enable debug console")

	cmd.AddCommand(configCmd)
}
