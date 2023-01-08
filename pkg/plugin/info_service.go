package plugin

import (
	"github.com/spf13/cobra"
)

type InfoService struct {
}

func (t *InfoService) Complete(cmd *cobra.Command, args []string) error {

	return nil
}
func (t *InfoService) Validate() error {
	return nil
}
func (t *InfoService) Run() error {
	return nil
}
