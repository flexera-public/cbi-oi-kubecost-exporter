# cbi-oi-kubecost-exporter

![Version: 1.17.0](https://img.shields.io/badge/Version-1.17.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.17](https://img.shields.io/badge/AppVersion-1.17-informational?style=flat-square)

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
| activeDeadlineSeconds | int | `10800` | The maximum duration in seconds for the cron job to complete |
| cronSchedule | string | `"0 */24 * * *"` | Setting up a cronJob scheduler to run an export task at the desired time. |
| env | object | `{}` | Pod environment variables. Example using envs to use proxy: {"NO_PROXY": ".svc,.cluster.local", "HTTP_PROXY": "http://proxy.example.com:80", "HTTPS_PROXY": "http://proxy.example.com:80"} |
| filePath | string | `"/var/kubecost"` | File path to mount persistent volume. |
| fileRotation | bool | `true` | Indicates whether to delete files generated for previous months. Note: current and previous months data is kept. |
| flexera.billConnectId | string | `"cbi-oi-kubecost-1"` | The ID of the bill connect to which to upload the data. To learn more about Bill Connect, and how to obtain your BILL_CONNECT_ID, please refer to [Creating Kubecost CBI Bill Connect](https://docs.flexera.com/flexera/EN/Optima/CreateKubecostBillConnect.htm) in the Flexera documentation. |
| flexera.createBillConnectIfNotExist | string | `"false"` | Flag to enable automatic creation of Bill Connect. |
| flexera.orgId | string | `""` | The ID of your Flexera One organization, please refer to [Organization ID Unique Identifier](https://docs.flexera.com/flexera/EN/FlexeraAPI/APIKeyConcepts.htm#gettingstarted_2697534192_1120261) in the Flexera documentation. |
| flexera.refreshToken | string | `""` | The refresh token used to obtain an access token for the Flexera One API. Please refer to [Generating a Refresh Token](https://docs.flexera.com/flexera/EN/FlexeraAPI/GenerateRefreshToken.htm) in the Flexera documentation. You can provide the refresh token in two ways: 1. Directly as a string:    refreshToken: "your_token_here" 2. Reference it from a Kubernetes secret:    refreshToken:      valueFrom:        secretKeyRef:          name: flexera-secrets  # Name of the Kubernetes secret          key: refresh_token     # Key in the secret containing the refresh token |
| flexera.serviceAppClientId | string | `""` | The service account client ID used to obtain an access token for the Flexera One API. Please refer to [Using a Service Account](https://docs.flexera.com/flexera/EN/FlexeraAPI/ServiceAccounts.htm?Highlight=service%20account) in the Flexera documentation. This parameter is incompatible with **refreshToken**, use only one of them. |
| flexera.serviceAppClientSecret | string | `""` | The service account client secret used to obtain an access token for the Flexera One API. Please refer to [Using a Service Account](https://docs.flexera.com/flexera/EN/FlexeraAPI/ServiceAccounts.htm?Highlight=service%20account) in the Flexera documentation. This parameter is incompatible with **refreshToken**, use only one of them. |
| flexera.shard | string | `"NAM"` | The zone of your Flexera One account. Valid values are NAM, EU or AU. |
| flexera.vendorName | string | `"Kubecost"` | Vendor name for the Bill Connect. It is used when CREATE_BILL_CONNECT_IF_NOT_EXIST is set to true. |
| image.pullPolicy | string | `"Always"` |  |
| image.repository | string | `"public.ecr.aws/flexera/cbi-oi-kubecost-exporter"` |  |
| image.tag | string | `"1.17"` |  |
| imagePullSecrets | list | `[]` |  |
| includePreviousMonth | bool | `false` | Indicates whether to collect and export previous month. |
| kubecost.aggregation | string | `"pod"` | The level of granularity to use when aggregating the cost data. Valid values are namespace, controller, node, or pod. |
| kubecost.apiPath | string | `"/model/"` | The base path for the Kubecost API endpoint. |
| kubecost.host | string | `"kubecost-cost-analyzer.kubecost.svc.cluster.local:9090"` | Default kubecost-cost-analyzer service host on the current cluster. For current cluster is serviceName.namespaceName.svc.cluster.local |
| kubecost.idle | bool | `true` | Indicates whether to include cost of idle resources. |
| kubecost.idleByNode | bool | `false` | Indicates whether idle allocations are created on a per node basis. |
| kubecost.multiplier | float | `1` | Optional multiplier for costs. |
| kubecost.shareIdle | bool | `false` | Indicates whether allocate idle cost proportionally across non-idle resources. |
| kubecost.shareNamespaces | string | `"kube-system,cadvisor"` | Comma-separated list of namespaces to share costs with the remaining non-idle, unshared allocations. |
| kubecost.shareTenancyCosts | bool | `true` | Indicates whether to share the cost of cluster overhead assets across tenants of those resources. |
| persistentVolume.enabled | bool | `true` | Enable Persistent Volume. Recommended setting is true to prevent loss of historical data. |
| persistentVolume.size | string | `"1Gi"` | Persistent Volume size. |
| requestTimeout | int | `5` | Indicates the timeout per each request in minutes. |
