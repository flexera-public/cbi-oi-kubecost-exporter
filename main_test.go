package main

import (
	"bytes"
	"compress/gzip"
	"encoding/csv"
	"encoding/hex"
	"fmt"
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

func Test_extractLabelsWithOverride(t *testing.T) {
	type args struct {
		properties Properties
	}
	tests := []struct {
		name           string
		args           args
		expextedLabels string
	}{
		{
			name: "success: with labels and some namespace labels repeated",
			args: args{
				properties: Properties{
					Labels:          map[string]string{"label1": "us-east-1a", "label2": "us-east-1a"},
					NamespaceLabels: map[string]string{"label1": "us-east-1a-ns", "label3": "us-weast-1a"},
				},
			},
			expextedLabels: "{\"label1\":\"us-east-1a-ns\",\"label2\":\"us-east-1a\",\"label3\":\"us-weast-1a\"}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLabels(tt.args.properties, true)
			if !reflect.DeepEqual(got, tt.expextedLabels) {
				t.Errorf("extractLabels() got = %v, want %v", got, tt.expextedLabels)
			}
		})
	}
}

func TestFileWriter_CompleteWorkflow(t *testing.T) {
	a := newApp()

	fw := &FileWriter{
		filePath: "/tmp/test_complete_workflow.csv.gz",
		maxRows:  5, // Small limit to test rotation
	}
	err := fw.initFile()
	if err != nil {
		t.Fatalf("initFile() error = %v", err)
	}
	defer fw.close()

	monthOfData := "2023-10"
	filesToUpload := map[string]map[string]struct{}{
		monthOfData: make(map[string]struct{}),
	}

	// Write headers
	err = fw.writeHeaders(a.getCSVHeaders())
	if err != nil {
		t.Errorf("writeHeaders() error = %v", err)
	}

	// Write rows that will trigger rotation
	testRows := [][]string{
		{"record1", "0.1", "USD", "pod", "cpuCost", "1.0", "cpuCoreHours", "cluster1", "container1", "ns1", "pod1", "node1", "ctrl1", "deployment", "provider1", "{}", "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"},
		{"record2", "0.2", "USD", "pod", "cpuCost", "2.0", "cpuCoreHours", "cluster1", "container2", "ns1", "pod2", "node1", "ctrl2", "deployment", "provider1", "{}", "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"},
		{"record3", "0.3", "USD", "pod", "cpuCost", "3.0", "cpuCoreHours", "cluster1", "container3", "ns1", "pod3", "node1", "ctrl3", "deployment", "provider1", "{}", "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"},
		{"record4", "0.4", "USD", "pod", "cpuCost", "4.0", "cpuCoreHours", "cluster1", "container4", "ns1", "pod4", "node1", "ctrl4", "deployment", "provider1", "{}", "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"},
		{"record5", "0.5", "USD", "pod", "cpuCost", "5.0", "cpuCoreHours", "cluster1", "container5", "ns1", "pod5", "node1", "ctrl5", "deployment", "provider1", "{}", "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"},
		{"record6", "0.6", "USD", "pod", "cpuCost", "6.0", "cpuCoreHours", "cluster1", "container6", "ns1", "pod6", "node1", "ctrl6", "deployment", "provider1", "{}", "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"},
		{"record7", "0.7", "USD", "pod", "cpuCost", "7.0", "cpuCoreHours", "cluster1", "container7", "ns1", "pod7", "node1", "ctrl7", "deployment", "provider1", "{}", "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"},
	}

	for _, row := range testRows {
		err = fw.writeRow(row, monthOfData, filesToUpload)
		if err != nil {
			t.Errorf("writeRow() error = %v", err)
		}
	}

	// Finalize the current file
	fw.finalizeFile(monthOfData, filesToUpload)

	// Should have created multiple files due to rotation
	if len(filesToUpload[monthOfData]) < 2 {
		t.Errorf("Expected at least 2 files due to rotation, got %d", len(filesToUpload[monthOfData]))
	}

	// Verify all files exist and contain valid data
	totalRecordsRead := 0
	for filePath := range filesToUpload[monthOfData] {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("File %s should exist", filePath)
			continue
		}

		// Read and verify file content
		fileData, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("failed to read file %s: %v", filePath, err)
			continue
		}

		gzReader, err := gzip.NewReader(bytes.NewReader(fileData))
		if err != nil {
			t.Errorf("file %s should be valid gzip: %v", filePath, err)
			continue
		}

		csvReader := csv.NewReader(gzReader)
		records, err := csvReader.ReadAll()
		gzReader.Close()

		if err != nil {
			t.Errorf("failed to read CSV content from %s: %v", filePath, err)
			continue
		}

		if len(records) < 2 { // At least header + 1 data row
			t.Errorf("file %s should have at least 2 records, got %d", filePath, len(records))
			continue
		}

		// Verify header
		expectedHeaders := a.getCSVHeaders()
		if !reflect.DeepEqual(records[0], expectedHeaders) {
			t.Errorf("header mismatch in %s: expected %v, got %v", filePath, expectedHeaders, records[0])
		}

		// Count data records (excluding header)
		totalRecordsRead += len(records) - 1

		// Clean up
		defer os.Remove(filePath)
	}

	// Should have read all test rows
	if totalRecordsRead != len(testRows) {
		t.Errorf("Expected to read %d records total, got %d", len(testRows), totalRecordsRead)
	}
}

func TestFileWriter_GzipIntegrity(t *testing.T) {
	fw := &FileWriter{
		filePath: "/tmp/test_gzip_integrity.csv.gz",
	}
	err := fw.initFile()
	if err != nil {
		t.Fatalf("initFile() error = %v", err)
	}

	// Write some test data
	fw.writeHeaders([]string{"col1", "col2", "col3"})
	for i := 0; i < 100; i++ {
		fw.writer.Write([]string{fmt.Sprintf("val%d", i), fmt.Sprintf("val%d", i+1), fmt.Sprintf("val%d", i+2)})
	}

	monthOfData := "2023-10"
	filesToUpload := map[string]map[string]struct{}{
		monthOfData: make(map[string]struct{}),
	}

	fw.finalizeFile(monthOfData, filesToUpload)

	// Read the file and verify gzip integrity
	fileData, err := os.ReadFile(fw.filePath)
	if err != nil {
		t.Errorf("failed to read file: %v", err)
		return
	}

	// Verify it's properly compressed
	if len(fileData) == 0 {
		t.Error("compressed file should not be empty")
	}

	// Verify we can decompress it completely
	gzReader, err := gzip.NewReader(bytes.NewReader(fileData))
	if err != nil {
		t.Errorf("failed to create gzip reader: %v", err)
		return
	}
	defer gzReader.Close()

	csvReader := csv.NewReader(gzReader)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Errorf("failed to read all records: %v", err)
		return
	}

	// Should have header + 100 data rows
	if len(records) != 101 {
		t.Errorf("expected 101 records, got %d", len(records))
	}

	// Verify MD5 checksum consistency
	md5Hash1 := getMD5FromFileBytes(fileData)

	// Read file again and check MD5
	fileData2, err := os.ReadFile(fw.filePath)
	if err != nil {
		t.Errorf("failed to read file second time: %v", err)
		return
	}

	md5Hash2 := getMD5FromFileBytes(fileData2)

	if md5Hash1 != md5Hash2 {
		t.Errorf("MD5 hashes should be identical: %s != %s", md5Hash1, md5Hash2)
	}

	defer os.Remove(fw.filePath)
}

func Test_extractLabels(t *testing.T) {
	type args struct {
		properties Properties
	}
	tests := []struct {
		name           string
		args           args
		expextedLabels string
	}{
		{
			name: "success: with labels and namespace labels",
			args: args{
				properties: Properties{
					Labels:          map[string]string{"label1": "us-east-1a", "label2": "us-east-1a"},
					NamespaceLabels: map[string]string{"label3": "us-weast-1a"},
				},
			},
			expextedLabels: "{\"label1\":\"us-east-1a\",\"label2\":\"us-east-1a\",\"label3\":\"us-weast-1a\"}",
		},
		{
			name: "success: only with labels",
			args: args{
				properties: Properties{
					Labels: map[string]string{"label1": "us-east-1a", "label2": "us-east-1a"},
				},
			},
			expextedLabels: "{\"label1\":\"us-east-1a\",\"label2\":\"us-east-1a\"}",
		},
		{
			name: "success: with labels and some namespace labels repeated",
			args: args{
				properties: Properties{
					Labels:          map[string]string{"label1": "us-east-1a", "label2": "us-east-1a"},
					NamespaceLabels: map[string]string{"label1": "us-east-1a-ns", "label3": "us-weast-1a"},
				},
			},
			expextedLabels: "{\"label1\":\"us-east-1a\",\"label2\":\"us-east-1a\",\"label3\":\"us-weast-1a\"}",
		},
		{
			name: "success: with labels and some namespace labels repeated and container, controller, pod and providerID",
			args: args{
				properties: Properties{
					Labels:          map[string]string{"label1": "us-east-1a", "label2": "us-east-1a"},
					NamespaceLabels: map[string]string{"label1": "us-east-1a", "label3": "us-weast-1a"},
					Container:       "container-123",
					Controller:      "aws-controller",
					Node:            "aws-node",
					Pod:             "aws-pod",
					ProviderID:      "i-090512345ae4d14ed",
				},
			},
			expextedLabels: "{\"kc-container\":\"container-123\",\"kc-controller\":\"aws-controller\",\"kc-node\":\"aws-node\",\"kc-pod-id\":\"aws-pod\",\"kc-provider-id\":\"i-090512345ae4d14ed\",\"label1\":\"us-east-1a\",\"label2\":\"us-east-1a\",\"label3\":\"us-weast-1a\"}",
		},
		{
			name:           "success: without labels or namespace labels",
			args:           args{},
			expextedLabels: "{}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractLabels(tt.args.properties, false)
			if !reflect.DeepEqual(got, tt.expextedLabels) {
				t.Errorf("extractLabels() got = %v, want %v", got, tt.expextedLabels)
			}
		})
	}
}

func Test_newApp(t *testing.T) {
	os.Setenv("REFRESH_TOKEN", "test_refresh_token")
	os.Setenv("SERVICE_APP_CLIENT_ID", "test_service_client_id")
	os.Setenv("SERVICE_APP_CLIENT_SECRET", "test_service_client_secret")
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
	os.Setenv("KUBECOST_CONFIG_API_PATH", "/")
	os.Setenv("REQUEST_TIMEOUT", "5")
	os.Setenv("MAX_FILE_ROWS", "1000")
	os.Setenv("PAGE_SIZE", "200")
	os.Setenv("OVERRIDE_POD_LABELS", "false")
	os.Setenv("STREAM_PROCESSING_BATCH_SIZE", "50")
	os.Setenv("CREATE_BILL_CONNECT_IF_NOT_EXIST", "false")

	defer func() {
		os.Unsetenv("REFRESH_TOKEN")
		os.Unsetenv("SERVICE_APP_CLIENT_ID")
		os.Unsetenv("SERVICE_APP_CLIENT_SECRET")
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
		os.Unsetenv("REQUEST_TIMEOUT")
		os.Unsetenv("PAGE_SIZE")
		os.Unsetenv("KUBECOST_CONFIG_API_PATH")
		os.Unsetenv("OVERRIDE_POD_LABELS")
		os.Unsetenv("STREAM_PROCESSING_BATCH_SIZE")
		os.Unsetenv("CREATE_BILL_CONNECT_IF_NOT_EXIST")
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

	expectedAggregation := "cluster,namespace,controllerKind,controller"
	if a.aggregation != expectedAggregation {
		t.Errorf("Aggregation is %s, expected %s", a.aggregation, expectedAggregation)
	}

	expectedConfig := Config{
		RefreshToken:                "test_refresh_token",
		ServiceClientID:             "test_service_client_id",
		ServiceClientSecret:         "test_service_client_secret",
		OrgID:                       "test_org_id",
		BillConnectID:               "test_bill_connect_id",
		Shard:                       "NAM",
		KubecostHost:                "test_kubecost_host",
		KubecostConfigHost:          "test_kubecost_host",
		Aggregation:                 "controller",
		ShareNamespaces:             "test_namespace1,test_namespace2",
		Idle:                        true,
		IdleByNode:                  false,
		ShareIdle:                   false,
		ShareTenancyCosts:           true,
		Multiplier:                  1.0,
		FileRotation:                true,
		FilePath:                    "/var/kubecost",
		KubecostAPIPath:             "/model/",
		KubecostConfigAPIPath:       "/",
		IncludePreviousMonth:        true,
		RequestTimeout:              5,
		MaxFileRows:                 1000,
		CreateBillConnectIfNotExist: false,
		VendorName:                  "Kubecost",
		PageSize:                    200,
		DefaultCurrency:             "USD",
		OverridePodLabels:           false,
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
				includePreviousMonth: "true",
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
				date:                 time.Now().Local().AddDate(0, -1, -1),
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

func TestApp_getCSVRowsFromRecord(t *testing.T) {
	type args struct {
		currency string
		month    string
		record   KubecostAllocation
	}

	testRecord := KubecostAllocation{
		Name: "nonprod-cluster/fish",
		Properties: Properties{
			Cluster:         "nonprod-cluster",
			Node:            "aks-npu01z2-15-vmss00000z",
			Container:       "fleet-agent",
			Controller:      "fleet-agent",
			ControllerKind:  "deployment",
			Namespace:       "fish",
			Pod:             "fleet-agent-7bccbd54bc-zn8b8",
			ProviderID:      "azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			Labels:          map[string]string{"app": "fleet-agent", "crosscharge_aks": "crosscharge"},
			NamespaceLabels: map[string]string{"field_cattle_io_projectId": "p-jj7wc"},
		},
		Window: Window{
			Start: "2023-10-15T00:00:00Z",
			End:   "2023-10-16T00:00:00Z",
		},
		Start:                      "2023-10-15T00:00:00Z",
		End:                        "2023-10-16T00:00:00Z",
		Minutes:                    1440,
		CPUCoreHours:               0.13594,
		CPUCost:                    0.0977,
		CPUCostAdjustment:          0,
		GPUHours:                   0,
		GPUCost:                    0,
		GPUCostAdjustment:          0,
		NetworkTransferBytes:       230280247.65369,
		NetworkCost:                0.00004,
		NetworkCostAdjustment:      0,
		LoadBalancerCost:           0,
		LoadBalancerCostAdjustment: 0,
		PVByteHours:                0,
		PVCost:                     0,
		PVCostAdjustment:           0,
		RAMByteHours:               4053048627.2,
		RAMCost:                    0.2033,
		RAMCostAdjustment:          -0.0321,
		ExternalCost:               0,
		SharedCost:                 0.00003,
	}

	expectedLabels := "{\"app\":\"fleet-agent\",\"crosscharge_aks\":\"crosscharge\",\"field_cattle_io_projectId\":\"p-jj7wc\",\"kc-cluster\":\"nonprod-cluster\",\"kc-container\":\"fleet-agent\",\"kc-controller\":\"fleet-agent\",\"kc-controller-kind\":\"deployment\",\"kc-namespace\":\"fish\",\"kc-node\":\"aks-npu01z2-15-vmss00000z\",\"kc-pod-id\":\"fleet-agent-7bccbd54bc-zn8b8\",\"kc-provider-id\":\"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35\"}"

	var expectedRows [][]string
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.09770", "USD", "pod", "cpuCost", "0.13594", "cpuCoreHours", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			expectedLabels, "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00000", "USD", "pod", "gpuCost", "0.00000", "gpuHours", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			expectedLabels, "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.17120", "USD", "pod", "ramCost", "4053048627.20000", "ramByteHours", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			expectedLabels, "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00000", "USD", "pod", "pvCost", "0.00000", "pvByteHours", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			expectedLabels, "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00004", "USD", "pod", "networkCost", "230280247.65369", "networkTransferBytes", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			expectedLabels, "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00003", "USD", "pod", "sharedCost", "1440.00000", "minutes", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			expectedLabels, "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00000", "USD", "pod", "externalCost", "1440.00000", "minutes", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			expectedLabels, "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})
	expectedRows = append(expectedRows,
		[]string{"nonprod-cluster/fish", "0.00000", "USD", "pod", "loadBalancerCost", "1440.00000", "minutes", "nonprod-cluster", "fleet-agent", "fish", "fleet-agent-7bccbd54bc-zn8b8", "aks-npu01z2-15-vmss00000z", "fleet-agent", "deployment",
			"azure:///subscriptions/c84ced2bee05/resourceGroups/nonprod-cluster-rg/providers/virtualMachines/35",
			expectedLabels, "202310", "2023-10-15T00:00:00Z", "2023-10-15T00:00:00Z", "2023-10-16T00:00:00Z"})

	tests := []struct {
		name string
		args args
		want [][]string
	}{
		{
			name: "success: generate CSV rows from single record",
			args: args{
				currency: "USD",
				month:    "2023-10",
				record:   testRecord,
			},
			want: expectedRows,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := newApp()
			got := a.getCSVRowsFromRecord(tt.args.currency, tt.args.month, tt.args.record)
			if len(got) != len(tt.want) {
				t.Errorf("len getCSVRowsFromRecord() = %v, want %v", len(got), len(tt.want))
				return
			}

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
			args: args{month: now.AddDate(0, -1, -1).Format("2006-01")},
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

func TestGetMD5FromFileBytes(t *testing.T) {
	testCases := []struct {
		input    []byte
		expected string
	}{
		{[]byte("Hello, World!"), "65a8e27d8879283831b664bd8b7f0ad4"},
		{[]byte("12345"), "827ccb0eea8a706c4c34a16891f84e7b"},
		{[]byte(""), "d41d8cd98f00b204e9800998ecf8427e"},
		{[]byte("*/&!"), "e720300025e73ebfd5320f06e5e1919a"},
	}

	for _, testCase := range testCases {
		t.Run(hex.EncodeToString(testCase.input), func(t *testing.T) {
			result := getMD5FromFileBytes(testCase.input)
			if result != testCase.expected {
				t.Errorf("Expected: %s, Got: %s", testCase.expected, result)
			}
		})
	}
}

func TestFileWriter_initFile(t *testing.T) {
	fw := &FileWriter{
		filePath:  "/tmp/test.csv.gz",
		maxRows:   1000,
		fileIndex: 1,
	}

	err := fw.initFile()
	if err != nil {
		t.Errorf("initFile() error = %v", err)
	}

	if fw.buffer == nil {
		t.Error("buffer should be initialized")
	}
	if fw.zipWriter == nil {
		t.Error("zipWriter should be initialized")
	}
	if fw.writer == nil {
		t.Error("writer should be initialized")
	}
	if fw.rowCount != 0 {
		t.Errorf("rowCount should be 0, got %d", fw.rowCount)
	}

	fw.close()
}

func TestFileWriter_writeHeaders(t *testing.T) {
	fw := &FileWriter{}
	err := fw.initFile()
	if err != nil {
		t.Fatalf("initFile() error = %v", err)
	}
	defer fw.close()

	headers := []string{"col1", "col2", "col3"}
	err = fw.writeHeaders(headers)
	if err != nil {
		t.Errorf("writeHeaders() error = %v", err)
	}
}

func TestFileWriter_writeRow(t *testing.T) {
	fw := &FileWriter{
		maxRows: 2,
	}
	err := fw.initFile()
	if err != nil {
		t.Fatalf("initFile() error = %v", err)
	}
	defer fw.close()

	monthOfData := "2023-10"
	filesToUpload := map[string]map[string]struct{}{
		monthOfData: make(map[string]struct{}),
	}

	row1 := []string{"value1", "value2", "value3"}
	err = fw.writeRow(row1, monthOfData, filesToUpload)
	if err != nil {
		t.Errorf("writeRow() error = %v", err)
	}

	if fw.rowCount != 1 {
		t.Errorf("rowCount should be 1, got %d", fw.rowCount)
	}

	row2 := []string{"value4", "value5", "value6"}
	err = fw.writeRow(row2, monthOfData, filesToUpload)
	if err != nil {
		t.Errorf("writeRow() error = %v", err)
	}

	if fw.rowCount != 2 {
		t.Errorf("rowCount should be 2, got %d", fw.rowCount)
	}
}

func TestFileWriter_finalizeFile(t *testing.T) {
	fw := &FileWriter{
		filePath: "/tmp/test_finalize.csv.gz",
	}
	err := fw.initFile()
	if err != nil {
		t.Fatalf("initFile() error = %v", err)
	}

	fw.writeHeaders([]string{"col1", "col2"})
	fw.writer.Write([]string{"val1", "val2"})
	fw.rowCount = 1

	monthOfData := "2023-10"
	filesToUpload := map[string]map[string]struct{}{
		monthOfData: make(map[string]struct{}),
	}

	fw.finalizeFile(monthOfData, filesToUpload)

	if _, exists := filesToUpload[monthOfData][fw.filePath]; !exists {
		t.Error("file should be added to filesToUpload")
	}

	// Verify file exists and has content
	if _, err := os.Stat(fw.filePath); os.IsNotExist(err) {
		t.Error("file should exist after finalization")
	}

	// Read and verify file content
	fileData, err := os.ReadFile(fw.filePath)
	if err != nil {
		t.Errorf("failed to read file: %v", err)
	}

	if len(fileData) == 0 {
		t.Error("file should not be empty")
	}

	// Verify it's a valid gzip file by reading it
	gzReader, err := gzip.NewReader(bytes.NewReader(fileData))
	if err != nil {
		t.Errorf("file should be valid gzip: %v", err)
	}
	defer gzReader.Close()

	csvReader := csv.NewReader(gzReader)
	records, err := csvReader.ReadAll()
	if err != nil {
		t.Errorf("failed to read CSV content: %v", err)
	}

	if len(records) != 2 { // header + 1 data row
		t.Errorf("expected 2 records (header + data), got %d", len(records))
	}

	if !reflect.DeepEqual(records[0], []string{"col1", "col2"}) {
		t.Errorf("header mismatch: expected [col1 col2], got %v", records[0])
	}

	if !reflect.DeepEqual(records[1], []string{"val1", "val2"}) {
		t.Errorf("data row mismatch: expected [val1 val2], got %v", records[1])
	}

	defer os.Remove(fw.filePath)
}

func TestApp_isIdleRecord(t *testing.T) {
	a := newApp()

	tests := []struct {
		name   string
		record KubecostAllocation
		want   bool
	}{
		{
			name:   "idle record",
			record: KubecostAllocation{Name: "cluster/_idle_"},
			want:   true,
		},
		{
			name:   "non-idle record",
			record: KubecostAllocation{Name: "cluster/namespace/pod"},
			want:   false,
		},
		{
			name:   "idle record with complex name",
			record: KubecostAllocation{Name: "production-cluster/_idle_/node-123"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := a.isIdleRecord(tt.record); got != tt.want {
				t.Errorf("isIdleRecord() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileWriter_rotateFile(t *testing.T) {
	fw := &FileWriter{
		filePath:  "/tmp/test_rotate.csv.gz",
		maxRows:   2,
		fileIndex: 1,
	}

	err := fw.initFile()
	if err != nil {
		t.Fatalf("initFile() error = %v", err)
	}

	monthOfData := "2023-10"
	filesToUpload := map[string]map[string]struct{}{
		monthOfData: make(map[string]struct{}),
	}

	originalPath := fw.filePath

	err = fw.rotateFile(monthOfData, filesToUpload)
	if err != nil {
		t.Errorf("rotateFile() error = %v", err)
	}

	if fw.fileIndex != 2 {
		t.Errorf("fileIndex should be 2, got %d", fw.fileIndex)
	}

	expectedNewPath := "/tmp/test_rotate-2.csv.gz"
	if fw.filePath != expectedNewPath {
		t.Errorf("filePath should be %s, got %s", expectedNewPath, fw.filePath)
	}

	if _, exists := filesToUpload[monthOfData][originalPath]; !exists {
		t.Error("original file should be added to filesToUpload")
	}

	fw.close()
	defer os.Remove(originalPath)
	defer os.Remove(fw.filePath)
}

func TestApp_cleanupOldFiles(t *testing.T) {
	a := newApp()

	monthOfData := "2023-10"
	currentDate := "2023-10-15"

	a.filesToUpload[monthOfData] = map[string]struct{}{
		fmt.Sprintf("/tmp/kubecost-%s.csv.gz", currentDate):     {},
		fmt.Sprintf("/tmp/kubecost-%s-2.csv.gz", currentDate):   {},
		fmt.Sprintf("/tmp/kubecost-%s-old.csv.gz", currentDate): {},
		"/tmp/kubecost-2023-10-14.csv.gz":                       {},
	}

	a.cleanupOldFiles(monthOfData, currentDate)

	expectedFiles := map[string]struct{}{
		fmt.Sprintf("/tmp/kubecost-%s.csv.gz", currentDate): {},
		"/tmp/kubecost-2023-10-14.csv.gz":                   {},
	}

	if !reflect.DeepEqual(a.filesToUpload[monthOfData], expectedFiles) {
		t.Errorf("cleanupOldFiles() failed, expected %v, got %v", expectedFiles, a.filesToUpload[monthOfData])
	}
}
