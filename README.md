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
-   `KUBECOST_API_PATH` - the base path for the Kubecost API endpoints
-   `BILL_CONNECT_ID` - the ID of the bill connect to which to upload the data. To learn more about Bill Connect, and how to obtain your BILL_CONNECT_ID, please refer to [this guide](https://docs.flexera.com/flexera/EN/Optima/CreateKubecostBillConnect.htm) in the Flexera documentation.
-   `SHARD` - the region of your Flexera One account. Valid values are NAM, EU or AU.
-   `ORG_ID` - the ID of your Flexera One organization.
-   `REFRESH_TOKEN` - the refresh token used to obtain an access token for the Flexera One API
-   `AGGREGATION` - the level of granularity to use when aggregating the cost data. Valid values are namespace, controller, or pod.
-   `SHARE_IDLE` - a flag indicating whether to share idle costs among namespaces
-   `SHARE_NAMESPACES` - a flag indicating whether to share namespace costs among clusters
-   `SHARE_TENANCY_COSTS` - a flag indicating whether to share tenancy costs among clusters
-   `MULTIPLIER` - a multiplier to apply to the cost data
-   `IDLE` - whether to include idle resources in the usage data. valid values are true or false.
-   `IDLE_BY_NODE` - Idle allocations are created on a per node basis.
-   `FILE_ROTATION` - whether to delete files generated during the previous month (or the month before the previous month if INCLUDE_PREVIOUS_MONTH is set to true). Valid values are true or false.
-   `FILE_PATH` - the path where the CSV files are stored
-   `INCLUDE_PREVIOUS_MONTH` - whether to include data from previous month to export process, only if we have files from every day of the previous month.. Valid values are true or false.
-   `REQUEST_TIMEOUT` - timeout per each request in minutes. Default value is 5 minutes.
 
To use this app, run:

```bash
flexera-kubecost-exporter
```

### Kubecost exporter helm chart for Kubernetes

There are two different ways to transfer custom Helm configuration values to the kubecost-exporter:

#### 1. Pass exact parameters via --set command-line flags:

```
helm install kubecost-exporter cbi-oi-kubecost-exporter \
    --repo https://flexera-public.github.io/cbi-oi-kubecost-exporter/ \
    --namespace kubecost-exporter --create-namespace \
    --set flexera.refreshToken="Ek-aGVsbUBrdWJlY29zdC5jb20..." \
    --set flexera.orgId="1105" \
    --set flexera.billConnectId="cbi-oi-kubecost-..." \
    ...
```

#### 2. Pass exact parameters via custom values.yaml file:

2.1 Create a **values.yaml** file and add the necessary settings to it as below:

```yml
flexera:
    refreshToken: "Ek-aGVsbUBrdWJlY29zdC5jb20..."
    orgId: "1105"
    billConnectId: "cbi-oi-kubecost-..."

kubecost:
    aggregation: "pod"
```

2.2 Apply this file when installing kubecost-exporter:

```
helm install kubecost-exporter cbi-oi-kubecost-exporter \
    --repo https://flexera-public.github.io/cbi-oi-kubecost-exporter/ \
    --namespace kubecost-exporter --create-namespace \
    --values values.yaml
```

### Verifying configuration

After successfully installing the helm chart, you can trigger the CronJob manually to ensure that everything is working as expected:

1. Check the schedule of the CronJob:

```
kubectl get cronjobs -n <your-namespace>
```

The `SCHEDULE` column should reflect the schedule you have set (default: "0 \*/6 \* \* \*"). The `NAME` column shows the name of your CronJob.

2. Manually create a job from the CronJob:

```
kubectl create job --from=cronjob/<your-cronjob-name> manual-001 -n <your-namespace>
```

Replace `<your-cronjob-name>` with the name of your CronJob you obtained from the previous command.

3. Monitor the logs of the job:

First, get the name of the pod associated with the job you have just created:

```
kubectl get pods -n <your-namespace>
```

Look for the pod that starts with the name "manual-001".

Then, fetch the logs:

```
kubectl logs <your-pod-name> -n <your-namespace>
```

Replace `<your-pod-name>` with the name of your pod.

You should see 200/201s in the logs, which indicates that the exporter is working as expected. This means that the CronJob will also run successfully according to its schedule.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| cronSchedule | string | `"0 */6 * * *"` | Setting up a cronJob scheduler to run an export task at the right time |
| env | object | `{}` | Pod environment variables |
| filePath | string | `"/var/kubecost"` | Filepath to mount persistent volume |
| fileRotation | bool | `true` | Delete files generated for the previous month (or the month before the previous month if INCLUDE_PREVIOUS_MONTH is set to true) |
| flexera.billConnectId | string | `"cbi-oi-kubecost-1"` | Bill Connect ID |
| flexera.orgId | string | `""` | Flexera Organization ID |
| flexera.refreshToken | string | `""` | Refresh Token from FlexeraOne You can provide the refresh token in two ways: 1. Directly as a string:    refreshToken: "your_token_here" 2. Reference it from a Kubernetes secret:    refreshToken:      valueFrom:        secretKeyRef:          name: flexera-secrets  # Name of the Kubernetes secret          key: refresh_token     # Key in the secret containing the refresh token |
| flexera.shard | string | `"NAM"` | Shard ("NAM", "EU", "AU") |
| image.pullPolicy | string | `"Always"` |  |
| image.repository | string | `"public.ecr.aws/flexera/cbi-oi-kubecost-exporter"` |  |
| image.tag | string | `"1.9"` |  |
| imagePullSecrets | list | `[]` |  |
| includePreviousMonth | bool | `false` | Include data from previous month to the export process, only if we have files from every day of the previous month. |
| kubecost.aggregation | string | `"pod"` | Aggregation Level ("namespace", "controller", "pod") |
| kubecost.apiPath | string | `"/model/"` | Base path for the Kubecost API endpoints |
| kubecost.host | string | `"kubecost-cost-analyzer.kubecost.svc.cluster.local:9090"` | Default kubecost-cost-analyzer service host on the current cluster. For current cluster is serviceName.namespaceName.svc.cluster.local |
| kubecost.idle | bool | `true` | Include cost of idle resources |
| kubecost.idleByNode | bool | `false` | Idle allocations are created on a per node basis |
| kubecost.multiplier | float | `1` | Cost multiplier |
| kubecost.shareIdle | bool | `false` | Allocate idle cost proportionally |
| kubecost.shareNamespaces | string | `"kube-system,cadvisor"` | Comma-separated list of namespaces to share costs |
| kubecost.shareTenancyCosts | bool | `true` | Share the cost of cluster overhead assets such as cluster management costs |
| persistentVolume.enabled | bool | `true` | Enable Persistent Volume. If this setting is disabled, it may lead to inability to store history and data uploads older than 15 days in Flexera One |
| persistentVolume.size | string | `"1Gi"` | Persistent Volume size |
| requestTimeout | int | `5` | Timeout per each request in minutes |

## License

This tool is licensed under the MIT license. See the LICENSE file for more details.
