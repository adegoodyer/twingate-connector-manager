package kube

import (
	"bytes"
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
