package kube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps kubectl interactions for the configured namespace
type Client struct {
	KubeCmd   string
	Namespace string
}

// NewClient constructs a Client
func NewClient(kubeCmd, namespace string) *Client {
	return &Client{KubeCmd: kubeCmd, Namespace: namespace}
}

func (c *Client) runCmd(name string, args ...string) (string, error) {
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

// Kubectl runs the configured kubectl with the provided args
func (c *Client) Kubectl(args ...string) (string, error) {
	all := append([]string{}, args...)
	return c.runCmd(c.KubeCmd, all...)
}

// FindDeployment searches for a deployment whose name contains ident
func (c *Client) FindDeployment(ident string) (string, error) {
	out, err := c.Kubectl("-n", c.Namespace, "get", "deployment", "-o", "name")
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

// FindPodForDeploy finds a pod whose name contains the deployment name
func (c *Client) FindPodForDeploy(deploy string) (string, error) {
	out, err := c.Kubectl("-n", c.Namespace, "get", "pods", "-o", "name")
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

// GetVersionFromPod runs ./connectord --version in the pod and returns the output
func (c *Client) GetVersionFromPod(pod string) (string, error) {
	if pod == "" {
		return "<no-pod>", nil
	}
	out, err := c.Kubectl("-n", c.Namespace, "exec", "pod/"+pod, "--", "./connectord", "--version")
	if err != nil {
		return "<unknown>", nil
	}
	return strings.TrimSpace(out), nil
}

// RolloutRestart restarts the deployment
func (c *Client) RolloutRestart(deploy string) error {
	_, err := c.Kubectl("-n", c.Namespace, "rollout", "restart", "deployment/"+deploy)
	return err
}

// RolloutStatus waits for rollout to complete with a timeout (e.g. "120s")
func (c *Client) RolloutStatus(deploy, timeout string) error {
	// use runCmd to avoid interpretation of stderr-only exits; we intentionally ignore the result in callers
	_, err := c.runCmd(c.KubeCmd, "-n", c.Namespace, "rollout", "status", "deployment/"+deploy, "--timeout="+timeout)
	return err
}

// GetHelmReleaseForDeployment returns the helm release name if the deployment has helm annotations
func (c *Client) GetHelmReleaseForDeployment(deploy string) (string, error) {
	// Use jsonpath to read the meta.helm.sh/release-name annotation
	out, err := c.Kubectl("-n", c.Namespace, "get", "deployment", deploy, "-o", "jsonpath={.metadata.annotations.meta\\.helm\\.sh/release-name}")
	if err != nil {
		return "", err
	}
	out = strings.TrimSpace(out)
	return out, nil
}

// HelmRepoUpdate runs `helm repo update` (updates all repos)
func (c *Client) HelmRepoUpdate() error {
	_, err := c.runCmd("helm", "repo", "update")
	return err
}

// HelmReleaseChart returns the chart name (without version) for a given release in the namespace
func (c *Client) HelmReleaseChart(release string) (string, error) {
	out, err := c.runCmd("helm", "list", "-n", c.Namespace, "-o", "json")
	if err != nil {
		return "", err
	}
	var list []struct {
		Name  string `json:"name"`
		Chart string `json:"chart"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return "", err
	}
	for _, r := range list {
		if r.Name == release {
			// Chart is like "connector-1.2.3" — strip the last hyphen+version
			idx := strings.LastIndex(r.Chart, "-")
			if idx <= 0 {
				return r.Chart, nil
			}
			return r.Chart[:idx], nil
		}
	}
	return "", fmt.Errorf("release %s not found in namespace %s", release, c.Namespace)
}

// HelmUpgrade performs `helm upgrade <release> <repo>/<chart>` with --reuse-values and --wait
func (c *Client) HelmUpgrade(release, repo, chart, timeout string, setImage string) (string, error) {
	args := []string{"upgrade", release, repo + "/" + chart, "--namespace", c.Namespace, "--reuse-values", "--wait", "--timeout", timeout}
	if setImage != "" {
		args = append(args, "--set", "image.tag="+setImage)
	}
	return c.runCmd("helm", args...)
}

// HelmList returns the output of `helm list -n <namespace>`
func (c *Client) HelmList() (string, error) {
	return c.runCmd("helm", "list", "-n", c.Namespace)
}
