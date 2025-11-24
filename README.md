# twingate-connector-manager

[![Go Reference](https://pkg.go.dev/badge/github.com/adegoodyer/twingate-connector-manager.svg)](https://pkg.go.dev/github.com/adegoodyer/twingate-connector-manager)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

CLI tool for managing/automating Twingate Connector upgrades within Kubernetes clusters.

## Install

```bash
go install github.com/adegoodyer/twingate-connector-manager@latest
```

## Usage
```bash
twingate-connector-manager
Usage: twingate-connector-manager [options] <command> [args]

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
```

## Samples
```bash
# list connector resources (pods and deployments)
twingate-connector-manager list

# show connector versions
twingate-connector-manager versions

# update two connectors by their identifiers (with confirmation prompt)
twingate-connector-manager update observant-beagle unyielding-copperhead
```

## Sample Output
```bash
# list connector resources (pods and deployments)
twingate-connector-manager list
Pods in namespace twingate-connectors:
NAME                                                       READY   STATUS    RESTARTS     AGE   IP             NODE             NOMINATED NODE   READINESS GATES
twingate-outrageous-alligator-connector-797895ddc4-4gv24   1/1     Running   5 (2d ago)   49d   10.244.2.253   talos-worker-0   <none>           <none>
twingate-unique-hyrax-connector-85c49fd4bc-29tqd           1/1     Running   0            61m   10.244.5.84    talos-worker-1   <none>           <none>

Deployments in namespace twingate-connectors:
NAME                                      READY   UP-TO-DATE   AVAILABLE   AGE    CONTAINERS   IMAGES                 SELECTOR
twingate-outrageous-alligator-connector   1/1     1            1           195d   connector    twingate/connector:1   app.kubernetes.io/instance=twingate-outrageous-alligator,app.kubernetes.io/name=connector
twingate-unique-hyrax-connector           1/1     1            1           195d   connector    twingate/connector:1   app.kubernetes.io/instance=twingate-unique-hyrax,app.kubernetes.io/name=connector

# show connector versions and helm releases
twingate-connector-manager versions
Connector versions in namespace twingate-connectors:
twingate-outrageous-alligator-connector  1.79.0
twingate-unique-hyrax-connector          1.79.0

Helm releases in namespace twingate-connectors:
NAME                            NAMESPACE               REVISION        UPDATED                                 STATUS          CHART                   APP VERSION
twingate-outrageous-alligator   twingate-connectors     1               2025-05-12 17:44:43.566766 +0100 BST    deployed        connector-0.1.31        latest
twingate-unique-hyrax           twingate-connectors     2               2025-11-24 11:36:10.68027 +0000 UTC     deployed        connector-0.1.31        latest

# upgrade multiple connectors by their identifiers (with confirmation prompt)
twingate-connector-manager upgrade outrageous-alligator unique-hyrax
Note: 'upgrade' requires Helm installed and targets Helm-managed deployments (annotation meta.helm.sh/release-name).
About to upgrade the following Helm-managed connectors in namespace twingate-connectors:

  Release: twingate-outrageous-alligator
  Chart: connector
  Deployment: twingate-outrageous-alligator-connector
  Pod: twingate-outrageous-alligator-connector-797895ddc4-4gv24
  Version: 1.79.0

  Release: twingate-unique-hyrax
  Chart: connector
  Deployment: twingate-unique-hyrax-connector
  Pod: twingate-unique-hyrax-connector-85c49fd4bc-29tqd
  Version: 1.79.0

Proceed with upgrading these Helm releases? [y/N]:

# full upgrade output for single connector
Note: 'upgrade' requires Helm installed and targets Helm-managed deployments (annotation meta.helm.sh/release-name).
About to upgrade the following Helm-managed connectors in namespace twingate-connectors:

  Release: twingate-outrageous-alligator
  Chart: connector
  Deployment: twingate-outrageous-alligator-connector
  Pod: twingate-outrageous-alligator-connector-797895ddc4-4gv24
  Version: 1.79.0

Proceed with upgrading these Helm releases? [y/N]: y
Upgrading helm release twingate-outrageous-alligator (chart: twingate/connector)

Summary for upgrade operation:
Connector: twingate-outrageous-alligator-connector
  old version: 1.79.0
  new version: 1.80.0

Steps taken:
 - helm repo update
 - upgraded helm release twingate-outrageous-alligator
 - waited for rollout twingate-outrageous-alligator-connector
```

## Tags

- `latest`: Most recent stable build
- `x.y.z`: Specific version builds (e.g., `2.7.5`)
- `x.y`: Minor version builds (e.g., `2.7`)

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
