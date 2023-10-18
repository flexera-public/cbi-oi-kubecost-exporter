# cbi-oi-kubecost-exporter

![Version: 1.4.1](https://img.shields.io/badge/Version-1.4.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 1.4](https://img.shields.io/badge/AppVersion-1.4-informational?style=flat-square)

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
| --- | --- | --- | --- |
| cronSchedule | string | `"0 */6 * * *"` | Setting up a cronJob scheduler to run an export task at the right time |
| filePath | string | `"/var/kubecost"` | Filepath to mount persistent volume |
| fileRotation | bool | `true` | Delete files generated for the previous month (or the month before the previous month if INCLUDE_PREVIOUS_MONTH is set to true) |
| flexera.billConnectId | string | `"cbi-oi-kubecost-1"` | Bill Connect ID |
| flexera.orgId | string | `""` | Flexera Organization ID |
| flexera.refreshToken | string | `""` | Refresh Token from FlexeraOne You can provide the refresh token in two ways: 1. Directly as a string: refreshToken: "your_token_here" 2. Reference it from a Kubernetes secret: refreshToken: valueFrom: secretKeyRef: name: flexera-secrets # Name of the Kubernetes secret key: refresh_token # Key in the secret containing the refresh token |
| flexera.shard | string | `"NAM"` | Shard ("NAM", "EU", "AU") |
| image.pullPolicy | string | `"Always"` |  |
| image.repository | string | `"public.ecr.aws/flexera/cbi-oi-kubecost-exporter"` |  |
| image.tag | string | `"latest"` |  |
| imagePullSecrets | list | `[]` |  |
| includePreviousMonth | bool | `false` | Include data from previous month to export process |
| kubecost.aggregation | string | `"pod"` | Aggregation Level ("namespace", "controller", "pod") |
| kubecost.apiPath | string | `"/model/"` | Base path for the Kubecost API endpoints |
| kubecost.host | string | `"kubecost-cost-analyzer.kubecost.svc.cluster.local:9090"` | Default kubecost-cost-analyzer service host on the current cluster. For current cluster is serviceName.namespaceName.svc.cluster.local |
| kubecost.idle | bool | `true` | Include cost of idle resources |
| kubecost.multiplier | float | `1` | Cost multiplier |
| kubecost.shareIdle | bool | `false` | Allocate idle cost proportionally |
| kubecost.shareNamespaces | string | `"kube-system,cadvisor"` | Comma-separated list of namespaces to share costs |
| kubecost.shareTenancyCosts | bool | `true` | Share the cost of cluster overhead assets such as cluster management costs |
| persistentVolume.enabled | bool | `true` | Enable Persistent Volume. If this setting is disabled, it may lead to inability to store history and data uploads older than 15 days in Flexera One |
| persistentVolume.size | string | `"1Gi"` | Persistent Volume size |

---

Autogenerated from chart metadata using [helm-docs v1.11.3](https://github.com/norwoodj/helm-docs/releases/v1.11.3)
