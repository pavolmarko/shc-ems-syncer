package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/pavolmarko/shc-ems-syncer/config"
	"github.com/pavolmarko/shc-ems-syncer/ems"
)

func main() {
	err := start()
	if err != nil {
		fmt.Println(err)
	}
}

func start() error {
	configPath := flag.String("config", "", "path to config file")

	flag.Parse()

	if configPath == nil || *configPath == "" {
		return errors.New("need --config <path>")
	}

	cfg, err := config.Read(*configPath)
	if err != nil {
		return fmt.Errorf("can't read config: %v", err)
	}

	emsCli := ems.CreateClient(cfg.EmsEspHostport, cfg.EmsEspAccessToken)
	res, err := emsCli.Ping()
	if err != nil {
		return fmt.Errorf("can't ping ems: %v", err)
	}

	fmt.Println(res)

	return nil
}
