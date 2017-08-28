package main

import (
	"os"

	"github.com/ryotarai/paramedic-agent/paramedic"
)

func main() {
	cli := paramedic.NewCLI()
	status := cli.Start()
	os.Exit(status)
}
