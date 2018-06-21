package cmd

import (
	"../info"
	"fmt"
	"github.com/spf13/cobra"
)

var format string
var configPath string
var showVersion bool

func init() {
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Print version information and quid")
	RootCmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file")
	RootCmd.PersistentFlags().StringVarP(&format, "format", "f", "toml", "config file format: \"toml\" or \"json\"")
}

var RootCmd = &cobra.Command{
	Use:	"mptun",
	Short:	"multi path udp tunnel",
	Run: func(cmd *cobra.Command, args []string) {
		if showVersion {
			fmt.Println(info.Version)
			return
		}

		if configPath == "" {
			cmd.Help()
			return
		}

		FromFileCmd.Run(cmd, []string{configPath})
	},
}
