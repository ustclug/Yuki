package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func NewCmdCompletion() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "completion",
		Short: "Output shell completion code for the specified shell (bash or zsh).",
	}

	bashCmd := &cobra.Command{
		Use: "bash",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Root().GenBashCompletion(os.Stdout)
		},
	}
	zshCmd := &cobra.Command{
		Use: "zsh",
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Root().GenZshCompletion(os.Stdout)
		},
	}
	rootCmd.AddCommand(bashCmd, zshCmd)
	return rootCmd
}
