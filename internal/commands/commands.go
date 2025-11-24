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

// CmdUpdate restarts N deployments (found by identifier substrings) and reports before/after versions
func CmdUpdate(c *kube.Client, ids []string, autoYes bool) error {
	if len(ids) == 0 {
		return fmt.Errorf("no identifiers provided")
	}

	// Resolve deployments
	type item struct {
		id     string
		deploy string
		pod    string
		oldVer string
		newVer string
	}
	items := make([]*item, 0, len(ids))
	for _, id := range ids {
		d, _ := c.FindDeployment(id)
		items = append(items, &item{id: id, deploy: d})
	}

	// Ensure all deployments found
	missing := []string{}
	for _, it := range items {
		if it.deploy == "" {
			missing = append(missing, it.id)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("could not find deployments for identifiers: %v", missing)
	}

	// Get pods and old versions
	for _, it := range items {
		p, _ := c.FindPodForDeploy(it.deploy)
		it.pod = p
		v, _ := c.GetVersionFromPod(p)
		it.oldVer = v
	}

	fmt.Println("About to restart the following deployments in namespace", c.Namespace)
	for _, it := range items {
		fmt.Printf("  %s (pod: %s, version: %s)\n", it.deploy, coalesce(it.pod, "<none>"), it.oldVer)
	}

	if !confirm("Proceed with restarting these deployments?", autoYes) {
		fmt.Println("Aborted by user.")
		return nil
	}

	steps := []string{}
	for _, it := range items {
		fmt.Println("Restarting", it.deploy)
		if err := c.RolloutRestart(it.deploy); err != nil {
			return err
		}
		steps = append(steps, "restarted "+it.deploy)
	}

	fmt.Println("Waiting for rollouts to complete (timeout 120s each)...")
	for _, it := range items {
		_ = c.RolloutStatus(it.deploy, "120s")
	}

	// Get new versions
	for _, it := range items {
		p, _ := c.FindPodForDeploy(it.deploy)
		it.newVer, _ = c.GetVersionFromPod(p)
	}

	fmt.Println()
	fmt.Println("Summary for update operation:")
	for _, it := range items {
		fmt.Println("Connector:", it.deploy)
		fmt.Println("  old version:", it.oldVer)
		fmt.Println("  new version:", it.newVer)
		fmt.Println()
	}
	fmt.Println("Steps taken:")
	for _, s := range steps {
		fmt.Println(" -", s)
	}
	return nil
}
