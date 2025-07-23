package main

import (
	"crypto/md5"
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
	"syscall"
	"time"

	"github.com/caarlos0/env/v11"
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
		RefreshToken                string  `env:"REFRESH_TOKEN"`
		ServiceClientID             string  `env:"SERVICE_APP_CLIENT_ID"`
		ServiceClientSecret         string  `env:"SERVICE_APP_CLIENT_SECRET"`
		OrgID                       string  `env:"ORG_ID"`
		BillConnectID               string  `env:"BILL_CONNECT_ID"`
		Shard                       string  `env:"SHARD" envDefault:"NAM"`
		KubecostHost                string  `env:"KUBECOST_HOST" envDefault:"localhost:9090"`
		KubecostAPIPath             string  `env:"KUBECOST_API_PATH" envDefault:"/model/"`
		KubecostConfigHost          string  `env:"KUBECOST_CONFIG_HOST"`
		KubecostConfigAPIPath       string  `env:"KUBECOST_CONFIG_API_PATH"`
		Aggregation                 string  `env:"AGGREGATION" envDefault:"pod"`
		ShareNamespaces             string  `env:"SHARE_NAMESPACES" envDefault:"kube-system,cadvisor"`
		Idle                        bool    `env:"IDLE" envDefault:"true"`
		IdleByNode                  bool    `env:"IDLE_BY_NODE" envDefault:"false"`
		ShareIdle                   bool    `env:"SHARE_IDLE" envDefault:"false"`
		ShareTenancyCosts           bool    `env:"SHARE_TENANCY_COSTS" envDefault:"true"`
		Multiplier                  float64 `env:"MULTIPLIER" envDefault:"1.0"`
		FileRotation                bool    `env:"FILE_ROTATION" envDefault:"true"`
		FilePath                    string  `env:"FILE_PATH" envDefault:"/var/kubecost"`
		IncludePreviousMonth        bool    `env:"INCLUDE_PREVIOUS_MONTH" envDefault:"true"`
		RequestTimeout              int     `env:"REQUEST_TIMEOUT" envDefault:"5"`
		MaxFileRows                 int     `env:"MAX_FILE_ROWS" envDefault:"1000000"`
		CreateBillConnectIfNotExist bool    `env:"CREATE_BILL_CONNECT_IF_NOT_EXIST" envDefault:"false"`
		VendorName                  string  `env:"VENDOR_NAME" envDefault:"Kubecost"`
		PageSize                    int     `env:"PAGE_SIZE" envDefault:"500"`
		DefaultCurrency             string  `env:"DEFAULT_CURRENCY" envDefault:"USD"`
		OverridePodLabels           bool    `env:"OVERRIDE_POD_LABELS" envDefault:"true"`
	}

	App struct {
		Config
		lockFile                           *os.File
		aggregation                        string
		filesToUpload                      map[string]map[string]struct{}
		client                             *http.Client
		lastInvoiceDate                    time.Time
		invoiceMonths                      []string
		mandatoryFileSavingPeriodStartDate time.Time
		billUploadURL                      string
	}
)

const lockFileName = ".kubecost-exporter.lock"

var uuidPattern = regexp.MustCompile(`an existing billUpload \(ID: ([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
var fileNameRe = regexp.MustCompile(`kubecost-(\d{4}-\d{2}-\d{2})(?:-(\d+))?\.csv(\.gz)?`)

func main() {
	exporter := newApp()
	exporter.lockState()
	defer exporter.unlockState()
	exporter.cleanupTempFiles()
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

	currency := a.getCurrency()

	for d := range dateIter(now.AddDate(0, -(len(a.invoiceMonths)), 0)) {
		if d.After(now) || !a.dateInInvoiceRange(d) {
			continue
		}

		err := a.processDateWithStreaming(d, currency)
		if err != nil {
			log.Printf("Error processing date %s: %v", d.Format("2006-01-02"), err)
			continue
		}
	}
}
func (a *App) processDateWithStreaming(d time.Time, currency string) error {
	tomorrow := d.AddDate(0, 0, 1)
	currentDate := d.Format("2006-01-02")
	monthOfData := d.Format("2006-01")

	fileWriter, err := newFileWriter(a, fmt.Sprintf(path.Join(a.FilePath, "kubecost-%v.csv.gz"), currentDate))
	if err != nil {
		return fmt.Errorf("failed to create file writer: %v", err)
	}
	defer func() {
		if err := fileWriter.close(); err != nil {
			log.Printf("Warning: failed to close file writer: %v", err)
		}
	}()

	err = fileWriter.writeHeaders(a.getCSVHeaders())
	if err != nil {
		return fmt.Errorf("failed to write headers: %v", err)
	}

	page := 0
	limit := a.PageSize
	requestNewPage := true
	totalRecordsProcessed := 0
	totalRowsProcessed := 0
	idleRecords := make(map[string]KubecostAllocation)

	log.Printf("Starting streaming processing for date %s", currentDate)

	// https://github.com/kubecost/docs/blob/master/allocation.md#querying
	baseURL := fmt.Sprintf("http://%s", a.KubecostHost)
	reqURL, err := url.JoinPath(baseURL, a.KubecostAPIPath, "allocation")
	if err != nil {
		return fmt.Errorf("failed to build allocation URL: %w", err)
	}

	for requestNewPage {
		req, err := http.NewRequest("GET", reqURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %v", err)
		}

		q := req.URL.Query()
		q.Add("window", fmt.Sprintf("%s,%s", d.Format("2006-01-02T15:04:05Z"), tomorrow.Format("2006-01-02T15:04:05Z")))
		q.Add("aggregate", a.aggregation)
		q.Add("idle", fmt.Sprintf("%t", a.Idle))
		q.Add("includeIdle", fmt.Sprintf("%t", a.Idle))
		q.Add("idleByNode", fmt.Sprintf("%t", a.IdleByNode))
		q.Add("shareIdle", fmt.Sprintf("%t", a.ShareIdle))
		q.Add("shareNamespaces", a.ShareNamespaces)
		q.Add("shareSplit", "weighted")
		q.Add("shareTenancyCosts", fmt.Sprintf("%t", a.ShareTenancyCosts))
		q.Add("step", "1d")
		q.Add("accumulate", "true")
		q.Add("offset", fmt.Sprintf("%d", page*limit))
		q.Add("limit", fmt.Sprintf("%d", limit))

		req.URL.RawQuery = q.Encode()
		log.Printf("Request: %s?%s", reqURL, q.Encode())

		resp, err := a.client.Do(req)
		if err != nil {
			return fmt.Errorf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		var j KubecostAllocationResponse
		if err = json.NewDecoder(resp.Body).Decode(&j); err != nil {
			return fmt.Errorf("failed to decode response: %v", err)
		}

		if j.Code != http.StatusOK {
			log.Printf("Kubecost API response code %d, skipping page %d", j.Code, page)
			break
		}

		pageRecordsProcessed := 0

		for k, allocation := range j.Data {
			if page == 0 && k == 0 && len(allocation) > 0 {
				log.Printf("Kubecost returned data for %s, cleaning up old indexed files", currentDate)
				a.cleanupOldFiles(monthOfData, currentDate)
			}

			for id, record := range allocation {
				if a.isIdleRecord(record) {
					idleRecords[id] = record
				} else {
					rows := a.getCSVRowsFromRecord(currency, monthOfData, record)
					for _, row := range rows {
						err := fileWriter.writeRow(row, monthOfData, a.filesToUpload)
						if err != nil {
							return err
						}
						totalRowsProcessed++
					}
					pageRecordsProcessed++
				}
			}
			totalRecords := len(allocation)
			if a.ShareIdle && totalRecords < a.PageSize || !a.ShareIdle && totalRecords < a.PageSize+1 {
				requestNewPage = false
			}
		}

		totalRecordsProcessed += pageRecordsProcessed
		log.Printf("Processed page %d: %d records, total: %d", page, pageRecordsProcessed, totalRecordsProcessed)
		page++
	}

	for _, record := range idleRecords {
		rows := a.getCSVRowsFromRecord(currency, monthOfData, record)
		for _, row := range rows {
			err := fileWriter.writeRow(row, monthOfData, a.filesToUpload)
			if err != nil {
				return err
			}
			totalRowsProcessed++
		}
	}
	totalRecordsProcessed += len(idleRecords)
	log.Printf("Processed %d idle records", len(idleRecords))

	// If the data obtained is empty, skip the iteration, because it might overwrite a previously obtained file for the same range time
	if totalRecordsProcessed == 0 {
		log.Printf("No data for date %s, removing empty file", currentDate)
		return nil
	}

	err = fileWriter.finalizeFile(monthOfData, a.filesToUpload)
	if err != nil {
		return fmt.Errorf("failed to finalize file: %v", err)
	}

	if err := validateGzipHeaders(fileWriter.filePath); err != nil {
		log.Printf("Warning: file failed validation: %v", err)
	}

	log.Printf("Completed processing for %s: %d total records, %d data rows", currentDate, totalRecordsProcessed, totalRowsProcessed)
	return nil
}

func (a *App) cleanupOldFiles(monthOfData, currentDate string) {
	filesToRemove := make([]string, 0)

	// Find all indexed files for this date (kubecost-date-2.csv.gz, kubecost-date-3.csv.gz, etc.)
	for filename := range a.filesToUpload[monthOfData] {
		if strings.Contains(filename, currentDate) && !strings.HasSuffix(filename, ".tmp") {
			filesToRemove = append(filesToRemove, filename)
		}
	}

	if len(filesToRemove) > 0 {
		log.Printf("Kubecost has data for %s, removing %d old indexed files", currentDate, len(filesToRemove))

		for _, filename := range filesToRemove {
			err := os.Remove(filename)
			if err != nil && !os.IsNotExist(err) {
				log.Printf("Warning: failed to remove old indexed file %s: %v", filename, err)
			}

			delete(a.filesToUpload[monthOfData], filename)
		}
	}
}

func (a *App) isIdleRecord(record KubecostAllocation) bool {
	return strings.Contains(record.Name, "_idle_")
}

func (a *App) uploadToFlexera() {
	accessToken, err := a.generateAccessToken()
	if err != nil {
		log.Fatalf("Error generating access token: %v", err)
	}

	authHeaders := map[string]string{"Authorization": "Bearer " + accessToken}

	atLeastOneError := false

	for month, files := range a.filesToUpload {

		if len(files) == 0 {
			log.Println("No files to upload for month", month)
			continue
		}

		// if we try to upload files for previous month, we need to check if we have files for all days in the month
		if !a.isCurrentMonth(month) {
			// Since there may be more than one file for the same day, we must ensure that there is at least one file for each day.
			daysToUpload := map[string]struct{}{}
			for filename := range files {
				matches := fileNameRe.FindStringSubmatch(filename)
				if len(matches) >= 2 {
					daysToUpload[matches[1]] = struct{}{}
				}
			}

			if a.DaysInMonth(month) > len(daysToUpload) {
				log.Println("Skipping month", month, "because not all days have a file to upload")
				continue
			}
		}

		billUploadID, err := a.StartBillUploadProcess(month, authHeaders)
		if err != nil {
			log.Println(err)
			atLeastOneError = true
			continue
		}

		for fileName := range files {
			err = a.UploadFile(billUploadID, fileName, authHeaders)
			if err != nil {
				log.Printf("Error uploading file: %s. %s\n", fileName, err.Error())
				atLeastOneError = true
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
			atLeastOneError = true
		}
	}

	//If at least one error in bill processing exit with code 1
	if atLeastOneError {
		//the below method internally uses os.Exit(1)
		log.Fatal("Error during bill upload. Internal server error")
	}
}

func (a *App) StartBillUploadProcess(month string, authHeaders map[string]string) (billUploadID string, err error) {
	//Before the upload process create bill connect
	a.createBillConnectIfNotExist(authHeaders)
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

func (a *App) createBillConnectIfNotExist(authHeaders map[string]string) {

	//If the flag is not enabled, do not attempt to create the bill connect
	if !a.CreateBillConnectIfNotExist {
		return
	}

	integrationID := "cbi-oi-kubecost"
	//Split the billConnectId using the integrationId based on the bill identifier
	if !strings.HasPrefix(a.BillConnectID, integrationID) {
		log.Fatal("billConnectId does not start with the required prefix")
	}
	billIdentifier := strings.TrimPrefix(a.BillConnectID, integrationID+"-")
	//Vendor name is same as display name
	params := map[string]string{"displayName": a.VendorName, "vendorName": a.VendorName}

	//name field has same value as bill identifier
	createBillConnectPayload := map[string]interface{}{"billIdentifier": billIdentifier, "integrationId": integrationID, "name": billIdentifier, "params": params}
	url := fmt.Sprintf("https://api.%s/%s/%s/%s", a.getFlexeraDomain(), "finops-onboarding/v1/orgs", a.OrgID, "bill-connects/cbi")

	billConnectJSON, _ := json.Marshal(createBillConnectPayload)
	response, err := a.doPost(url, string(billConnectJSON), authHeaders)
	if err != nil {
		//When the bill connect id is not provided, abort the process
		log.Fatalf("Error while creating the bill connect : %v", err)
	}

	switch response.StatusCode {
	case 201:
		log.Printf("Bill Connect Id is created %s", a.BillConnectID)
	case 409:
		log.Printf("Bill Connect Id already exists %s", a.BillConnectID)
	default:
		log.Fatalf("Error while creating the bill connect : %v", err)
	}
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
	accessTokenURL := fmt.Sprintf("https://login.%s/oidc/token", a.getFlexeraDomain())
	reqBody := url.Values{}
	if len(a.RefreshToken) > 0 {
		reqBody.Set("grant_type", "refresh_token")
		reqBody.Set("refresh_token", a.RefreshToken)
	} else {
		reqBody.Set("grant_type", "client_credentials")
		reqBody.Set("client_id", a.ServiceClientID)
		reqBody.Set("client_secret", a.ServiceClientSecret)
	}

	req, err := http.NewRequest("POST", accessTokenURL, strings.NewReader(reqBody.Encode()))
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

func (a *App) getCurrency() string {
	baseURL := fmt.Sprintf("http://%s", a.KubecostConfigHost)
	reqURL, err := url.JoinPath(baseURL, a.KubecostConfigAPIPath, "getConfigs")
	if err != nil {
		log.Printf("Failed to build config URL, taking default value '%s'. Error: %v", a.DefaultCurrency, err)
		return a.DefaultCurrency
	}

	resp, err := a.client.Get(reqURL)
	log.Printf("Request: %+v\n", reqURL)
	if err != nil {
		log.Printf("Something went wrong, taking default value '%s'. \n Error: %s.\n", a.DefaultCurrency, err.Error())
		return a.DefaultCurrency
	}
	log.Printf("Response Status Code: %+v\n", resp.StatusCode)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected http status at get config request, taking default value '%s'.\n", a.DefaultCurrency)
		return a.DefaultCurrency
	}

	var config KubecostConfig
	err = json.NewDecoder(resp.Body).Decode(&config)
	if err != nil {
		log.Printf("Something went wrong during decoding, taking default value '%s'. \n Error: %s.\n", a.DefaultCurrency, err.Error())
		return a.DefaultCurrency
	}

	if config.Data.CurrencyCode == "" {
		log.Printf("Currency has no value in the config, taking default value '%s'.\n", a.DefaultCurrency)
		return a.DefaultCurrency
	}

	return config.Data.CurrencyCode
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

func (a *App) getOptimaAPIDomain() string {
	optimaAPIDomainsDict := map[string]string{
		"NAM": "api.optima.flexeraeng.com",
		"EU":  "api.optima-eu.flexeraeng.com",
		"AU":  "api.optima-apac.flexeraeng.com",
		"DEV": "api.optima.flexeraengdev.com",
	}
	return optimaAPIDomainsDict[a.Shard]
}

func (a *App) getFlexeraDomain() string {
	domainsDict := map[string]string{
		"NAM": "flexera.com",
		"EU":  "flexera.eu",
		"AU":  "flexera.au",
		"DEV": "flexeratest.com",
	}
	return domainsDict[a.Shard]
}

func (a *App) validateAppConfiguration() error {
	switch a.Shard {
	case "NAM", "EU", "AU", "DEV":
	default:
		return fmt.Errorf("shard: %s is wrong", a.Shard)
	}

	a.Aggregation = strings.ToLower(a.Aggregation)

	switch a.Aggregation {
	case "namespace":
		a.aggregation = "cluster," + a.Aggregation
	case "controller":
		a.aggregation = "cluster,namespace,controllerKind," + a.Aggregation
	case "node":
		a.aggregation = "cluster,namespace,controllerKind,controller," + a.Aggregation
	case "pod":
		a.aggregation = "cluster,namespace,controllerKind,controller,node," + a.Aggregation
	default:
		return fmt.Errorf("aggregation type: %s is wrong", a.Aggregation)
	}

	if a.KubecostConfigHost == "" {
		a.KubecostConfigHost = a.KubecostHost
	}

	if a.KubecostConfigAPIPath == "" {
		a.KubecostConfigAPIPath = a.KubecostAPIPath
	}

	return nil
}

func newApp() *App {
	lastInvoiceDate := time.Now().Local().AddDate(0, 0, -1)
	a := App{
		filesToUpload:   make(map[string]map[string]struct{}),
		client:          &http.Client{},
		lastInvoiceDate: lastInvoiceDate,
	}
	if err := env.Parse(&a.Config); err != nil {
		log.Fatal(err)
	}

	if err := a.validateAppConfiguration(); err != nil {
		log.Fatal(err)
	}

	a.client.Timeout = time.Duration(a.RequestTimeout) * time.Minute
	a.billUploadURL = fmt.Sprintf("https://%s/optima/orgs/%s/billUploads", a.getOptimaAPIDomain(), a.OrgID)

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

func (a *App) getCSVRowsFromRecord(currency string, month string, v KubecostAllocation) [][]string {
	rows := make([][]string, 0, 8) // Pre-allocate for 8 cost types

	labels := extractLabels(v.Properties, a.OverridePodLabels)
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

	if v.Properties.Cluster == "" {
		v.Properties.Cluster = "Cluster"
	}

	for i, c := range types {
		multiplierFloat := a.Multiplier * vals[i]

		rows = append(rows, []string{
			v.Name,
			strconv.FormatFloat(multiplierFloat, 'f', 5, 64),
			currency,
			a.Aggregation,
			c,
			strconv.FormatFloat(amounts[i], 'f', 5, 64),
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

	return rows
}

func (a *App) lockState() {
	lockPath := filepath.Join(a.FilePath, lockFileName)

	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalf("Failed to create lock file: %v", err)
	}

	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err := lockFile.Close(); err != nil {
			log.Printf("Warning: failed to close lockFile: %v", err)
		}

		log.Fatalf("Another instance is already running (failed to acquire lock): %v", err)
	}

	_, err = fmt.Fprintf(lockFile, "%d\n", os.Getpid())
	if err != nil {
		if err := lockFile.Close(); err != nil {
			log.Printf("Warning: failed to close lockFile: %v", err)
		}
		log.Fatalf("Failed to write PID to lock file: %v", err)
	}

	log.Printf("Acquired directory lock: %s", lockPath)

	a.lockFile = lockFile
}

func (a *App) unlockState() {
	lockPath := filepath.Join(a.FilePath, lockFileName)
	if a.lockFile != nil {
		if err := syscall.Flock(int(a.lockFile.Fd()), syscall.LOCK_UN); err != nil {
			log.Printf("Warning: failed to unlock file: %v", err)
		}

		if err := a.lockFile.Close(); err != nil {
			log.Printf("Warning: failed to close lock file: %v", err)
		}

		if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
			log.Printf("Warning: failed to remove lock file %s: %v", lockPath, err)
		}

		log.Printf("Released directory lock: %s", lockPath)
		a.lockFile = nil
	}
}

func (a *App) cleanupTempFiles() {
	entries, err := os.ReadDir(a.FilePath)
	if err != nil {
		log.Printf("Warning: failed to read directory %s for cleanup: %v", a.FilePath, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".tmp") && strings.Contains(name, "kubecost-") {
			tempPath := filepath.Join(a.FilePath, name)

			if err := os.Remove(tempPath); err != nil {
				log.Printf("Warning: failed to remove temp file %s: %v", tempPath, err)
			} else {
				log.Printf("Removed temp file: %s", tempPath)
			}
		}
	}
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
func extractLabels(properties Properties, overridePodLabels bool) string {
	mapLabels := make(map[string]string)
	if properties.Labels != nil {
		mapLabels = properties.Labels
	}
	if properties.NamespaceLabels != nil {
		for k, v := range properties.NamespaceLabels {
			//Check if key is present inside the existing labels
			//If the key is present and override is set to true, only then reset the label
			if _, ok := mapLabels[k]; !ok || overridePodLabels {
				mapLabels[k] = v
			}
		}
	}
	if properties.Container != "" {
		mapLabels["kc-container"] = properties.Container
	}
	if properties.Controller != "" {
		mapLabels["kc-controller"] = properties.Controller
	}
	if properties.ControllerKind != "" {
		mapLabels["kc-controller-kind"] = properties.ControllerKind
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

	//Map labels with cluster and namespace.
	if properties.Cluster != "" {
		mapLabels["kc-cluster"] = properties.Cluster
	}

	if properties.Namespace != "" {
		mapLabels["kc-namespace"] = properties.Namespace
	}

	labelsJSON, _ := json.Marshal(mapLabels)
	return string(labelsJSON)
}

func getMD5FromFileBytes(fileBytes []byte) string {
	hash := md5.New()
	hash.Write(fileBytes)

	return hex.EncodeToString(hash.Sum(nil))
}
