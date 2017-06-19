package main

import (
	"os"

	"github.com/ryotarai/paramedic-agent/paramedic"
)

func main() {
	cli := paramedic.NewCLI()
	os.Exit(cli.Start())
}
