# cbi-oi-kubecost-exporter

Kubecost exporter for Flexera CCO

## Installation

There are two different approaches for passing custom Helm config values into the kubecost-exporter:

1. Pass exact parameters via --set command-line flags:

```
helm install kubecost-exporter kubecost-exporter \
    --repo https://flexera-public.github.io/cbi-oi-kubecost-exporter/helm-chart/ \
    --namespace kubecost-exporter --create-namespace \
    --set flexera.refreshToken="Ek-aGVsbUBrdWJlY29zdC5jb20..." \
	--set flexera.orgId="1105" \
	--set flexera.billConnectId="cbi-oi-kubecost-test-1" \
    ...
```

2. Pass exact parameters via custom values.yaml file:

```
helm install kubecost-exporter kubecost-exporter \
    --repo https://flexera-public.github.io/cbi-oi-kubecost-exporter/helm-chart/ \
    --namespace kubecost-exporter --create-namespace \
    --values values.yaml
```

## Values

| Key | Type | Default | Description |
| --- | --- | --- | --- |
| cronSchedule | string | `"0 */6 * * *"` | setting up a cronJob scheduler to run an export task at the right time |
| filePath | string | `"/var/kubecost"` |  |
| fileRotation | bool | `true` |  |
| flexera.billConnectId | string | `""` | Bill Connect ID |
| flexera.host | string | `"api.flexera.com"` | IAM API Endpoint |
| flexera.orgId | string | `""` | flexera Organization ID |
| flexera.refreshToken | string | `""` | refresh Token from FlexeraOne |
| flexera.shard | string | `"NAM"` | Shard ("NAM", "EU") |
| flexera.uploadTimeout | int | `600` | file upload timeout in seconds |
| flexera.vendorName | string | `"Google"` | CSV file ManufacturerName |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"docker.io/mirrr/cbi-oi-kubecost-exporter"` |  |
| image.tag | string | `"1.0.0"` |  |
| imagePullSecrets | list | `[]` |  |
| kubecost.aggregation | string | `"controller"` | Aggregation Level ("namespace", "controller", "pod") |
| kubecost.host | string | `"kubecost-cost-analyzer.kubecost.svc.cluster.local:9090"` | default kubecost-cost-analyzer service host on the current cluster. For current cluster is <serviceName>.<namespaceName>.svc.cluster.local |
| kubecost.idle | bool | `true` | Include cost of idle resources |
| kubecost.multiplier | float | `1` | cost multiplier |
| kubecost.shareIdle | bool | `false` | Allocate idle cost proportionally |
| kubecost.shareNamespaces | string | `"kube-system,cadvisor"` | Comma-separated list of namespaces to share costs |
| kubecost.shareTenancyCosts | bool | `true` | Share the cost of cluster overhead assets such as cluster management costs |
| persistentVolume.enabled | bool | `true` |  |
| persistentVolume.size | string | `"1Gi"` |  |
