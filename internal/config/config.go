package config

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds the parsed flags and command/args
type Config struct {
	Kubectl   string
	Namespace string
	AutoYes   bool
	Command   string
	Args      []string
}

var (
	kubeCmd string
	ns      string
	autoYes bool
)

// Usage prints the tool help. Kept similar to original main.go text.
func Usage() {
	fmt.Fprintf(os.Stderr, `Usage: %s [options] <command> [args]

Commands:
  list                         List pods then deployments in namespace (connectors)
  versions                     Show connector pod versions for all deployments in the namespace
  update <id1> <id2>           Restart deployments for two connectors (identifiers) and report before/after versions

Options:
  -n, --namespace NAMESPACE    Kubernetes namespace (default: %s)
  -y, --yes                    Auto-confirm actions
  -k, --kubectl PATH           Kubectl binary to use (default: %s)
  -h, --help                   Show this help and exit

Examples:
  %s list -n twingate-connectors
  %s versions
  %s update observant-beagle unyielding-copperhead -n twingate-connectors

`, os.Args[0], ns, kubeCmd, os.Args[0], os.Args[0], os.Args[0])
}

// ParseFlags parses the CLI flags and returns a Config and a boolean indicating whether help was requested
func ParseFlags() (Config, bool) {
	kubeCmd = os.Getenv("KUBECTL")
	if kubeCmd == "" {
		kubeCmd = "kubectl"
	}

	flag.StringVar(&ns, "n", "twingate-connectors", "namespace")
	flag.StringVar(&ns, "namespace", "twingate-connectors", "namespace")
	flag.BoolVar(&autoYes, "y", false, "auto-confirm actions")
	flag.BoolVar(&autoYes, "yes", false, "auto-confirm actions")
	flag.StringVar(&kubeCmd, "k", kubeCmd, "kubectl binary")
	flag.StringVar(&kubeCmd, "kubectl", kubeCmd, "kubectl binary")
	help := flag.Bool("h", false, "help")
	flag.Parse()

	args := flag.Args()
	var cmd string
	var cmdArgs []string
	if len(args) > 0 {
		cmd = args[0]
		if len(args) > 1 {
			cmdArgs = args[1:]
		}
	}

	cfg := Config{
		Kubectl:   kubeCmd,
		Namespace: ns,
		AutoYes:   autoYes,
		Command:   cmd,
		Args:      cmdArgs,
	}
	// Normalize namespace for display in usage
	if cfg.Namespace == "" {
		cfg.Namespace = "twingate-connectors"
	}

	// guard: ensure printed usage shows reasonable defaults
	ns = cfg.Namespace
	kubeCmd = cfg.Kubectl

	return cfg, *help || strings.EqualFold(cfg.Command, "help")
}
