# Changelog

## v1.6

- Save two months of cvs files instead of only current month.
- Added new env var SEND_ONLY_FULL_PREVIOUS_MONTH to manage if we send data from previous month only if we have all the data for the previous month.
- Validate that the number of files to upload match with the number of days of the previous month, in case the above var is set to true. 

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
