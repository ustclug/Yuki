package cmd

import (
	"github.com/spf13/cobra"
)

func NewCmdCompletion() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "completion",
		Short: "Output shell completion code for the specified shell (bash or zsh).",
	}

	bashCmd := &cobra.Command{
		Use: "bash",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
		},
	}
	zshCmd := &cobra.Command{
		Use: "zsh",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
		},
	}
	rootCmd.AddCommand(bashCmd, zshCmd)
	return rootCmd
}
