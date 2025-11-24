package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

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

	// Also print helm releases in the namespace
	fmt.Println()
	fmt.Printf("Helm releases in namespace %s:\n", c.Namespace)
	helmOut, herr := c.HelmList()
	if herr != nil {
		fmt.Printf("(helm list failed: %v)\n", herr)
	} else {
		fmt.Print(helmOut)
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
func CmdUpgrade(c *kube.Client, ids []string, helmRepo string, setImage string, timeout string, autoYes bool) error {
	if len(ids) == 0 {
		return fmt.Errorf("no identifiers provided")
	}

	// prerequisite note
	fmt.Println("Note: 'upgrade' requires Helm installed and targets Helm-managed deployments (annotation meta.helm.sh/release-name).")

	// Resolve deployments
	type item struct {
		id      string
		deploy  string
		release string
		chart   string
		pod     string
		oldVer  string
		newVer  string
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

	// Detect helm releases and chart names for each deployment
	nonHelm := []string{}
	for _, it := range items {
		rel, _ := c.GetHelmReleaseForDeployment(it.deploy)
		it.release = rel
		if rel == "" {
			nonHelm = append(nonHelm, it.deploy)
			continue
		}
		chart, err := c.HelmReleaseChart(rel)
		if err != nil {
			return fmt.Errorf("could not determine chart for release %s: %v", rel, err)
		}
		it.chart = chart
	}
	if len(nonHelm) > 0 {
		return fmt.Errorf("the following deployments are not helm-managed; this command requires helm-managed deployments: %v", nonHelm)
	}

	// Get pods and old versions
	for _, it := range items {
		p, _ := c.FindPodForDeploy(it.deploy)
		it.pod = p
		v, _ := c.GetVersionFromPod(p)
		it.oldVer = v
	}

	fmt.Printf("About to upgrade the following Helm-managed connectors in namespace %s:\n\n", c.Namespace)
	for _, it := range items {
		fmt.Printf("  Release: %s\n", it.release)
		fmt.Printf("  Chart: %s\n", it.chart)
		fmt.Printf("  Deployment: %s\n", it.deploy)
		fmt.Printf("  Pod: %s\n", coalesce(it.pod, "<none>"))
		fmt.Printf("  Version: %s\n", it.oldVer)
		fmt.Println()
	}

	if !confirm("Proceed with upgrading these Helm releases?", autoYes) {
		fmt.Println("Aborted by user.")
		return nil
	}
	// perform helm repo update once and record the step
	steps := []string{}
	if err := c.HelmRepoUpdate(); err != nil {
		return fmt.Errorf("helm repo update failed: %v", err)
	}
	steps = append(steps, "helm repo update")

	for _, it := range items {
		fmt.Printf("Upgrading helm release %s (chart: %s/%s)\n", it.release, helmRepo, it.chart)
		if _, err := c.HelmUpgrade(it.release, helmRepo, it.chart, timeout, setImage); err != nil {
			return fmt.Errorf("helm upgrade failed for release %s: %v", it.release, err)
		}
		steps = append(steps, "upgraded helm release "+it.release)
		// after upgrade, wait for rollout
		_ = c.RolloutStatus(it.deploy, timeout)
		steps = append(steps, "waited for rollout "+it.deploy)
	}

	// Determine polling deadline from timeout string
	dur, err := time.ParseDuration(timeout)
	if err != nil || dur <= 0 {
		dur = 120 * time.Second
	}
	// Get new versions — poll until version changes or timeout per item
	for _, it := range items {
		deadline := time.Now().Add(dur)
		var last string
		for time.Now().Before(deadline) {
			p, _ := c.FindPodForDeploy(it.deploy)
			ver, _ := c.GetVersionFromPod(p)
			last = ver
			if ver != "<unknown>" && ver != it.oldVer {
				it.newVer = ver
				break
			}
			// if old was unknown, accept any known value
			if it.oldVer == "<unknown>" && ver != "<unknown>" {
				it.newVer = ver
				break
			}
			time.Sleep(3 * time.Second)
		}
		if it.newVer == "" {
			// set last-seen value (may be "<unknown>")
			it.newVer = last
		}
	}

	fmt.Println()
	fmt.Println("Summary for upgrade operation:")
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
