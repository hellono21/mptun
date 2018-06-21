package cmd

import (
	"../config"
)

var start func(*config.Config)

func Execute(f func(*config.Config)) {
	start = f
	RootCmd.Execute()
}
