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
	HelmRepo  string
}

var (
	kubeCmd  string
	ns       string
	autoYes  bool
	helmRepo string
)

// Usage prints the tool help. Kept similar to original main.go text.
func Usage() {
	fmt.Fprint(os.Stderr, `Usage: twingate-connector-manager [options] <command> [args]

Commands:
	list                         List pods then deployments in namespace (connectors)
	versions                     Show connector pod versions for all deployments in the namespace
	upgrade <id1> [id2 ... idN]  Upgrade one or more connectors and report before/after versions

Options:
	-n, --namespace NAMESPACE    Kubernetes namespace (default: twingate-connectors)
	-y, --yes                    Auto-confirm actions
	-k, --kubectl PATH           Kubectl binary to use (default: kubectl)
	--helm-repo NAME             Helm chart repository name to use for upgrades (default: twingate)
	-h, --help                   Show this help and exit

Examples:
	twingate-connector-manager list -n twingate-connectors
	twingate-connector-manager versions
	twingate-connector-manager upgrade unique-hyrax
	twingate-connector-manager upgrade connector-a connector-b connector-c -n twingate-connectors

`)
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
	flag.StringVar(&helmRepo, "helm-repo", "twingate", "helm chart repository name (default: twingate)")
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
		HelmRepo:  helmRepo,
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
