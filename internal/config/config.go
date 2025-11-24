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
	SetImage  string
	Timeout   string
}

var (
	kubeCmd  string
	ns       string
	autoYes  bool
	helmRepo string
	setImage string
	timeout  string
)

// Usage prints the tool help. Kept similar to original main.go text.
func Usage() {
	fmt.Fprint(os.Stderr, `Usage: twingate-connector-manager [options] <command> [args]

Commands:
	list                         List pods then deployments in namespace (connectors)
	versions                     Show connector pod versions for all deployments in the namespace (also prints helm list for the namespace)
	upgrade <id1> [id2 ... idN]  Upgrade one or more connectors (Helm-managed releases only) and report before/after versions

Options:
	-n, --namespace NAMESPACE    Kubernetes namespace (default: twingate-connectors)
	-y, --yes                    Auto-confirm actions
	-k, --kubectl PATH           Kubectl binary to use (default: kubectl)
	--helm-repo NAME             Helm chart repository name to use for upgrades (default: twingate)
	--set-image TAG              Optional: set the image tag when upgrading (overrides chart default)
	--timeout DURATION           Optional: helm/rollout timeout (default: 120s)
	-h, --help                   Show this help and exit

Examples:
	twingate-connector-manager list -n twingate-connectors
	twingate-connector-manager versions
	twingate-connector-manager upgrade unique-hyrax
	twingate-connector-manager upgrade connector-a connector-b connector-c -n twingate-connectors

Notes:
	- upgrade requires Helm to be installed and that each target Deployment be Helm-managed (annotation meta.helm.sh/release-name).
	- The command runs helm repo update and then helm upgrade <release> <repo>/<chart> with --reuse-values --wait --timeout for each release.
	- If any target deployment is not Helm-managed the command will fail and list the non-Helm deployments.

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
	flag.StringVar(&setImage, "set-image", "", "Set image tag when upgrading (overrides chart default)")
	flag.StringVar(&timeout, "timeout", "120s", "Upgrade and rollout timeout (e.g., 120s, 2m)")
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
		SetImage:  setImage,
		Timeout:   timeout,
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
