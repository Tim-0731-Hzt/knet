package plugin

import (
	"github.com/spf13/cobra"
)

type KnetService interface {
	Complete(cmd *cobra.Command, args []string) error
	Validate() error
	Run() error
}
