# Changelog

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
