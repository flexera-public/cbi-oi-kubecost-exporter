# Changelog

## v1.19.0

-   Removed label kc-provider-id

## v1.18.0

-   Changed the default schedule of the cronJob scheduler to run an export task to - Once in every 24 hours.

## v1.17.0

-   Added labels "kc-cluster" for cluster-name, "kc-namespace" for namespace to the exported field "labels" in the CSV file.

## v1.16.0

-   Allow creation of Bill Connect (if not existing) conditionally based on flag CREATE_BILL_CONNECT_IF_NOT_EXIST
-   To support the creation of Bill Connect, introduced additional Parameter - VENDOR_NAME. This parameter is only used when the CREATE_BILL_CONNECT_IF_NOT_EXIST is set to true.
-   Added query parameter 'accumulate' for Kubecost Allocation API requests to sum the entire range of time intervals into a single set and reduce response size.

## v1.15.0

-   Added some modifications building aggregation query param to address Kubecost Allocation API changes made for versions 2.3 and higher.

## v1.14.0

-   Improved OpenCost support. Added description of settings for integration with OpenCost in README.md

## v1.13.0

-   Fixed crash on getConfigs request for OpenCost

## v1.12.0

-   Implemented compression and chunking for large CSV files to improve file upload reliability and performance with Flexera CBI.

## v1.11.1

-   Enhanced Helm chart to dynamically configure `activeDeadlineSeconds` through `values.yaml`, providing greater flexibility for cron job execution duration.

## v1.11.0

-   Added labels "kc-container" for container, "kc-controller" for controller, "kc-node" for node, "kc-pod-id" for pod and "kc-provider-id" for providerID to the exported field "labels" in the CSV file.

## v1.10.0

-   Added Flexera Service Accounts support (SERVICE_APP_CLIENT_ID and SERVICE_APP_CLIENT_SECRET env variables)

## v1.9.1

-   Addressed an issue with Kubecost where it returns "empty" data for selected date ranges. Also ensured that no data is written to files in cases where the data is empty.

## v1.9

-   Added new env var REQUEST_TIMEOUT to manage the timeout of the requests. Default value is 5 minutes.
-   Added new query param 'step' set to '1d' to get daily data metrics from kubecost API.

## v1.8

-   The bill upload process is aborted if an error occurs during the same process. If at the time the exporter wants to start the bill upload process there is another one in progress for the same organization and period, it is aborted and a new one is started.

## v1.7

-   The MD5 hash of the files is verified to ensure the integrity of the uploaded files.

## v1.6

-   Save two months of cvs files instead of only current month.
-   When uploading files for previous month, exporter validates that the number of files to upload match with the number of days of the previous month.

## v1.5

-   Added costAdjustments for cpuCost, gpuCost, ramCost, pvCost, networkCost and loadBalancerCost
-   Added new env var IDLE_BY_NODE to manage if idle allocations are created on a per node basis

## v1.4

-   Time window sent to kubecost API was corrected. Was added a config var to allow send previous month of data to Flexera.

## v1.3

-   Added namespaceLabels field to the exported field "labels" in the CSV file

## v1.2

-   Added `apiPath` to `kubecost` configuration to support custom API paths.

## v1.1

-   Added AU shard and changed default aggregate value to "pod"

## v1.0

-   Initial release
