package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/adegoodyer/twingate-connector-manager/internal/kube"
)

// CmdList prints pods and deployments (wide) in the namespace
func CmdList(c *kube.Client) error {
	fmt.Printf("Pods in namespace %s:\n", c.Namespace)
	if out, err := c.Kubectl("-n", c.Namespace, "get", "pods", "-o", "wide"); err != nil {
		return err
	} else {
		fmt.Print(out)
	}
	fmt.Println()
	fmt.Printf("Deployments in namespace %s:\n", c.Namespace)
	if out, err := c.Kubectl("-n", c.Namespace, "get", "deployment", "-o", "wide"); err != nil {
		return err
	} else {
		fmt.Print(out)
	}
	return nil
}

// CmdVersions lists deployments and shows the connector version (by exec'ing into a pod)
func CmdVersions(c *kube.Client) error {
	fmt.Printf("Connector versions in namespace %s:\n", c.Namespace)
	out, err := c.Kubectl("-n", c.Namespace, "get", "deployment", "-o", "name")
	if err != nil {
		return err
	}
	lines := strings.FieldsFunc(strings.TrimSpace(out), func(r rune) bool { return r == '\n' || r == '\r' })
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		fmt.Printf("No deployments found in namespace %s\n", c.Namespace)
		return nil
	}
	for _, line := range lines {
		parts := strings.SplitN(line, "/", 2)
		name := parts[len(parts)-1]
		pod, _ := c.FindPodForDeploy(name)
		ver, _ := c.GetVersionFromPod(pod)
		fmt.Printf("%-40s %s\n", name, ver)
	}
	return nil
}

func confirm(prompt string, autoYes bool) bool {
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

func coalesce(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// CmdUpdate restarts two deployments (found by identifier substrings) and reports before/after versions
func CmdUpdate(c *kube.Client, id1, id2 string, autoYes bool) error {
	d1, _ := c.FindDeployment(id1)
	d2, _ := c.FindDeployment(id2)
	if d1 == "" || d2 == "" {
		return fmt.Errorf("could not find both deployments: d1=%s d2=%s", d1, d2)
	}
	p1, _ := c.FindPodForDeploy(d1)
	p2, _ := c.FindPodForDeploy(d2)
	v1old, _ := c.GetVersionFromPod(p1)
	v2old, _ := c.GetVersionFromPod(p2)

	fmt.Println("About to restart the following deployments in namespace", c.Namespace)
	fmt.Printf("  %s (pod: %s, version: %s)\n", d1, coalesce(p1, "<none>"), v1old)
	fmt.Printf("  %s (pod: %s, version: %s)\n", d2, coalesce(p2, "<none>"), v2old)

	if !confirm("Proceed with restarting these deployments?", autoYes) {
		fmt.Println("Aborted by user.")
		return nil
	}

	steps := []string{}
	fmt.Println("Restarting", d1)
	if err := c.RolloutRestart(d1); err != nil {
		return err
	}
	steps = append(steps, "restarted "+d1)

	fmt.Println("Restarting", d2)
	if err := c.RolloutRestart(d2); err != nil {
		return err
	}
	steps = append(steps, "restarted "+d2)

	fmt.Println("Waiting for rollouts to complete (timeout 120s each)...")
	// best-effort waits; ignore errors but run them
	_ = c.RolloutStatus(d1, "120s")
	_ = c.RolloutStatus(d2, "120s")

	p1n, _ := c.FindPodForDeploy(d1)
	p2n, _ := c.FindPodForDeploy(d2)
	v1new, _ := c.GetVersionFromPod(p1n)
	v2new, _ := c.GetVersionFromPod(p2n)

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
