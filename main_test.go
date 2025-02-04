package main

import (
	"encoding/hex"
	"os"
	"reflect"
	"testing"
	"time"
)

func Test_newApp(t *testing.T) {
	os.Setenv("REFRESH_TOKEN", "test_refresh_token")
	os.Setenv("SERVICE_APP_CLIENT_ID", "test_service_client_id")
	os.Setenv("SERVICE_APP_CLIENT_SECRET", "test_service_client_secret")
	os.Setenv("ORG_ID", "test_org_id")
	os.Setenv("BILL_CONNECT_ID", "test_bill_connect_id")
	os.Setenv("SHARD", "NAM")
	os.Setenv("CSV_FILE_PATH", "/path/to/test.csv")

	defer func() {
		os.Unsetenv("REFRESH_TOKEN")
		os.Unsetenv("SERVICE_APP_CLIENT_ID")
		os.Unsetenv("SERVICE_APP_CLIENT_SECRET")
		os.Unsetenv("ORG_ID")
		os.Unsetenv("BILL_CONNECT_ID")
		os.Unsetenv("SHARD")
		os.Unsetenv("CSV_FILE_PATH")
	}()

	a := newApp()

	if a.client == nil {
		t.Error("client is not initialized")
	}

	expectedConfig := Config{
		RefreshToken:        "test_refresh_token",
		ServiceClientId:     "test_service_client_id",
		ServiceClientSecret: "test_service_client_secret",
		OrgID:              "test_org_id",
		BillConnectID:      "test_bill_connect_id",
		Shard:              "NAM",
		CSVFilePath:        "/path/to/test.csv",
	}
	
	if !reflect.DeepEqual(a.Config, expectedConfig) {
		t.Errorf("Config is %+v, expected %+v", a.Config, expectedConfig)
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

func Test_getFlexeraDomain(t *testing.T) {
	testCases := []struct {
		shard    string
		expected string
	}{
		{"NAM", "flexera.com"},
		{"EU", "flexera.eu"},
		{"AU", "flexera.au"},
		{"DEV", "flexeratest.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.shard, func(t *testing.T) {
			a := &App{Config: Config{Shard: tc.shard}}
			result := a.getFlexeraDomain()
			if result != tc.expected {
				t.Errorf("Expected %s for shard %s, got %s", tc.expected, tc.shard, result)
			}
		})
	}
}

func Test_getOptimaAPIDomain(t *testing.T) {
	testCases := []struct {
		shard    string
		expected string
	}{
		{"NAM", "api.optima.flexeraeng.com"},
		{"EU", "api.optima-eu.flexeraeng.com"},
		{"AU", "api.optima-apac.flexeraeng.com"},
		{"DEV", "api.optima.flexeraengdev.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.shard, func(t *testing.T) {
			a := &App{Config: Config{Shard: tc.shard}}
			result := a.getOptimaAPIDomain()
			if result != tc.expected {
				t.Errorf("Expected %s for shard %s, got %s", tc.expected, tc.shard, result)
			}
		})
	}
}
