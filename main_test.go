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
	os.Setenv("SHARE_IDLE", "false")
	os.Setenv("SHARE_TENANCY_COSTS", "true")
	os.Setenv("MULTIPLIER", "1")
	os.Setenv("FILE_ROTATION", "true")
	os.Setenv("FILE_PATH", "/var/kubecost")
	os.Setenv("KUBECOST_API_PATH", "/model/")

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
		os.Unsetenv("KUBECOST_API_PATH")
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
		KubecostAPIPath:   "/model/",
	}
	if !reflect.DeepEqual(a.Config, expectedConfig) {
		t.Errorf("Config is %+v, expected %+v", a.Config, expectedConfig)
	}
}

func TestApp_dateInInvoiceRange(t *testing.T) {
	type args struct {
		allowPreviousmonth string
		date               time.Time
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "success: date in range",
			args: args{
				allowPreviousmonth: "false",
				date:               time.Now().Local().AddDate(0, 0, -1),
			},
			want: true,
		},
		{
			name: "success: date in range using previous month env var as true",
			args: args{
				allowPreviousmonth: "true",
				date:               time.Now().Local().AddDate(0, -1, 0),
			},
			want: true,
		},
		{
			name: "fail: date out of range using previous month env var as false",
			args: args{
				allowPreviousmonth: "false",
				date:               time.Now().Local().AddDate(0, -1, 0),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ALLOW_PREVIOUS_MONTH", tt.args.allowPreviousmonth)
			a := newApp()

			if got := a.dateInInvoiceRange(tt.args.date); got != tt.want {
				t.Errorf("dateInInvoiceRange() = %v, want %v", got, tt.want)
			}
			os.Unsetenv("ALLOW_PREVIOUS_MONTH")
		})
	}
}
