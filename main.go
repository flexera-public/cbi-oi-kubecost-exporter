package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v7"
)

type (
	KubecostAllocationResponse struct {
		Code    int64                           `json:"code"`
		Message string                          `json:"message"`
		Data    []map[string]KubecostAllocation `json:"data"`
	}

	KubecostAllocation struct {
		Name                       string        `json:"name"`
		Properties                 Properties    `json:"properties"`
		Window                     Window        `json:"window"`
		Start                      string        `json:"start"`
		End                        string        `json:"end"`
		Minutes                    float64       `json:"minutes"`
		CPUCores                   float64       `json:"cpuCores"`
		CPUCoreRequestAverage      float64       `json:"cpuCoreRequestAverage"`
		CPUCoreUsageAverage        float64       `json:"cpuCoreUsageAverage"`
		CPUCoreHours               float64       `json:"cpuCoreHours"`
		CPUCost                    float64       `json:"cpuCost"`
		CPUCostAdjustment          float64       `json:"cpuCostAdjustment"`
		CPUEfficiency              float64       `json:"cpuEfficiency"`
		GPUCount                   float64       `json:"gpuCount"`
		GPUHours                   float64       `json:"gpuHours"`
		GPUCost                    float64       `json:"gpuCost"`
		GPUCostAdjustment          float64       `json:"gpuCostAdjustment"`
		NetworkTransferBytes       float64       `json:"networkTransferBytes"`
		NetworkReceiveBytes        float64       `json:"networkReceiveBytes"`
		NetworkCost                float64       `json:"networkCost"`
		NetworkCrossZoneCost       float64       `json:"networkCrossZoneCost"`
		NetworkCrossRegionCost     float64       `json:"networkCrossRegionCost"`
		NetworkInternetCost        float64       `json:"networkInternetCost"`
		NetworkCostAdjustment      float64       `json:"networkCostAdjustment"`
		LoadBalancerCost           float64       `json:"loadBalancerCost"`
		LoadBalancerCostAdjustment float64       `json:"loadBalancerCostAdjustment"`
		PVBytes                    float64       `json:"pvBytes"`
		PVByteHours                float64       `json:"pvByteHours"`
		PVCost                     float64       `json:"pvCost"`
		PVs                        map[string]PV `json:"pvs"`
		PVCostAdjustment           float64       `json:"pvCostAdjustment"`
		RAMBytes                   float64       `json:"ramBytes"`
		RAMByteRequestAverage      float64       `json:"ramByteRequestAverage"`
		RAMByteUsageAverage        float64       `json:"ramByteUsageAverage"`
		RAMByteHours               float64       `json:"ramByteHours"`
		RAMCost                    float64       `json:"ramCost"`
		RAMCostAdjustment          float64       `json:"ramCostAdjustment"`
		RAMEfficiency              float64       `json:"ramEfficiency"`
		SharedCost                 float64       `json:"sharedCost"`
		ExternalCost               float64       `json:"externalCost"`
		TotalCost                  float64       `json:"totalCost"`
		TotalEfficiency            float64       `json:"totalEfficiency"`
		RawAllocationOnly          interface{}   `json:"rawAllocationOnly"`
	}

	PV struct {
		ByteHours float64 `json:"byteHours"`
		Cost      float64 `json:"cost"`
	}

	Properties struct {
		Cluster        string            `json:"cluster"`
		Container      string            `json:"container"`
		Namespace      string            `json:"namespace"`
		Pod            string            `json:"pod"`
		Node           string            `json:"node"`
		Controller     string            `json:"controller"`
		ControllerKind string            `json:"controllerKind"`
		Services       []string          `json:"services"`
		ProviderID     string            `json:"providerID"`
		Labels         map[string]string `json:"labels"`
	}

	Window struct {
		Start string `json:"start"`
		End   string `json:"end"`
	}

	KubecostConfig struct {
		Data struct {
			CurrencyCode string `json:"currencyCode"`
		}
	}

	Config struct {
		RefreshToken      string  `env:"REFRESH_TOKEN"`
		OrgID             string  `env:"ORG_ID"`
		BillConnectID     string  `env:"BILL_CONNECT_ID"`
		UploadTimeout     int64   `env:"UPLOAD_TIMEOUT" envDefault:"600"`
		Shard             string  `env:"SHARD" envDefault:"NAM"`
		VendorName        string  `env:"VENDOR_NAME" envDefault:"Google"`
		KubecostHost      string  `env:"KUBECOST_HOST" envDefault:"localhost:9090"`
		Aggregation       string  `env:"AGGREGATION" envDefault:"controller"`
		ShareNamespaces   string  `env:"SHARE_NAMESPACES" envDefault:"kube-system,cadvisor"`
		Idle              bool    `env:"IDLE" envDefault:"true"`
		ShareIdle         bool    `env:"SHARE_IDLE" envDefault:"false"`
		ShareTenancyCosts bool    `env:"SHARE_TENANCY_COSTS" envDefault:"true"`
		Multiplier        float64 `env:"MULTIPLIER" envDefault:"1.0"`
		FileRotation      bool    `env:"FILE_ROTATION" envDefault:"true"`
		FilePath          string  `env:"FILE_PATH" envDefault:"/var/kubecost"`
	}

	App struct {
		Config
		resourceType     string
		filesToUpload    map[string]struct{}
		client           *http.Client
		invoiceYearMonth string
	}
)

func main() {
	exporter := newApp()
	exporter.updateFileList()
	exporter.updateFromKubecost()
	exporter.uploadToFlexera()
}

func (e *App) updateFromKubecost() {
	now := time.Now().Local()

	err := os.MkdirAll(e.FilePath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	curr, err := e.getCurrency()
	if err != nil {
		log.Fatal(err)
	}

	for d := range dateIter(now.AddDate(0, -1, 0)) {
		if d.After(now) || d.Format("2006-01") != e.invoiceYearMonth {
			continue
		}

		tomorrow := d.AddDate(0, 0, 1)

		// https://github.com/kubecost/docs/blob/master/allocation.md#querying
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/model/allocation", e.KubecostHost), nil)
		if err != nil {
			log.Fatal(err)
		}
		q := req.URL.Query()
		q.Add("window", fmt.Sprintf("%s,%s", d.Format("2006-01-02T15:04:05Z"), tomorrow.Format("2006-01-02T15:04:05Z")))
		q.Add("aggregate", e.Aggregation)
		q.Add("idle", fmt.Sprintf("%t", e.Idle))
		q.Add("shareIdle", fmt.Sprintf("%t", e.ShareIdle))
		q.Add("shareNamespaces", e.ShareNamespaces)
		q.Add("shareSplit", "weighted")
		q.Add("shareTenancyCosts", fmt.Sprintf("%t", e.ShareTenancyCosts))
		req.URL.RawQuery = q.Encode()

		resp, err := e.client.Do(req)
		if err != nil {
			log.Fatal(err)
		}

		var j KubecostAllocationResponse
		err = json.NewDecoder(resp.Body).Decode(&j)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		data := j.Data
		b := new(bytes.Buffer)
		writer := csv.NewWriter(b)

		writer.Write([]string{
			"CloudVendorAccountID",
			"CloudVendorAccountName",
			"Category",
			"LineItemType",
			"ResourceGroup",
			"ResourceType",
			"ResourceID",
			"Service",
			"UsageType",
			"Tags",
			"UsageAmount",
			"UsageUnit",
			"Cost",
			"CurrencyCode",
			"UsageStartTime",
			"InvoiceYearMonth",
			"ManufacturerName",
		})

		for _, allocation := range data {
			for id, v := range allocation {
				tagsJSON, _ := json.Marshal(v.Properties.Labels)
				tags := string(tagsJSON)
				types := []string{"cpuCost", "gpuCost", "ramCost", "pvCost", "networkCost", "sharedCost", "loadBalancerCost"}
				vals := []float64{v.CPUCost, v.GPUCost, v.RAMCost, v.PVCost, v.NetworkCost, v.SharedCost, v.LoadBalancerCost}
				units := []string{"cpuCoreHours", "gpuHours", "ramByteHours", "pvByteHours", "networkTransferBytes", "minutes", "minutes"}
				amounts := []float64{v.CPUCoreHours, v.GPUHours, v.RAMByteHours, v.PVByteHours, v.NetworkTransferBytes, v.Minutes, v.Minutes}

				for i, c := range types {
					multiplierFloat := e.Multiplier * vals[i]
					if v.Properties.Cluster == "" {
						v.Properties.Cluster = "Cluster"
					}
					writer.Write([]string{
						v.Properties.Cluster,
						"kubernetes",
						"Compute",
						"Usage",
						strings.Split(id, "/")[0],
						e.resourceType,
						id,
						"Kubernetes",
						c,
						tags,
						strconv.FormatFloat(amounts[i], 'f', 2, 64),
						units[i],
						strconv.FormatFloat(multiplierFloat, 'f', 2, 64),
						curr,
						d.Format("2006-01-02T15:04:05Z"),
						e.invoiceYearMonth,
						e.VendorName,
					})
				}
			}
		}

		writer.Flush()
		var csvFile = fmt.Sprintf(path.Join(e.FilePath, "kubecost-%v.csv"), d.Format("2006-01-02"))
		log.Println(csvFile)
		e.filesToUpload[csvFile] = struct{}{}
		err = os.WriteFile(csvFile, b.Bytes(), 0644)
		if err != nil {
			log.Fatal(err)
		}
		b.Reset()
	}
}

func (e *App) uploadToFlexera() {
	accessToken, err := e.generateAccessToken()
	if err != nil {
		log.Fatalf("Error generating access token: %v", err)
	}

	shardDict := map[string]string{
		"NAM": "api.optima.flexeraeng.com",
		"EU":  "api.optima-eu.flexeraeng.com",
	}

	billUploadURL := fmt.Sprintf("https://%s/optima/orgs/%s/billUploads", shardDict[strings.ToUpper(e.Shard)], e.OrgID)

	authHeaders := map[string]string{"Authorization": "Bearer " + accessToken}
	billUpload := map[string]string{"billConnectId": e.BillConnectID, "billingPeriod": e.invoiceYearMonth}

	billUploadJSON, _ := json.Marshal(billUpload)
	response := e.doPost(billUploadURL, string(billUploadJSON), authHeaders, time.Minute)
	existingID := ""

	switch response.StatusCode {
	case 429:
		time.Sleep(120 * time.Second)
		response = e.doPost(billUploadURL, string(billUploadJSON), authHeaders)
		checkForError(response)
	case 409:
		existingID = strings.Split(response.Header.Get("message"), " ")[4][:len(response.Header.Get("message"))-1]
	default:
		checkForError(response)
	}

	var billUploadID string
	if existingID != "" {
		billUploadID = existingID
	} else {

		bodyBytes, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.Fatal(err)
		}
		var jsonResponse map[string]string
		json.Unmarshal(bodyBytes, &jsonResponse)
		billUploadID = jsonResponse["id"]
	}

	for fileName := range e.filesToUpload {
		baseName := filepath.Base(fileName)
		uploadFileURL := fmt.Sprintf("%s/%s/files/%s", billUploadURL, billUploadID, baseName)

		fileData, _ := ioutil.ReadFile(fileName)
		response = e.doPost(uploadFileURL, string(fileData), authHeaders, time.Duration(e.UploadTimeout)*time.Second)
		checkForError(response)
	}

	operationsURL := fmt.Sprintf("%s/%s/operations", billUploadURL, billUploadID)
	response = e.doPost(operationsURL, `{"operation":"commit"}`, authHeaders)
	checkForError(response)

	os.Exit(0)
}

func (a *App) doPost(url, data string, headers map[string]string, uploadTimeout ...time.Duration) *http.Response {
	log.Printf("---> %+v\n", url)
	client := a.client
	if len(uploadTimeout) > 0 {
		client = &http.Client{
			Timeout: uploadTimeout[0],
		}
	}
	request, _ := http.NewRequest("POST", url, strings.NewReader(data))

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, _ := client.Do(request)
	log.Printf("<--- %+v\n", response.StatusCode)
	return response
}

// generateAccessToken returns an access token from the Flexera One API using a given refreshToken.
func (a *App) generateAccessToken() (string, error) {
	accessTokenUrl := "https://login.flexera.com/oidc/token"
	reqBody := url.Values{}
	reqBody.Set("grant_type", "refresh_token")
	reqBody.Set("refresh_token", a.RefreshToken)

	log.Printf("---> %+v\n", a.RefreshToken)

	req, err := http.NewRequest("POST", accessTokenUrl, strings.NewReader(reqBody.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating access token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error retrieving access token: %v", err)
	}

	log.Printf("---> %+v\n", resp)

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error retrieving access token: %v", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading access token response body: %v", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("error parsing access token response body: %v", err)
	}

	log.Printf("%+v\n", tokenResp.AccessToken)

	return tokenResp.AccessToken, nil
}

// update file list and remove old files
func (a *App) updateFileList() {
	now := time.Now().Local()
	files, err := ioutil.ReadDir(a.FilePath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.Mode().IsRegular() {
			if t, err := time.Parse("kubecost-2006-01-02.csv", file.Name()); err == nil {
				if t.Month() == now.Month() {
					a.filesToUpload[path.Join(a.FilePath, file.Name())] = struct{}{}
				} else if a.FileRotation {
					if err = os.Remove(path.Join(a.FilePath, file.Name())); err != nil {
						log.Printf("error removing file %s: %v", file.Name(), err)
					}
				}
			}
		}
	}
}

func (a *App) getCurrency() (string, error) {
	resp, err := a.client.Get(fmt.Sprintf("http://%s/model/getConfigs", a.KubecostHost))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var config KubecostConfig
	err = json.Unmarshal(bodyBytes, &config)
	if err != nil {
		return "", err
	}

	if config.Data.CurrencyCode == "" {
		return "USD", nil
	}

	return config.Data.CurrencyCode, nil
}

func newApp() *App {
	a := App{
		filesToUpload:    make(map[string]struct{}),
		client:           &http.Client{Timeout: time.Minute},
		invoiceYearMonth: time.Now().Local().AddDate(0, 0, -1).Format("2006-01"),
	}
	if err := env.Parse(&a.Config); err != nil {
		log.Fatal(err)
	}

	log.Printf("APP >>>> %+v\n\n", a)
	log.Printf("Config >>>> %+v\n\n", a.Config)

	a.Aggregation = strings.ToLower(a.Aggregation)
	a.resourceType = a.Aggregation

	switch a.Aggregation {
	case "namespace":
		a.Aggregation = "cluster," + a.Aggregation
	case "controller", "pod":
		a.Aggregation = "cluster,namespace," + a.Aggregation
	default:
		log.Fatal("Aggregation type is wrong")
	}

	return &a
}

func checkForError(response *http.Response) {
	if response.StatusCode < 200 || response.StatusCode > 299 {
		log.Fatalf("Request failed with status code: %d", response.StatusCode)
	}
}

// dateIter is a generator function that yields a sequence of dates starting
// from start_year and start_month (formatted as strings) until today.
func dateIter(startDate time.Time) <-chan time.Time {
	c := make(chan time.Time)

	go func() {
		defer close(c)
		for !time.Now().Before(startDate) {
			c <- startDate
			startDate = startDate.AddDate(0, 0, 1)
		}
	}()

	return c
}
