package main

import (
	"./info"
	"./cmd"
	"./config"
	"./server"
	"log"
	"math/rand"
	"os"
	"time"
	"runtime"
	"./logging"
)

var version string = "1.0.0"

func init() {
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	rand.Seed(time.Now().UnixNano())

	info.Version = version
	info.StartTime = time.Now()
}

func Start(name string, cfg config.Server) error {
	server, err := server.New(name, cfg)
	if err != nil {
		return err
	}

	if err = server.Start(); err != nil {
		return err
	}

	return nil
}


func main() {
	log.Printf("mptun v%s", version)
	cmd.Execute(func(cfg *config.Config){
		logging.Configure(cfg.Logging.Output, cfg.Logging.Level)
		Start("mptun", cfg.Server)
		// block forever
		<-(chan string)(nil)
	})
}
