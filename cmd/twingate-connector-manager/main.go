package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	kubeCmd string
	ns      string
	autoYes bool
)

func usage() {
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

func runCmd(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	var errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("%v: %s", err, strings.TrimSpace(errb.String()))
	}
	return out.String(), nil
}

func kubectl(args ...string) (string, error) {
	all := append([]string{}, args...)
	return runCmd(kubeCmd, all...)
}

func findDeployment(ident string) (string, error) {
	out, err := kubectl("-n", ns, "get", "deployment", "-o", "name")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "/", 2)
		if len(parts) == 2 && strings.Contains(parts[1], ident) {
			return parts[1], nil
		}
	}
	return "", nil
}

func findPodForDeploy(deploy string) (string, error) {
	out, err := kubectl("-n", ns, "get", "pods", "-o", "name")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "/", 2)
		if len(parts) == 2 && strings.Contains(parts[1], deploy) {
			return parts[1], nil
		}
	}
	return "", nil
}

func getVersionFromPod(pod string) (string, error) {
	if pod == "" {
		return "<no-pod>", nil
	}
	out, err := kubectl("-n", ns, "exec", "pod/"+pod, "--", "./connectord", "--version")
	if err != nil {
		return "<unknown>", nil
	}
	return strings.TrimSpace(out), nil
}

func cmdList() error {
	fmt.Printf("Pods in namespace %s:\n", ns)
	if out, err := kubectl("-n", ns, "get", "pods", "-o", "wide"); err != nil {
		return err
	} else {
		fmt.Print(out)
	}
	fmt.Println()
	fmt.Printf("Deployments in namespace %s:\n", ns)
	if out, err := kubectl("-n", ns, "get", "deployment", "-o", "wide"); err != nil {
		return err
	} else {
		fmt.Print(out)
	}
	return nil
}

func cmdVersions() error {
	fmt.Printf("Connector versions in namespace %s:\n", ns)
	out, err := kubectl("-n", ns, "get", "deployment", "-o", "name")
	if err != nil {
		return err
	}
	lines := strings.FieldsFunc(strings.TrimSpace(out), func(r rune) bool { return r == '\n' || r == '\r' })
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Printf("No deployments found in namespace %s\n", ns)
		return nil
	}
	for _, line := range lines {
		parts := strings.SplitN(line, "/", 2)
		name := parts[len(parts)-1]
		pod, _ := findPodForDeploy(name)
		ver, _ := getVersionFromPod(pod)
		fmt.Printf("%-40s %s\n", name, ver)
	}
	return nil
}

func confirm(prompt string) bool {
	if autoYes {
		return true
	}
	fmt.Printf("%s [y/N]: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		txt := strings.TrimSpace(scanner.Text())
		if strings.EqualFold(txt, "y") || strings.EqualFold(txt, "yes") {
			return true
		}
	}
	return false
}

func cmdUpdate(id1, id2 string) error {
	d1, _ := findDeployment(id1)
	d2, _ := findDeployment(id2)
	if d1 == "" || d2 == "" {
		return fmt.Errorf("could not find both deployments: d1=%s d2=%s", d1, d2)
	}
	p1, _ := findPodForDeploy(d1)
	p2, _ := findPodForDeploy(d2)
	v1old, _ := getVersionFromPod(p1)
	v2old, _ := getVersionFromPod(p2)

	fmt.Println("About to restart the following deployments in namespace", ns)
	fmt.Printf("  %s (pod: %s, version: %s)\n", d1, coalesce(p1, "<none>"), v1old)
	fmt.Printf("  %s (pod: %s, version: %s)\n", d2, coalesce(p2, "<none>"), v2old)

	if !confirm("Proceed with restarting these deployments?") {
		fmt.Println("Aborted by user.")
		return nil
	}

	steps := []string{}
	fmt.Println("Restarting", d1)
	if _, err := kubectl("-n", ns, "rollout", "restart", "deployment/"+d1); err != nil {
		return err
	}
	steps = append(steps, "restarted "+d1)

	fmt.Println("Restarting", d2)
	if _, err := kubectl("-n", ns, "rollout", "restart", "deployment/"+d2); err != nil {
		return err
	}
	steps = append(steps, "restarted "+d2)

	fmt.Println("Waiting for rollouts to complete (timeout 120s each)...")
	_, _ = runCmd(kubeCmd, "-n", ns, "rollout", "status", "deployment/"+d1, "--timeout=120s")
	_, _ = runCmd(kubeCmd, "-n", ns, "rollout", "status", "deployment/"+d2, "--timeout=120s")

	p1n, _ := findPodForDeploy(d1)
	p2n, _ := findPodForDeploy(d2)
	v1new, _ := getVersionFromPod(p1n)
	v2new, _ := getVersionFromPod(p2n)

	fmt.Println()
	fmt.Println("Summary for update operation:")
	fmt.Println("Connector:", d1)
	fmt.Println("  old version:", v1old)
	fmt.Println("  new version:", v1new)
	fmt.Println()
	fmt.Println("Connector:", d2)
	fmt.Println("  old version:", v2old)
	fmt.Println("  new version:", v2new)
	fmt.Println()
	fmt.Println("Steps taken:")
	for _, s := range steps {
		fmt.Println(" -", s)
	}
	return nil
}

func coalesce(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func main() {
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

	if *help {
		usage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	cmd := args[0]
	switch cmd {
	case "list":
		if err := cmdList(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	case "versions":
		if err := cmdVersions(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	case "update":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "update requires two identifiers")
			usage()
			os.Exit(2)
		}
		if err := cmdUpdate(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(2)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", cmd)
		usage()
		os.Exit(2)
	}
}
