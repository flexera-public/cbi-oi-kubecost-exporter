# Kubecost exporter for Flexera CCO

Kubecost Flexera Exporter is a utility to collect cost allocation data. It is a command line tool that automates the transfer of Kubernetes cluster cost allocation data to Cloud Cost Optimization. This tool generates a CSV file for each day of the current (and optionally previous) month in a format compatible with the Flexera One platform and then uploads files into Cloud Cost Optimization via CBI connect. Kubecost Flexera Exporter utilizes Kubecost Allocation API to request cost allocation data. The majority of Kubecost Allocation API parameters are exposed as exporter settings, matching Kubecost API parameters are listed in the exporter setting descriptions.

## Supported Kubecost Allocation API Versions

This exporter is able to work with 1.x, 2.2.3 or higher versions.

With versions from 2.0 to 2.2.2 there could be some unexpected results when using the Allocation API, especially using the shareIdle parameter which in those versions is not correctly implemented. That is why it is not recommended to use any of these versions.

NOTE: For versions 2.2.3 and higher, the properties received from the Kubecost Allocation API depend on the level of aggregation being used, smaller levels of granularity will include higher levels but not vice versa. For instance if you set aggregation as pod, exporter will include data for cluster, namespace, controller, controllerKind, node and pod but if you set aggregation as namespace, exporter won't include data for controller, controllerKind, node and pod.

## OpenCost Support

The Kubecost Flexera Exporter now also supports the [OpenCost API](https://www.opencost.io/docs/integrations/api), which is largely compatible with the [Kubecost Allocation API](https://docs.kubecost.com/apis/apis-overview/api-allocation). This means you can easily switch between Kubecost and OpenCost for retrieving Kubernetes cost allocation data, depending on your preference or requirements.

### Key Differences

While [OpenCost Allocation API](https://www.opencost.io/docs/integrations/api#allocation-api) mirrors that of Kubecost's for the most part, however, there are a few key differences to be aware of:

-   **Unsupported Parameters**: OpenCost does not support the following parameters:
    -   `idleByNode`
    -   `shareIdle`
    -   `shareNamespaces`
    -   `shareTenancyCosts`

These parameters are specific to Kubecost's approach to handling idle costs, shared namespace costs, and tenancy costs allocation. If your use case relies on these features, you might need to adjust your cost analysis strategy when using OpenCost. More information can be found in the [comparison table](#kubecostopencost-integration-configuration).

-   **Default Values for Certain Parameters:** When using OpenCost, it's noteworthy that certain parameters have default values distinct from used in Kubecost. Specifically:
    -   `host` should be set to `opencost.opencost.svc.cluster.local:9003`
    -   `apiPath` should default to `/`

Ensure that these values are correctly set in your configuration files to avoid connectivity or functionality issues with the OpenCost service.

## Installation

The application can be installed from golang sources, from a docker image or via the helm package manager.

### Go sources

This app requires Go version 1.21 or higher. To install, run:

```bash
go install github.com/flexera-public/cbi-oi-kubecost-exporter
```

#### Settings

The app is configured using environment variables defined in a .env file. The following configuration options are available:

| Environment Variable | Description                                                                                                                                                                                                                                                                                                                            |
| --- |----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| FILE_PATH | The path where the generated CSV files are stored. Default is "/var/kubecost"                                                                                                                                                                                                                                                          |
| FILE_ROTATION | Indicates whether to delete files generated for previous months. Default is true. Note: current and previous months data is kept.                                                                                                                                                                                                      |
| BILL_CONNECT_ID | The ID of the bill connect to which to upload the data. Default value is "cbi-oi-kubecost-1". To learn more about Bill Connect, and how to obtain your BILL_CONNECT_ID, please refer to [Creating Kubecost CBI Bill Connect](https://docs.flexera.com/flexera/EN/Optima/CreateKubecostBillConnect.htm) in the Flexera documentation.   |
| ORG_ID | The ID of your Flexera One organization, please refer to [Organization ID Unique Identifier](https://docs.flexera.com/flexera/EN/FlexeraAPI/APIKeyConcepts.htm#gettingstarted_2697534192_1120261) in the Flexera documentation.                                                                                                        |
| REFRESH_TOKEN | The refresh token used to obtain an access token for the Flexera One API. Please refer to [Generating a Refresh Token](https://docs.flexera.com/flexera/EN/FlexeraAPI/GenerateRefreshToken.htm) in the Flexera documentation.                                                                                                          |
| SERVICE_APP_CLIENT_ID | The service account client ID used to obtain an access token for the Flexera One API. Please refer to [Using a Service Account](https://docs.flexera.com/flexera/EN/FlexeraAPI/ServiceAccounts.htm?Highlight=service%20account) in the Flexera documentation. This parameter is incompatible with REFRESH_TOKEN, use only one of them. |
| SERVICE_APP_CLIENT_SECRET | The service account client secret used to obtain an access token for the Flexera One API. Please refer to [Using a Service Account](https://docs.flexera.com/flexera/EN/FlexeraAPI/ServiceAccounts.htm?Highlight=service%20account) in the Flexera documentation.                                                                      |
| SHARD | The zone of your Flexera One account. Valid values are NAM, EU or AU.                                                                                                                                                                                                                                                                  |
| INCLUDE_PREVIOUS_MONTH | Indicates whether to collect and export previous month data. Default is true. Setting this flag to false will prevent collecting and uploading the data from previous month and only upload data for the current month. Partial Data (data for some days are missing) for previous month will not be uploaded even if the flag value is set to true.|
| REQUEST_TIMEOUT | Indicates the timeout per each request in minutes.                                                                                                                                                                                                                                                                                     |
| KUBECOST_HOST | The hostname of the Kubecost instance. Default is "kubecost-cost-analyzer.kubecost.svc.cluster.local:9090".                                                                                                                                                                                                                            |
| KUBECOST_API_PATH | The base path for the Kubecost API endpoint. Default is "/model/"                                                                                                                                                                                                                                                                      |
| AGGREGATION | The level of granularity to use when aggregating the cost data. Valid values are namespace, controller, node or pod. Default is pod. Note: Exporter collects namespace labels regardless of set aggregation level and includes them into entity labels.                                                                                |
| IDLE | Indicates whether to include cost of idle resources. Valid values are true and false. Default is true.                                                                                                                                                                                                                                 |
| IDLE_BY_NODE | Indicates whether idle allocations are created on a per node basis. Valid values are true and false. Default is false.                                                                                                                                                                                                                 |
| SHARE_IDLE | Indicates whether allocate idle cost proportionally across non-idle resources. Default is false.                                                                                                                                                                                                                                       |
| SHARE_NAMESPACES | Comma-separated list of namespaces to share costs with the remaining non-idle, unshared allocations. Default = kube-system,cadvisor                                                                                                                                                                                                    |
| SHARE_TENANCY_COSTS | Indicates whether to share the cost of cluster overhead assets across tenants of those resources. Default is true.                                                                                                                                                                                                                     |
| MULTIPLIER | Optional multiplier for costs. Default is 1.                                                                                                                                                                                                                                                                                           |
| CREATE_BILL_CONNECT_IF_NOT_EXIST | Flag to enable automatic creation of Bill Connect. Default is false.                                                                                                                                                                                                                                                                                           |
| VENDOR_NAME | Vendor name for the Bill Connect. It is used when CREATE_BILL_CONNECT_IF_NOT_EXIST is set to true . Default value is "Kubecost".                                                                                                                                                                                                                                                                                           |

#### Execution

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
    --set flexera.createBillConnectIfNotExist="true" \
    --set flexera.vendorName="test-vendor" \
    --set flexera.billConnectId="cbi-oi-kubecost-..." \
    ...
```

#### 2. Pass exact parameters via custom values.yaml file:

2.1 Create a **values.yaml** file and add the necessary settings to it as below:

```yml
flexera:
    refreshToken: 'Ek-aGVsbUBrdWJlY29zdC5jb20...'
    orgId: '1105'
    billConnectId: 'cbi-oi-kubecost-...'
    createBillConnectIfNotExist: 'true'
    vendorName: 'test-vendor'

kubecost:
    aggregation: 'pod'
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

The `SCHEDULE` column should reflect the schedule you have set (default: "0 \*/24 \* \* \*"). The `NAME` column shows the name of your CronJob.

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

### Helm configuration Values

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
| includePreviousMonth | bool | `true` | Indicates whether to collect and export previous month data. Default is true. Setting this
flag to false
will prevent collecting and uploading the data from previous month and only upload data for the current month. Partial Data (data for some days are missing) for previous month will not be uploaded even if the flag value is set to true.|
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

## Kubecost/Opencost Integration Configuration

Below are the parameters used in the Helm configuration along with the environment variables that are directly employed for API requests.

| Helm Value | Environment Variable | Kubecost API Parameter | OpenCost API Parameter | Description |
| --- | --- | --- | --- | --- |
| kubecost.aggregation | AGGREGATION | `aggregate` | `aggregate` | Determines the level of granularity to use when aggregating the cost data. Valid values are namespace, controller, or pod. |
| kubecost.idle | IDLE | `idle` | `includeIdle` | Indicates whether to include the cost of idle resources in the analysis. |
| kubecost.idleByNode | IDLE_BY_NODE | `idleByNode` | - | Specifies whether idle allocations are created on a per-node basis. |
| kubecost.shareIdle | SHARE_IDLE | `shareIdle` | - | Indicates whether to allocate idle cost proportionally across non-idle resources. |
| kubecost.shareNamespaces | SHARE_NAMESPACES | `shareNamespaces` | - | Specifies a comma-separated list of namespaces to share costs with the remaining non-idle, unshared allocations. |
| kubecost.shareTenancyCosts | SHARE_TENANCY_COSTS | `shareTenancyCosts` | - | Indicates whether to share the cost of cluster overhead assets across tenants of those resources. |

## License

This tool is licensed under the MIT license. See the LICENSE file for more details.
