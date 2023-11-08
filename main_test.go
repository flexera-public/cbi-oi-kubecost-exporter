package main

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"
)

func Test_dateIter(t *testing.T) {
	startDate := time.Now().AddDate(0, 0, -30)
	i := 0
	for date := range dateIter(startDate) {
		expectedDate := startDate.AddDate(0, 0, i)
		if !date.Equal(expectedDate) {
			t.Errorf("Expected date %v, but got %v", expectedDate, date)
		}
		i++
	}
}

func Test_extractLabels(t *testing.T) {
	type args struct {
		labels          map[string]string
		namespaceLabels map[string]string
	}
	tests := []struct {
		name           string
		args           args
		expextedLabels string
	}{
		{
			name: "success: with labels and namespace labels",
			args: args{
				labels:          map[string]string{"label1": "us-east-1a", "label2": "us-east-1a"},
				namespaceLabels: map[string]string{"label3": "us-weast-1a"},
			},
			expextedLabels: "{\"label1\":\"us-east-1a\",\"label2\":\"us-east-1a\",\"label3\":\"us-weast-1a\"}",
		},
		{
			name: "success: only with labels",
			args: args{
				labels: map[string]string{"label1": "us-east-1a", "label2": "us-east-1a"},
			},
			expextedLabels: "{\"label1\":\"us-east-1a\",\"label2\":\"us-east-1a\"}",
		},
		{
			name: "success: with labels and some namespace labels repeated",
			args: args{
				labels:          map[string]string{"label1": "us-east-1a", "label2": "us-east-1a"},
				namespaceLabels: map[string]string{"label1": "us-east-1a", "label3": "us-weast-1a"},
			},
			expextedLabels: "{\"label1\":\"us-east-1a\",\"label2\":\"us-east-1a\",\"label3\":\"us-weast-1a\"}",
		},
		{
			name:           "success: without labels or namespace labels",
			args:           args{},
			expextedLabels: "{}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLabels(tt.args.labels, tt.args.namespaceLabels)
			if !reflect.DeepEqual(got, tt.expextedLabels) {
				t.Errorf("extractLabels() got = %v, want %v", got, tt.expextedLabels)
			}
		})
	}
}

func Test_newApp(t *testing.T) {
	os.Setenv("REFRESH_TOKEN", "test_refresh_token")
	os.Setenv("ORG_ID", "test_org_id")
	os.Setenv("BILL_CONNECT_ID", "test_bill_connect_id")
	os.Setenv("SHARD", "NAM")
	os.Setenv("KUBECOST_HOST", "test_kubecost_host")
	os.Setenv("AGGREGATION", "controller")
	os.Setenv("SHARE_NAMESPACES", "test_namespace1,test_namespace2")
	os.Setenv("IDLE", "true")
	os.Setenv("IDLE_BY_NODE", "false")
	os.Setenv("SHARE_IDLE", "false")
	os.Setenv("SHARE_TENANCY_COSTS", "true")
	os.Setenv("MULTIPLIER", "1")
	os.Setenv("FILE_ROTATION", "true")
	os.Setenv("FILE_PATH", "/var/kubecost")
	os.Setenv("KUBECOST_API_PATH", "/model/")
	os.Setenv("SEND_ONLY_FULL_PREVIOUS_MONTH", "true")

	defer func() {
		os.Unsetenv("REFRESH_TOKEN")
		os.Unsetenv("ORG_ID")
		os.Unsetenv("BILL_CONNECT_ID")
		os.Unsetenv("SHARD")
		os.Unsetenv("KUBECOST_HOST")
		os.Unsetenv("AGGREGATION")
		os.Unsetenv("SHARE_NAMESPACES")
		os.Unsetenv("IDLE")
		os.Unsetenv("IDLE_BY_NODE")
		os.Unsetenv("SHARE_IDLE")
		os.Unsetenv("SHARE_TENANCY_COSTS")
		os.Unsetenv("MULTIPLIER")
		os.Unsetenv("FILE_ROTATION")
		os.Unsetenv("FILE_PATH")
		os.Unsetenv("KUBECOST_API_PATH")
		os.Unsetenv("SEND_ONLY_FULL_PREVIOUS_MONTH")
	}()

	a := newApp()

	if a.filesToUpload == nil {
		t.Error("filesToUpload is not initialized")
	}
	if a.client == nil {
		t.Error("client is not initialized")
	}
	if a.aggregation == "" {
		t.Error("aggregation is not initialized")
	}

	expectedAggregation := "cluster,namespace,controller"
	if a.aggregation != expectedAggregation {
		t.Errorf("Aggregation is %s, expected %s", a.aggregation, expectedAggregation)
	}

	expectedConfig := Config{
		RefreshToken:              "test_refresh_token",
		OrgID:                     "test_org_id",
		BillConnectID:             "test_bill_connect_id",
		Shard:                     "NAM",
		KubecostHost:              "test_kubecost_host",
		Aggregation:               "controller",
		ShareNamespaces:           "test_namespace1,test_namespace2",
		Idle:                      true,
		IdleByNode:                false,
		ShareIdle:                 false,
		ShareTenancyCosts:         true,
		Multiplier:                1.0,
		FileRotation:              true,
		FilePath:                  "/var/kubecost",
		KubecostAPIPath:           "/model/",
		IncludePreviousMonth:      false,
		SendOnlyFullPreviousMonth: true,
	}
	if !reflect.DeepEqual(a.Config, expectedConfig) {
		t.Errorf("Config is %+v, expected %+v", a.Config, expectedConfig)
	}
}

func TestApp_dateInInvoiceRange(t *testing.T) {
	type args struct {
		includePreviousMonth string
		date                 time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success: date in range",
			args: args{
				includePreviousMonth: "false",
				date:                 time.Now().Local().AddDate(0, 0, -1),
			},
			want: true,
		},
		{
			name: "success: date in range using previous month env var as true",
			args: args{
				includePreviousMonth: "true",
				date:                 time.Now().Local().AddDate(0, -1, 0),
			},
			want: true,
		},
		{
			name: "fail: date out of range using previous month env var as false",
			args: args{
				includePreviousMonth: "false",
				date:                 time.Now().Local().AddDate(0, -1, 0),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("INCLUDE_PREVIOUS_MONTH", tt.args.includePreviousMonth)
			a := newApp()

			if got := a.dateInInvoiceRange(tt.args.date); got != tt.want {
				t.Errorf("dateInInvoiceRange() = %v, want %v", got, tt.want)
			}
			os.Unsetenv("INCLUDE_PREVIOUS_MONTH")
		})
	}
}

func TestApp_getCSVRows(t *testing.T) {
	type args struct {
		currency string
		month    string
		data     []map[string]KubecostAllocation
	}

	dataJson := `[{
            "nonprod-cluster/fish": {
                "name": "nonprod-cluster/fish",
                "properties": {
                    "cluster": "nonprod-cluster",
                    "node": "aks-npu01z2-15-vmss00000z",
                    "container": "fleet-agent",
                    "controller": "fleet-agent",
                    "controllerKind": "deployment",
                    "namespace": "fish",
                    "pod": "fleet-agent-7bccbd54bc-zn8b8",
                    "providerID": "azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
                    "labels": {
                        "app": "fleet-agent",
                        "crosscharge_aks": "crosscharge"
                    },
                    "namespaceLabels": {
                        "field_cattle_io_projectId": "p-jj7wc"
                    }
                },
                "window": {
                    "start": "2023-10-15T00:00:00Z",
                    "end": "2023-10-16T00:00:00Z"
                },
                "start": "2023-10-15T00:00:00Z",
                "end": "2023-10-16T00:00:00Z",
                "minutes": 1440,
                "cpuCores": 0.00566,
                "cpuCoreRequestAverage": 0,
                "cpuCoreUsageAverage": 0.00664,
                "cpuCoreHours": 0.13594,
                "cpuCost": 0.0977,
                "cpuCostAdjustment": -0.0235,
                "cpuEfficiency": 1,
                "gpuCount": 0,
                "gpuHours": 0,
                "gpuCost": 0,
                "gpuCostAdjustment": 0,
                "networkTransferBytes": 230280247.65369,
                "networkReceiveBytes": 3834780812.31544,
                "networkCost": 0.00004,
                "networkCrossZoneCost": 0,
                "networkCrossRegionCost": 0,
                "networkInternetCost": 0.00004,
                "networkCostAdjustment": 0,
                "loadBalancerCost": 0,
                "loadBalancerCostAdjustment": 0,
                "pvBytes": 0,
                "pvByteHours": 0,
                "pvCost": 0,
                "pvs": null,
                "pvCostAdjustment": 0,
                "ramBytes": 168877026.13333,
                "ramByteRequestAverage": 0,
                "ramByteUsageAverage": 180387256.21762,
                "ramByteHours": 4053048627.2,
                "ramCost": 0.2033,
                "ramCostAdjustment": -0.0321,
                "ramEfficiency": 1,
                "externalCost": 0,
                "sharedCost": 0.00003,
                "totalCost": 0.0246,
                "totalEfficiency": 1,
                "rawAllocationOnly": {
                    "cpuCoreUsageMax": 0.04475121093829057,
                    "ramByteUsageMax": 320245760
                },
                "lbAllocations": null
            }
        }
    ]`

	var data []map[string]KubecostAllocation
	err := json.Unmarshal([]byte(dataJson), &data)
	if err != nil {
		t.Errorf("Error unmarshalling data: %v", err)
	}

	var expectedRows [][]string
	expectedRows = make([][]string, 0)
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.07", "USD", "pod", "cpuCost", "0.14", "cpuCoreHours", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			"{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\"}",
			"202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00", "USD", "pod", "gpuCost", "0.00", "gpuHours", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			"{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\"}",
			"202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.17", "USD", "pod", "ramCost", "4053048627.20", "ramByteHours", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			"{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\"}",
			"202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00", "USD", "pod", "pvCost", "0.00", "pvByteHours", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			"{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\"}",
			"202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00", "USD", "pod", "networkCost", "230280247.65", "networkTransferBytes", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			"{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\"}",
			"202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00", "USD", "pod", "sharedCost", "1440.00", "minutes", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			"{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\"}",
			"202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00", "USD", "pod", "externalCost", "1440.00", "minutes", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			"{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\"}",
			"202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00", "USD", "pod", "loadBalancerCost", "1440.00", "minutes", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			"{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\"}",
			"202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})

	tests := []struct {
		name string
		args args
		want [][]string
	}{
		{
			name: "success: date in range",
			args: args{
				currency: "USD",
				month:    "2023-10",
				data:     data,
			},
			want: expectedRows,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApp()

			got := a.getCSVRows(tt.args.currency, tt.args.month, tt.args.data)
			if len(got) != len(tt.want) {
				t.Errorf("len getCSVRows() = %v, want %v", len(got), len(tt.want))
				return
			}
			// Find all records got in want
			for _, rowGot := range got {
				found := false
				for _, rowWant := range tt.want {
					if reflect.DeepEqual(rowGot, rowWant) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("row %v not found in want", rowGot)
				}

			}

		})
	}
}

func TestApp_dateInMandatoryFileSavingPeriod(t *testing.T) {
	type args struct {
		customStartDateOfMandatoryPeriod *time.Time
		date                             time.Time
	}

	firstDayOfMonth, _ := time.Parse("2006-01-02", "2023-10-01")
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success: date in mandatory period",
			args: args{
				customStartDateOfMandatoryPeriod: nil,
				date:                             time.Now().Local().AddDate(0, 0, -1),
			},
			want: true,
		},
		{
			name: "success: one month before current date",
			args: args{
				customStartDateOfMandatoryPeriod: nil,
				date:                             time.Now().Local().AddDate(0, -1, 0),
			},
			want: true,
		},
		{
			name: "fail: one day before the mandatory period",
			args: args{
				customStartDateOfMandatoryPeriod: &firstDayOfMonth,
				date:                             firstDayOfMonth.AddDate(0, 0, -1),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApp()
			if tt.args.customStartDateOfMandatoryPeriod != nil {
				a.mandatoryFileSavingPeriodStartDate = *tt.args.customStartDateOfMandatoryPeriod
			}
			if got := a.dateInMandatoryFileSavingPeriod(tt.args.date); got != tt.want {
				t.Errorf("dateInMandatoryFileSavingPeriod() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApp_isCurrentMonth(t *testing.T) {
	type args struct {
		month string
	}
	now := time.Now().Local()
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success: current month",
			args: args{month: now.Format("2006-01")},
			want: true,
		},
		{
			name: "fail: previous month",
			args: args{month: now.AddDate(0, -1, 0).Format("2006-01")},
			want: false,
		},
		{
			name: "fail: next month",
			args: args{month: now.AddDate(0, 1, 0).Format("2006-01")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApp()
			if got := a.isCurrentMonth(tt.args.month); got != tt.want {
				t.Errorf("isCurrentMonth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApp_DaysInMonth(t *testing.T) {
	type args struct {
		month string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "success: October 2023",
			args: args{month: "2023-10"},
			want: 31,
		},
		{
			name: "success: February 2023",
			args: args{month: "2023-02"},
			want: 28,
		},
		{
			name: "success: February 2024",
			args: args{month: "2024-02"},
			want: 29,
		},
		{
			name: "success: November 2023",
			args: args{month: "2023-11"},
			want: 30,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApp()
			if got := a.DaysInMonth(tt.args.month); got != tt.want {
				t.Errorf("DaysInMonth() = %v, want %v", got, tt.want)
			}
		})
	}
}
