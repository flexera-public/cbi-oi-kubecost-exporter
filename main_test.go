package main

import (
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

func Test_newApp(t *testing.T) {
	os.Setenv("REFRESH_TOKEN", "test_refresh_token")
	os.Setenv("ORG_ID", "test_org_id")
	os.Setenv("BILL_CONNECT_ID", "test_bill_connect_id")
	os.Setenv("SHARD", "NAM")
	os.Setenv("KUBECOST_HOST", "test_kubecost_host")
	os.Setenv("AGGREGATION", "controller")
	os.Setenv("SHARE_NAMESPACES", "test_namespace1,test_namespace2")
	os.Setenv("IDLE", "true")
	os.Setenv("SHARE_IDLE", "false")
	os.Setenv("SHARE_TENANCY_COSTS", "true")
	os.Setenv("MULTIPLIER", "1")
	os.Setenv("FILE_ROTATION", "true")
	os.Setenv("FILE_PATH", "/var/kubecost")

	defer func() {
		os.Unsetenv("REFRESH_TOKEN")
		os.Unsetenv("ORG_ID")
		os.Unsetenv("BILL_CONNECT_ID")
		os.Unsetenv("SHARD")
		os.Unsetenv("KUBECOST_HOST")
		os.Unsetenv("AGGREGATION")
		os.Unsetenv("SHARE_NAMESPACES")
		os.Unsetenv("IDLE")
		os.Unsetenv("SHARE_IDLE")
		os.Unsetenv("SHARE_TENANCY_COSTS")
		os.Unsetenv("MULTIPLIER")
		os.Unsetenv("FILE_ROTATION")
		os.Unsetenv("FILE_PATH")
	}()

	a := newApp()

	if a.filesToUpload == nil {
		t.Error("filesToUpload is not initialized")
	}
	if a.client == nil {
		t.Error("client is not initialized")
	}
	if a.invoiceYearMonth == "" {
		t.Error("invoiceYearMonth is not initialized")
	}
	if a.aggregation == "" {
		t.Error("aggregation is not initialized")
	}

	expectedAggregation := "cluster,namespace,controller"
	if a.aggregation != expectedAggregation {
		t.Errorf("Aggregation is %s, expected %s", a.aggregation, expectedAggregation)
	}

	expectedConfig := Config{
		RefreshToken:      "test_refresh_token",
		OrgID:             "test_org_id",
		BillConnectID:     "test_bill_connect_id",
		Shard:             "NAM",
		KubecostHost:      "test_kubecost_host",
		Aggregation:       "controller",
		ShareNamespaces:   "test_namespace1,test_namespace2",
		Idle:              true,
		ShareIdle:         false,
		ShareTenancyCosts: true,
		Multiplier:        1.0,
		FileRotation:      true,
		FilePath:          "/var/kubecost",
	}
	if !reflect.DeepEqual(a.Config, expectedConfig) {
		t.Errorf("Config is %+v, expected %+v", a.Config, expectedConfig)
	}
}
