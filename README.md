# Kubecost exporter for Flexera CCO

Kubecost Flexera Exporter is a command line tool used to automate the transfer of Kubernetes cluster spending data from Kubecost to Flexera One for further processing and analysis. It generates a CSV file for each day of the current month containing Kubernetes cluster usage data in a format compatible with the Flexera One platform and then uploads it. The app is designed to be run as a scheduled task, preferably on a Kubernetes cluster.

## Installation

The application can be installed from golang sources, from a docker image or via the helm package manager.

### Go sources

This app requires Go version 1.16 or higher. To install, run:

```bash
go install github.com/flexera-public/cbi-oi-kubecost-exporter
```

The app is configured using environment variables defined in a .env file. The following configuration options are available:

-   `KUBECOST_HOST` - the hostname of the Kubecost instance
-   `BILL_CONNECT_ID` - the ID of the bill connect to which to upload the data
-   `SHARD` - the region of your Flexera One account. Valid values are NAM, EU or AU.
-   `ORG_ID` - the ID of your Flexera One organization.
-   `REFRESH_TOKEN` - the refresh token used to obtain an access token for the Flexera One API
-   `AGGREGATION` - the level of granularity to use when aggregating the cost data. Valid values are namespace, controller, or pod.
-   `SHARE_IDLE` - a flag indicating whether to share idle costs among namespaces
-   `SHARE_NAMESPACES` - a flag indicating whether to share namespace costs among clusters
-   `SHARE_TENANCY_COSTS` - a flag indicating whether to share tenancy costs among clusters
-   `MULTIPLIER` - a multiplier to apply to the cost data
-   `IDLE` - whether to include idle resources in the usage data. valid values are true or false.
-   `FILE_PATH` - the path where the CSV files are stored
-   `UPLOAD_TIMEOUT` - the timeout for uploading the CSV files to Flexera One, in seconds.

To use this app, run:

```bash
flexera-kubecost-exporter
```

### Helm package manager

There are two different approaches for passing custom Helm config values into the kubecost-exporter:

#### 1. Pass exact parameters via --set command-line flags:

```
helm install kubecost-exporter helm-chart \
    --repo https://flexera-public.github.io/cbi-oi-kubecost-exporter/ \
    --namespace kubecost-exporter --create-namespace \
    --set flexera.refreshToken="Ek-aGVsbUBrdWJlY29zdC5jb20..." \
    --set flexera.orgId="1105" \
    --set flexera.billConnectId="cbi-oi-kubecost-test-1" \
    ...
```

#### 2. Pass exact parameters via custom values.yaml file:

2.1 Create a **values.yaml** file and add the necessary settings to it as below:

```yml
flexera:
    refreshToken: "xx-xxxxxxxxx"
    orgId: "7777"
    billConnectId: "cbi-oi-kubecost-1"

kubecost:
    host: "demo.kubecost.xyz"
    aggregation: "controller"
```

2.2 Apply this file when installing kubecost-exporter:

```
helm install kubecost-exporter helm-chart \
    --repo https://flexera-public.github.io/cbi-oi-kubecost-exporter/ \
    --namespace kubecost-exporter --create-namespace \
    --values values.yaml
```

## Values

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| cronSchedule | string | `"0 */6 * * *"` | Setting up a cronJob scheduler to run an export task at the right time |
| filePath | string | `"/var/kubecost"` | Filepath to mount persistent volume |
| fileRotation | bool | `true` | Delete files generated for the previous month |
| flexera.billConnectId | string | `"cbi-oi-kubecost-1"` | Bill Connect ID |
| flexera.orgId | string | `""` | Flexera Organization ID |
| flexera.refreshToken | string | `""` | Refresh Token from FlexeraOne |
| flexera.shard | string | `"NAM"` | Shard ("NAM", "EU", "AU") |
| kubecost.aggregation | string | `"controller"` | Aggregation Level ("namespace", "controller", "pod") |
| kubecost.host | string | `"kubecost-cost-analyzer.kubecost.svc.cluster.local:9090"` | Default kubecost-cost-analyzer service host on the current cluster. For current cluster is serviceName.namespaceName.svc.cluster.local |
| kubecost.idle | bool | `true` | Include cost of idle resources |
| kubecost.multiplier | float | `1` | Cost multiplier |
| kubecost.shareIdle | bool | `false` | Allocate idle cost proportionally |
| kubecost.shareNamespaces | string | `"kube-system,cadvisor"` | Comma-separated list of namespaces to share costs |
| kubecost.shareTenancyCosts | bool | `true` | Share the cost of cluster overhead assets such as cluster management costs |
| persistentVolume.enabled | bool | `true` | Enable Persistent Volume. If this setting is disabled, it may lead to inability to store history and data uploads older than 15 days in Flexera One |
| persistentVolume.size | string | `"1Gi"` | Persistent Volume size |

## License

This tool is licensed under the MIT license. See the LICENSE file for more details.
