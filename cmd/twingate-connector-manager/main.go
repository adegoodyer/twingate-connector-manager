package main

import (
	"fmt"
	"os"

	"github.com/adegoodyer/twingate-connector-manager/internal/commands"
	"github.com/adegoodyer/twingate-connector-manager/internal/config"
	"github.com/adegoodyer/twingate-connector-manager/internal/kube"
)

func main() {
	cfg, help := config.ParseFlags()
	if help {
		config.Usage()
		os.Exit(0)
	}
	if cfg.Command == "" {
		config.Usage()
		os.Exit(1)
	}

	client := kube.NewClient(cfg.Kubectl, cfg.Namespace)

	switch cfg.Command {
	case "list":
		if err := commands.CmdList(client); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	case "versions":
		if err := commands.CmdVersions(client); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	case "upgrade":
		if len(cfg.Args) < 1 {
			fmt.Fprintln(os.Stderr, "update requires at least one identifier")
			config.Usage()
			os.Exit(2)
		}
		if err := commands.CmdUpgrade(client, cfg.Args, cfg.HelmRepo, cfg.AutoYes); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cfg.Command)
		config.Usage()
		os.Exit(2)
	}
}
