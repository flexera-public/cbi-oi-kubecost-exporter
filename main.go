package main

import (
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
		Cluster         string            `json:"cluster"`
		Container       string            `json:"container"`
		Namespace       string            `json:"namespace"`
		Pod             string            `json:"pod"`
		Node            string            `json:"node"`
		Controller      string            `json:"controller"`
		ControllerKind  string            `json:"controllerKind"`
		Services        []string          `json:"services"`
		ProviderID      string            `json:"providerID"`
		Labels          map[string]string `json:"labels"`
		NamespaceLabels map[string]string `json:"namespaceLabels"`
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

	OptimaFileUploadResponse struct {
		ID           string `json:"id"`
		Status       string `json:"status"`
		BillUploadID string `json:"billUploadId"`
		MD5          string `json:"md5"`
	}

	Config struct {
		RefreshToken         string  `env:"REFRESH_TOKEN"`
		ServiceClientId      string  `env:"SERVICE_APP_CLIENT_ID"`
		ServiceClientSecret  string  `env:"SERVICE_APP_CLIENT_SECRET"`
		OrgID                string  `env:"ORG_ID"`
		BillConnectID        string  `env:"BILL_CONNECT_ID"`
		Shard                string  `env:"SHARD" envDefault:"NAM"`
		KubecostHost         string  `env:"KUBECOST_HOST" envDefault:"localhost:9090"`
		KubecostAPIPath      string  `env:"KUBECOST_API_PATH" envDefault:"/model/"`
		Aggregation          string  `env:"AGGREGATION" envDefault:"pod"`
		ShareNamespaces      string  `env:"SHARE_NAMESPACES" envDefault:"kube-system,cadvisor"`
		Idle                 bool    `env:"IDLE" envDefault:"true"`
		IdleByNode           bool    `env:"IDLE_BY_NODE" envDefault:"false"`
		ShareIdle            bool    `env:"SHARE_IDLE" envDefault:"false"`
		ShareTenancyCosts    bool    `env:"SHARE_TENANCY_COSTS" envDefault:"true"`
		Multiplier           float64 `env:"MULTIPLIER" envDefault:"1.0"`
		FileRotation         bool    `env:"FILE_ROTATION" envDefault:"true"`
		FilePath             string  `env:"FILE_PATH" envDefault:"/var/kubecost"`
		IncludePreviousMonth bool    `env:"INCLUDE_PREVIOUS_MONTH" envDefault:"false"`
		RequestTimeout       int     `env:"REQUEST_TIMEOUT" envDefault:"5"`
		MaxFileRows          int     `env:"MAX_FILE_ROWS" envDefault:"1000000"`
	}

	App struct {
		Config
		aggregation                        string
		filesToUpload                      map[string]map[string]struct{}
		client                             *http.Client
		lastInvoiceDate                    time.Time
		invoiceMonths                      []string
		mandatoryFileSavingPeriodStartDate time.Time
		billUploadURL                      string
	}
)

var uuidPattern = regexp.MustCompile(`an existing billUpload \(ID: ([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
var fileNameRe = regexp.MustCompile(`kubecost-(\d{4}-\d{2}-\d{2})(?:-(\d+))?\.csv(\.gz)?`)

func main() {
	exporter := newApp()
	exporter.updateFileList()
	exporter.updateFromKubecost()
	exporter.uploadToFlexera()
}

func (a *App) updateFromKubecost() {
	now := time.Now().Local()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	err := os.MkdirAll(a.FilePath, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	currency, err := a.getCurrency()
	if err != nil {
		log.Fatal(err)
	}

	for d := range dateIter(now.AddDate(0, -(len(a.invoiceMonths)), 0)) {
		if d.After(now) || !a.dateInInvoiceRange(d) {
			continue
		}

		tomorrow := d.AddDate(0, 0, 1)

		// https://github.com/kubecost/docs/blob/master/allocation.md#querying
		reqUrl := fmt.Sprintf("http://%s%sallocation", a.KubecostHost, a.KubecostAPIPath)
		req, err := http.NewRequest("GET", reqUrl, nil)
		log.Printf("Request: %+v\n", reqUrl)
		if err != nil {
			log.Fatal(err)
		}

		q := req.URL.Query()
		q.Add("window", fmt.Sprintf("%s,%s", d.Format("2006-01-02T15:04:05Z"), tomorrow.Format("2006-01-02T15:04:05Z")))
		q.Add("aggregate", a.aggregation)
		q.Add("idle", fmt.Sprintf("%t", a.Idle))
		q.Add("idleByNode", fmt.Sprintf("%t", a.IdleByNode))
		q.Add("shareIdle", fmt.Sprintf("%t", a.ShareIdle))
		q.Add("shareNamespaces", a.ShareNamespaces)
		q.Add("shareSplit", "weighted")
		q.Add("shareTenancyCosts", fmt.Sprintf("%t", a.ShareTenancyCosts))
		q.Add("step", "1d")
		req.URL.RawQuery = q.Encode()

		resp, err := a.client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Response Status Code: %+v\n", resp.StatusCode)

		var j KubecostAllocationResponse
		err = json.NewDecoder(resp.Body).Decode(&j)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()

		data := j.Data
		monthOfData := d.Format("2006-01")
		var csvFile = fmt.Sprintf(path.Join(a.FilePath, "kubecost-%v.csv.gz"), d.Format("2006-01-02"))

		if j.Code != http.StatusOK {
			log.Println("Kubecost API response code different than 200, skipping")
			continue
		}

		// If the data obtained is empty, skip the iteration, because it might overwrite a previously obtained file for the same range time
		dataExist := false
		for _, allocation := range j.Data {
			if len(allocation) > 0 {
				dataExist = true
			}
		}

		if dataExist == false {
			log.Printf(
				"Kubecost doesn't have data for date range %s to %s, skipping\n",
				d.Format("2006-01-02T15:04:05Z"),
				tomorrow.Format("2006-01-02T15:04:05Z"))
			continue
		}

		var fileIndex int = 1
		var rowCount int = 0

		b := new(bytes.Buffer)
		zipWriter := gzip.NewWriter(b)
		writer := csv.NewWriter(zipWriter)

		writer.Write(a.getCSVHeaders())

		// Logs to validate date range requested and date range gotten in the data
		log.Printf("Requested date range, from %s to %s \n", d.Format("2006-01-02T15:04:05Z"), tomorrow.Format("2006-01-02T15:04:05Z"))

		for _, row := range a.getCSVRows(currency, monthOfData, data) {
			if rowCount >= a.MaxFileRows {
				a.closeAndSaveFile(writer, zipWriter, b, monthOfData, csvFile)

				fileIndex++
				csvFile = fmt.Sprintf(path.Join(a.FilePath, "kubecost-%v_%d.csv.gz"), d.Format("2006-01-02"), fileIndex)
				b = new(bytes.Buffer)
				zipWriter = gzip.NewWriter(b)
				writer = csv.NewWriter(zipWriter)

				writer.Write(a.getCSVHeaders())
				rowCount = 1
			}

			writer.Write(row)
			rowCount++
		}
		a.closeAndSaveFile(writer, zipWriter, b, monthOfData, csvFile)
	}
}

func (a *App) closeAndSaveFile(writer *csv.Writer, zipWriter *gzip.Writer, b *bytes.Buffer, monthOfData, csvFile string) {
	writer.Flush()
	zipWriter.Flush()
	zipWriter.Close()

	a.filesToUpload[monthOfData][csvFile] = struct{}{}
	err := os.WriteFile(csvFile, b.Bytes(), 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Retrieved", csvFile)
	b.Reset()

	// Delete the .csv file if it exists
	csvFilename := strings.TrimSuffix(csvFile, ".gz")
	if _, err := os.Stat(csvFilename); err == nil {
		if err := os.Remove(csvFilename); err != nil {
			log.Printf("error removing file %s: %v", csvFilename, err)
		}
	}
	delete(a.filesToUpload[monthOfData], csvFilename)
}

func (a *App) uploadToFlexera() {
	accessToken, err := a.generateAccessToken()
	if err != nil {
		log.Fatalf("Error generating access token: %v", err)
	}

	authHeaders := map[string]string{"Authorization": "Bearer " + accessToken}

	for month, files := range a.filesToUpload {

		if len(files) == 0 {
			log.Println("No files to upload for month", month)
			continue
		}

		// if we try to upload files for previous month, we need to check if we have files for all days in the month
		if !a.isCurrentMonth(month) && a.DaysInMonth(month) != len(files) {
			log.Println("Skipping month", month, "because not all days have a file to upload")
			continue
		}

		billUploadID, err := a.StartBillUploadProcess(month, authHeaders)
		if err != nil {
			log.Println(err)
			continue
		}

		for fileName := range files {
			err = a.UploadFile(billUploadID, fileName, authHeaders)
			if err != nil {
				log.Printf("Error uploading file: %s. %s\n", fileName, err.Error())
				break
			}
		}

		if err != nil {
			err = a.AbortBillUploadProcess(billUploadID, authHeaders)
		} else {
			err = a.CommitBillUploadProcess(billUploadID, authHeaders)
		}
		if err != nil {
			log.Println(err)
		}
	}
}

func (a *App) StartBillUploadProcess(month string, authHeaders map[string]string) (billUploadID string, err error) {
	billUpload := map[string]string{"billConnectId": a.BillConnectID, "billingPeriod": month}

	billUploadJSON, _ := json.Marshal(billUpload)
	response, err := a.doPost(a.billUploadURL, string(billUploadJSON), authHeaders)
	if err != nil {
		return "", err
	}

	switch response.StatusCode {
	case 429:
		time.Sleep(120 * time.Second)
		return a.StartBillUploadProcess(month, authHeaders)
	case 409:
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}
		uuidMatch := uuidPattern.FindStringSubmatch(string(bodyBytes))
		if len(uuidMatch) < 2 {
			return "", fmt.Errorf("billUpload ID not found")
		}
		inProgressBillUploadID := uuidMatch[1]
		err = a.AbortBillUploadProcess(inProgressBillUploadID, authHeaders)
		if err != nil {
			return "", err
		}
		return a.StartBillUploadProcess(month, authHeaders)
	}

	err = checkForError(response)
	if err != nil {
		return "", err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var jsonResponse map[string]interface{}
	if err = json.Unmarshal(bodyBytes, &jsonResponse); err != nil {
		return "", err
	}

	return jsonResponse["id"].(string), nil
}

func (a *App) CommitBillUploadProcess(billUploadID string, headers map[string]string) error {
	url := fmt.Sprintf("%s/%s/operations", a.billUploadURL, billUploadID)
	response, err := a.doPost(url, `{"operation":"commit"}`, headers)
	if err != nil {
		return err
	}
	log.Println("commit upload bill process with id", billUploadID)

	return checkForError(response)
}

func (a *App) AbortBillUploadProcess(billUploadID string, headers map[string]string) error {
	url := fmt.Sprintf("%s/%s/operations", a.billUploadURL, billUploadID)
	response, err := a.doPost(url, `{"operation":"abort"}`, headers)
	if err != nil {
		return err
	}
	log.Println("aborting upload bill process with id", billUploadID)

	return checkForError(response)
}

func (a *App) UploadFile(billUploadID, fileName string, authHeaders map[string]string) error {
	baseName := filepath.Base(fileName)
	uploadFileURL := fmt.Sprintf("%s/%s/files/%s", a.billUploadURL, billUploadID, baseName)

	fileData, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	response, err := a.doPost(uploadFileURL, string(fileData), authHeaders)
	if err != nil {
		return err
	}

	err = checkForError(response)
	if err != nil {
		return err
	}

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("error reading response: %s", err.Error())
	}

	defer response.Body.Close()
	var uploadResponse OptimaFileUploadResponse
	if err = json.Unmarshal(bodyBytes, &uploadResponse); err != nil {
		return fmt.Errorf("error parsing response: %s", err.Error())
	}

	md5Hash := getMD5FromFileBytes(fileData)
	if md5Hash != uploadResponse.MD5 {
		return fmt.Errorf("MD5 of file %s does not match MD5 of uploaded file", fileName)
	}

	log.Printf("File %s uploaded and MD5 of file matches MD5 of uploaded file\n", fileName)
	return nil
}

func (a *App) doPost(url, data string, headers map[string]string) (*http.Response, error) {
	request, _ := http.NewRequest("POST", url, strings.NewReader(data))
	log.Printf("Request: %+v\n", url)

	for key, value := range headers {
		request.Header.Set(key, value)
	}

	response, err := a.client.Do(request)
	if err != nil {
		return nil, err
	}

	log.Printf("Response Status Code: %+v\n", response.StatusCode)
	return response, nil
}

// generateAccessToken returns an access token from the Flexera One API using a given refreshToken or service account.
func (a *App) generateAccessToken() (string, error) {
	domainsDict := map[string]string{
		"NAM": "flexera.com",
		"EU":  "flexera.eu",
		"AU":  "flexera.au",
	}
	accessTokenUrl := fmt.Sprintf("https://login.%s/oidc/token", domainsDict[a.Shard])
	reqBody := url.Values{}
	if len(a.RefreshToken) > 0 {
		reqBody.Set("grant_type", "refresh_token")
		reqBody.Set("refresh_token", a.RefreshToken)
	} else {
		reqBody.Set("grant_type", "client_credentials")
		reqBody.Set("client_id", a.ServiceClientId)
		reqBody.Set("client_secret", a.ServiceClientSecret)
	}

	req, err := http.NewRequest("POST", accessTokenUrl, strings.NewReader(reqBody.Encode()))
	if err != nil {
		return "", fmt.Errorf("error creating access token request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error retrieving access token: %v", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error retrieving access token: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading access token response body: %v", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("error parsing access token response body: %v", err)
	}

	return tokenResp.AccessToken, nil
}

// update file list and remove old files
func (a *App) updateFileList() {
	files, err := os.ReadDir(a.FilePath)
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		if file.Type().IsRegular() {
			matches := fileNameRe.FindStringSubmatch(file.Name())
			if matches != nil {
				if t, err := time.Parse("2006-01-02", matches[1]); err == nil {
					if a.dateInInvoiceRange(t) {
						a.filesToUpload[t.Format("2006-01")][path.Join(a.FilePath, file.Name())] = struct{}{}
					} else if a.FileRotation && !a.dateInMandatoryFileSavingPeriod(t) {
						if err = os.Remove(path.Join(a.FilePath, file.Name())); err != nil {
							log.Printf("error removing file %s: %v", file.Name(), err)
						}
					}
				}
			}
		}
	}
}

func (a *App) getCurrency() (string, error) {
	reqUrl := fmt.Sprintf("http://%s%sgetConfigs", a.KubecostHost, a.KubecostAPIPath)
	resp, err := a.client.Get(reqUrl)
	log.Printf("Request: %+v\n", reqUrl)
	if err != nil {
		return "", err
	}
	log.Printf("Response Status Code: %+v\n", resp.StatusCode)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status code: %d", resp.StatusCode)
	}

	var config KubecostConfig
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		return "", err
	}

	if config.Data.CurrencyCode == "" {
		return "USD", nil
	}

	return config.Data.CurrencyCode, nil
}

func (a *App) dateInInvoiceRange(date time.Time) bool {
	for _, month := range a.invoiceMonths {
		if date.Format("2006-01") == month {
			return true
		}
	}
	return false
}

func (a *App) dateInMandatoryFileSavingPeriod(date time.Time) bool {
	return !date.Before(a.mandatoryFileSavingPeriodStartDate)
}

func (a *App) isCurrentMonth(month string) bool {
	return time.Now().Local().Format("2006-01") == month
}

func (a *App) DaysInMonth(month string) int {
	date, err := time.Parse("2006-01", month)
	if err != nil {
		return 0
	}
	numDays := date.AddDate(0, 1, 0).Sub(date).Hours() / 24
	return int(numDays)
}

func newApp() *App {
	shardDict := map[string]string{
		"NAM": "api.optima.flexeraeng.com",
		"EU":  "api.optima-eu.flexeraeng.com",
		"AU":  "api.optima-apac.flexeraeng.com",
	}

	lastInvoiceDate := time.Now().Local().AddDate(0, 0, -1)
	a := App{
		filesToUpload:   make(map[string]map[string]struct{}),
		client:          &http.Client{},
		lastInvoiceDate: lastInvoiceDate,
	}
	if err := env.Parse(&a.Config); err != nil {
		log.Fatal(err)
	}

	a.client.Timeout = time.Duration(a.RequestTimeout) * time.Minute
	a.billUploadURL = fmt.Sprintf("https://%s/optima/orgs/%s/billUploads", shardDict[a.Shard], a.OrgID)

	a.invoiceMonths = []string{lastInvoiceDate.Format("2006-01")}
	if a.IncludePreviousMonth {
		a.invoiceMonths = append(a.invoiceMonths, a.lastInvoiceDate.AddDate(0, -1, 0).Format("2006-01"))
	}
	// The mandatory file saving period is the period since the first day of the previous month of last invoice date
	previousMonthOfLastInvoiceDate := lastInvoiceDate.AddDate(0, -1, 0)
	a.mandatoryFileSavingPeriodStartDate = time.Date(previousMonthOfLastInvoiceDate.Year(), previousMonthOfLastInvoiceDate.Month(), 1, 0, 0, 0, 0, previousMonthOfLastInvoiceDate.Location())

	for _, month := range a.invoiceMonths {
		a.filesToUpload[month] = make(map[string]struct{})
	}

	a.Aggregation = strings.ToLower(a.Aggregation)

	switch a.Aggregation {
	case "namespace":
		a.aggregation = "cluster," + a.Aggregation
	case "controller", "pod":
		a.aggregation = "cluster,namespace," + a.Aggregation
	default:
		log.Fatal("Aggregation type is wrong")
	}

	switch a.Shard {
	case "NAM", "EU", "AU":
	default:
		log.Fatal("Shard is wrong")
	}

	return &a
}

func (a *App) getCSVHeaders() []string {
	return []string{
		"ResourceID",
		"Cost",
		"CurrencyCode",
		"Aggregation",
		"UsageType",
		"UsageAmount",
		"UsageUnit",
		"Cluster",
		"Container",
		"Namespace",
		"Pod",
		"Node",
		"Controller",
		"ControllerKind",
		"ProviderID",
		"Labels",
		"InvoiceYearMonth",
		"InvoiceDate",
		"StartTime",
		"EndTime",
	}
}

func (a *App) getCSVRows(currency string, month string, data []map[string]KubecostAllocation) [][]string {
	rows := make([][]string, 0)
	mapDatesGotten := make(map[string]string)
	for _, allocation := range data {
		for id, v := range allocation {
			mapDatesGotten[v.Start] = v.End
			labels := extractLabels(v.Properties)
			types := []string{"cpuCost", "gpuCost", "ramCost", "pvCost", "networkCost", "sharedCost", "externalCost", "loadBalancerCost"}
			vals := []float64{
				v.CPUCost + v.CPUCostAdjustment,
				v.GPUCost + v.GPUCostAdjustment,
				v.RAMCost + v.RAMCostAdjustment,
				v.PVCost + v.PVCostAdjustment,
				v.NetworkCost + v.NetworkCostAdjustment,
				v.SharedCost,
				v.ExternalCost,
				v.LoadBalancerCost + v.LoadBalancerCostAdjustment,
			}
			units := []string{"cpuCoreHours", "gpuHours", "ramByteHours", "pvByteHours", "networkTransferBytes", "minutes", "minutes", "minutes"}
			amounts := []float64{v.CPUCoreHours, v.GPUHours, v.RAMByteHours, v.PVByteHours, v.NetworkTransferBytes, v.Minutes, v.Minutes, v.Minutes}

			for i, c := range types {
				multiplierFloat := a.Multiplier * vals[i]
				if v.Properties.Cluster == "" {
					v.Properties.Cluster = "Cluster"
				}

				rows = append(rows, []string{
					id,
					strconv.FormatFloat(multiplierFloat, 'f', 2, 64),
					currency,
					a.Aggregation,
					c,
					strconv.FormatFloat(amounts[i], 'f', 2, 64),
					units[i],
					v.Properties.Cluster,
					v.Properties.Container,
					v.Properties.Namespace,
					v.Properties.Pod,
					v.Properties.Node,
					v.Properties.Controller,
					v.Properties.ControllerKind,
					v.Properties.ProviderID,
					labels,
					strings.ReplaceAll(month, "-", ""),
					v.Window.Start,
					v.Start,
					v.End,
				})
			}
		}
	}
	log.Printf("Gotten dates range: %v \n", mapDatesGotten)
	return rows
}

func checkForError(response *http.Response) error {
	if response.StatusCode < 200 || response.StatusCode > 299 {
		if bodyBytes, err := io.ReadAll(response.Body); err == nil {
			log.Println(string(bodyBytes))
		}
		log.Printf("Request failed with status code: %d \n", response.StatusCode)
		return fmt.Errorf("request failed with status code: %d", response.StatusCode)
	}
	return nil
}

// dateIter is a generator function that yields a sequence of dates starting
// from start_year and start_month (formatted as strings) until today.
func dateIter(startDate time.Time) <-chan time.Time {
	c := make(chan time.Time)

	go func() {
		defer close(c)
		for !time.Now().Local().Before(startDate) {
			c <- startDate
			startDate = startDate.AddDate(0, 0, 1)
		}
	}()

	return c
}

// extractLabels returns a JSON string with all the properties labels, merging labels and namespace labels
// and adding labels for the container, controller, pod and provider.
func extractLabels(properties Properties) string {
	mapLabels := make(map[string]string)
	if properties.Labels != nil {
		mapLabels = properties.Labels
	}
	if properties.NamespaceLabels != nil {
		for k, v := range properties.NamespaceLabels {
			mapLabels[k] = v
		}
	}
	if properties.Container != "" {
		mapLabels["kc-container"] = properties.Container
	}
	if properties.Controller != "" {
		mapLabels["kc-controller"] = properties.Controller
	}
	if properties.Node != "" {
		mapLabels["kc-node"] = properties.Node
	}
	if properties.Pod != "" {
		mapLabels["kc-pod-id"] = properties.Pod
	}
	if properties.ProviderID != "" {
		mapLabels["kc-provider-id"] = properties.ProviderID
	}

	labelsJSON, _ := json.Marshal(mapLabels)
	return string(labelsJSON)
}

func getMD5FromFileBytes(fileBytes []byte) string {
	hash := md5.New()
	hash.Write(fileBytes)

	return hex.EncodeToString(hash.Sum(nil))
}
