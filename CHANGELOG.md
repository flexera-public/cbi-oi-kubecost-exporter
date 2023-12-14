# Changelog

## v1.8

- The bill upload process is aborted if an error occurs during the same process. If at the time the exporter wants to start the bill upload process there is another one in progress for the same organization and period, it is aborted and a new one is started.

## v1.7

- The MD5 hash of the files is verified to ensure the integrity of the uploaded files.

## v1.6

- Save two months of cvs files instead of only current month.
- When uploading files for previous month, exporter validates that the number of files to upload match with the number of days of the previous month.

## v1.5

- Added costAdjustments for cpuCost, gpuCost, ramCost, pvCost, networkCost and loadBalancerCost
- Added new env var IDLE_BY_NODE to manage if idle allocations are created on a per node basis

## v1.4

- Time window sent to kubecost API was corrected. Was added a config var to allow send previous month of data to Flexera.

## v1.3

-   Added namespaceLabels field to the exported field "labels" in the CSV file

## v1.2

-   Added `apiPath` to `kubecost` configuration to support custom API paths.

## v1.1

-   Added AU shard and changed default aggregate value to "pod"

## v1.0

-   Initial release
