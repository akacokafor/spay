package spay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

const (
	sterlingBankCbnCode  = "232"
	successfulStatus     = "Successful"
	successfulStatusCode = "00"
	tellerId             = "sample-teller-e5dc63e264d29b7578e96bf"
	StagingBaseUrl       = "https://sbdevzone.sterling.ng/Spay"
	ProdBaseUrl          = "https://webapps.sterling.ng/spay"
	defaultLocation      = "6.44,3.53"
)

var (
	ErrInvalidArgument = fmt.Errorf("invalid argument provided")
)

type BitString string

func (b BitString) AsByteSlice() ([]byte, error) {
	var out []byte
	var str string
	for i := len(b); i > 0; i -= 8 {
		if i-8 < 0 {
			str = string(b[0:i])
		} else {
			str = string(b[i-8 : i])
		}
		v, err := strconv.ParseUint(str, 2, 8)
		if err != nil {
			return nil, err
		}
		out = append([]byte{byte(v)}, out...)
	}
	return out, nil
}

type Config struct {
	appId        int32
	sharedKey    BitString
	sharedVector BitString
	baseUrl      string
	FromAccount  string
	transferCost float64
}

type Api struct {
	config                Config
	httpClient            *http.Client
	tellerId              string
	shouldDecryptResponse bool
}

func NewApi(
	sharedVector,
	sharedKey BitString,
	appId int32,
	fromAccount string,
	shouldDecryptResponse bool,
	baseUrl string,
) (*Api, error) {

	if baseUrl == "" {
		return nil, fmt.Errorf("base url is required for spay api")
	}

	var config Config
	config.appId = appId
	config.sharedKey = sharedKey
	config.sharedVector = sharedVector
	config.baseUrl = baseUrl
	config.FromAccount = fromAccount
	config.transferCost = 10.0

	httpClient := &http.Client{}
	return &Api{
		config:                config,
		httpClient:            httpClient,
		tellerId:              tellerId,
		shouldDecryptResponse: shouldDecryptResponse,
	}, nil
}

func (a *Api) InitiateInterBankTransfer(transfer *InterBankTransferRequest) (*InterBankTransferResult, error) {
	req := interBankTransferRequest{
		BaseApiReq: BaseApiReq{
			Referenceid:   transfer.Reference,
			RequestType:   160,
			Translocation: transfer.Translocation,
		},
		SessionID:           transfer.NameEnquirySessionID,
		FromAccount:         a.config.FromAccount,
		ToAccount:           transfer.ToAccount,
		Amount:              transfer.Amount,
		DestinationBankCode: transfer.DestinationBankCode,
		NEResponse:          transfer.NEResponse,
		BenefiName:          transfer.BenefiName,
		PaymentReference:    transfer.PaymentReference,
		Tellerid:            transfer.Tellerid,
	}

	if req.Translocation == "" {
		req.Translocation = defaultLocation
	}

	url := "/api/Spay/InterbankTransferReq"
	method := http.MethodPost
	inputBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	base64Encrypted, err := a.encrypt(string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("3des encryption: %w", err)
	}
	result, err := a.request(url, method, []byte(base64Encrypted))
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			return nil, ErrInsufficientFunds
		}
		return nil, fmt.Errorf("interbank transfer request: %w", err)
	}

	if a.shouldDecryptResponse {
		decodedStr, err := a.decrypt(string(result))
		if err != nil {
			return nil, fmt.Errorf("decoded interbank transfer response: %w", err)
		}
		result = []byte(decodedStr)
	}

	var output InterBankTransferResult
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	if output.Response != successfulStatusCode {
		logrus.WithField("transferResult", output).WithField("request", req).Error("interbank transfer completed without success")
		return nil, fmt.Errorf("could not complete transfer: %v", output.Message)
	}

	return &output, nil
}

func (a *Api) ListBanks() (ListOfBankResponse, error) {
	req := ListBanksRequest{
		BaseApiReq: BaseApiReq{
			Referenceid:   fmt.Sprintf("%d", time.Now().UnixMilli()),
			RequestType:   152,
			Translocation: "N/A", //defaultLocation,
		},
	}
	url := "/api/Spay/GetBankListReq"
	method := http.MethodPost
	inputBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	logrus.WithField("req", string(inputBytes)).Info("request body")

	base64Encrypted, err := a.encrypt(string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("3des encryption: %w", err)
	}

	result, err := a.request(url, method, []byte(base64Encrypted))
	if err != nil {
		return nil, fmt.Errorf("interbank transfer request: %w", err)
	}

	if a.shouldDecryptResponse {
		decodedStr, err := a.decrypt(string(result))
		if err != nil {
			return nil, fmt.Errorf("decoded interbank transfer response: %w", err)
		}
		result = []byte(decodedStr)
	}

	var output ApiOperationResponse[ApiOperationResponseData]
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	if output.Data.Status != successfulStatus {
		return nil, fmt.Errorf("could not complete request: %s", output.Data.Response)
	}

	var item ListOfBankResponse
	if err := json.Unmarshal([]byte(output.Data.Response), &item); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	return item, nil
}

func (a *Api) GetStatement() (ListOfBankResponse, error) {
	req := ListBanksRequest{
		BaseApiReq: BaseApiReq{
			Referenceid:   fmt.Sprintf("%d", time.Now().UnixMilli()),
			RequestType:   153,
			Translocation: "N/A", //defaultLocation,
		},
	}
	url := "/api/Spay/GetStatement"
	method := http.MethodPost
	inputBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	logrus.WithField("req", string(inputBytes)).Info("request body")

	base64Encrypted, err := a.encrypt(string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("3des encryption: %w", err)
	}

	result, err := a.request(url, method, []byte(base64Encrypted))
	if err != nil {
		return nil, fmt.Errorf("interbank transfer request: %w", err)
	}

	if a.shouldDecryptResponse {
		decodedStr, err := a.decrypt(string(result))
		if err != nil {
			return nil, fmt.Errorf("decoded interbank transfer response: %w", err)
		}
		result = []byte(decodedStr)
	}

	var output ApiOperationResponse[ApiOperationResponseData]
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	if output.Data.Status != successfulStatus {
		return nil, fmt.Errorf("could not complete request: %s", output.Data.Response)
	}

	var item ListOfBankResponse
	if err := json.Unmarshal([]byte(output.Data.Response), &item); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	return item, nil
}

func (a *Api) BalanceEnquiry() (ListOfBankResponse, error) {
	req := ListBanksRequest{
		BaseApiReq: BaseApiReq{
			Referenceid:   fmt.Sprintf("%d", time.Now().UnixMilli()),
			RequestType:   151,
			Translocation: defaultLocation,
		},
	}
	url := "/api/Spay/BalanceEnquiry"
	method := http.MethodPost
	inputBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	logrus.WithField("req", string(inputBytes)).Info("request body")

	base64Encrypted, err := a.encrypt(string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("3des encryption: %w", err)
	}

	result, err := a.request(url, method, []byte(base64Encrypted))
	if err != nil {
		return nil, fmt.Errorf("interbank transfer request: %w", err)
	}

	if a.shouldDecryptResponse {
		decodedStr, err := a.decrypt(string(result))
		if err != nil {
			return nil, fmt.Errorf("decoded interbank transfer response: %w", err)
		}
		result = []byte(decodedStr)
	}

	var output ApiOperationResponse[ApiOperationResponseData]
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	if output.Data.Status != successfulStatus {
		return nil, fmt.Errorf("could not complete request: %s", output.Data.Response)
	}

	var item ListOfBankResponse
	if err := json.Unmarshal([]byte(output.Data.Response), &item); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	return item, nil
}

func (a *Api) SterlingTransfer(req *SterlingToSterlingTransferRequest) (*SterlingToSterlingTransferResult, error) {

	if req == nil {
		return nil, ErrInvalidArgument
	}

	if req.ReferenceId == "" {
		ref, err := gonanoid.New(15)
		if err != nil {
			return nil, fmt.Errorf("could not generate nano id reference: %w", err)
		}
		req.ReferenceId = ref
	}

	if req.Translocation == "" {
		req.Translocation = defaultLocation
	}

	sterlingReq := sterlingToSterlingTransfer{
		BaseApiReq: BaseApiReq{
			Referenceid:   req.ReferenceId,
			RequestType:   110,
			Translocation: req.Translocation,
		},
		Amt:        fmt.Sprintf("%.2f", req.Amt),
		Tellerid:   req.Tellerid,
		Frmacct:    a.config.FromAccount,
		Toacct:     req.ToAcct,
		PaymentRef: req.PaymentRef,
		Remarks:    req.Remarks,
	}

	url := "/api/Spay/SBPT24txnRequest"
	method := http.MethodPost
	inputBytes, err := json.Marshal(sterlingReq)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	base64Encrypted, err := a.encrypt(string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("3des encryption: %w", err)
	}

	result, err := a.request(url, method, []byte(base64Encrypted))
	if err != nil {
		if errors.Is(err, ErrInsufficientFunds) {
			return nil, ErrInsufficientFunds
		}

		if errors.Is(err, ErrToAccountNotAllowed) {
			return nil, ErrToAccountNotAllowed
		}

		if aErr, ok := err.(*ApiResponseErrorResult); ok {
			if aErr.Response == ErrToAccountNotAllowed.Response {
				return nil, ErrToAccountNotAllowed
			}
		}

		return nil, fmt.Errorf("intrabank transfer request: %w", err)
	}

	if a.shouldDecryptResponse {
		decodedStr, err := a.decrypt(string(result))
		if err != nil {
			return nil, fmt.Errorf("decoded interbank transfer response: %w", err)
		}
		result = []byte(decodedStr)
	}

	var output SterlingToSterlingTransferResult
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	if output.Response != successfulStatusCode {
		logrus.WithField("transferResult", output).WithField("request", req).Error("sterling intrabank transfer completed without success")
		return nil, fmt.Errorf("could not complete transfer: %v", output.Message)
	}

	return &output, nil
}

func (a *Api) SterlingNameEnquiry(accountNumber string) (*SterlingNameEnquiryResponse, error) {
	req := sterlingNameEnquiryReq{
		BaseApiReq: BaseApiReq{
			Referenceid:   fmt.Sprintf("%d", time.Now().UnixMilli()),
			RequestType:   219,
			Translocation: defaultLocation,
		},
		NUBAN: accountNumber,
	}

	url := "/api/Spay/SBPNameEnquiry"
	method := http.MethodPost
	inputBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	base64Encrypted, err := a.encrypt(string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("3des encryption: %w", err)
	}

	result, err := a.request(url, method, []byte(base64Encrypted))
	if err != nil {
		return nil, fmt.Errorf("interbank transfer request: %w", err)
	}

	if a.shouldDecryptResponse {
		decodedStr, err := a.decrypt(string(result))
		if err != nil {
			return nil, fmt.Errorf("decoded interbank transfer response: %w", err)
		}
		result = []byte(decodedStr)
	}

	var output ApiOperationResponse[SterlingNameEnquiryResponse]
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	if output.Data.Status != successfulStatusCode {
		return nil, fmt.Errorf("could not complete request: %s", output.Response)
	}

	return &output.Data, nil
}

func (a *Api) OtherBanksNameEnquiry(accountNumber, bankCode string) (*InterbankNameEnquiryResponseData, error) {
	ref, err := gonanoid.New(15)
	if err != nil {
		return nil, fmt.Errorf("could not generate nano id reference: %w", err)
	}
	req := interBankNameEnquiryReq{
		BaseApiReq: BaseApiReq{
			Referenceid:   ref,
			RequestType:   161,
			Translocation: defaultLocation,
		},
		ToAccount:           accountNumber,
		DestinationBankCode: bankCode,
	}

	url := "/api/Spay/InterbankNameEnquiry"
	method := http.MethodPost
	inputBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("json encoding: %w", err)
	}

	base64Encrypted, err := a.encrypt(string(inputBytes))
	if err != nil {
		return nil, fmt.Errorf("3des encryption: %w", err)
	}

	result, err := a.request(url, method, []byte(base64Encrypted))
	if err != nil {
		return nil, fmt.Errorf("interbank transfer request: %w", err)
	}

	if a.shouldDecryptResponse {
		decodedStr, err := a.decrypt(string(result))
		if err != nil {
			return nil, fmt.Errorf("decoded interbank transfer response: %w", err)
		}
		result = []byte(decodedStr)
	}

	logrus.WithField("rawResult", string(result)).Info("printing raw result")

	var output InterbankNameEnquiryResponse
	if err := json.Unmarshal(result, &output); err != nil {
		return nil, fmt.Errorf("json decoding: %w", err)
	}

	if output.Data.Status != successfulStatusCode {
		return nil, fmt.Errorf("could not complete request: %s", output.Response)
	}

	return &output.Data, nil
}

func (a *Api) ListInflowsForToday() (*ListInflowResponse, error) {

	todayDate := time.Now().Format("2006-01-02")
	reqData := map[string]any{
		"accountNumber": a.config.FromAccount,
		"startDate":     todayDate,
		"endDate":       todayDate,
	}
	reqDataBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("list inflows for today failed: %w", err)
	}

	dataReader := bytes.NewReader(reqDataBytes)
	url := "https://epayments.sterling.ng/NIPRequery/api/GetTransactionController/GetTransactionByAccount"
	newReq, err := http.NewRequest(http.MethodGet, url, dataReader)
	if err != nil {
		return nil, fmt.Errorf("request for requery: %w", err)
	}

	newReq.Header.Add("Content-Type", "application/json")
	logrus.
		WithField("url", url).
		WithField("method", "GET").
		WithField("body", string(reqDataBytes)).
		Info("sending request for inflow re-query")

	result, err := a.httpClient.Do(newReq)
	if err != nil {
		return nil, fmt.Errorf("inflow re-query response: %w", err)
	}

	if result.Body != nil {
		defer result.Body.Close()
	}

	resultBytes, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("spay response reading: %w", err)
	}

	logrus.
		WithField("status", result.Status).
		WithField("statusCode", result.StatusCode).
		WithField("statusCodeText", http.StatusText(result.StatusCode)).
		WithField("body", string(resultBytes)).
		Info("response result")

	if result.StatusCode < 200 || result.StatusCode > 299 {
		if len(resultBytes) <= 0 {
			return nil, fmt.Errorf("empty response received: %s", result.Status)
		}

		var errMsg ApiResponseErrorResult
		if err := json.Unmarshal(resultBytes, &errMsg); err != nil {
			return nil, fmt.Errorf("could not unmarshal error response to error obj: %w", err)
		}
		return nil, &errMsg
	}

	var resultStruct ListInflowResponse
	if err := json.Unmarshal(resultBytes, &resultStruct); err != nil {
		return nil, fmt.Errorf("could not unmarshal response to inflow result obj: %w", err)
	}

	return &resultStruct, nil
}

func (a *Api) ListInflowsForTodayForAccountID(accountNumber string) (*ListInflowForAccountResponse, error) {

	reqData := map[string]any{
		"AccountNumber": accountNumber,
		"SessionID":     "",
	}
	reqDataBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("list inflows for today failed: %w", err)
	}

	dataReader := bytes.NewReader(reqDataBytes)
	url := "https://epayments.sterling.ng/NIPrequeryV2/api/v1.0/NIP/FetchTransactionStatus"
	newReq, err := http.NewRequest(http.MethodPost, url, dataReader)
	if err != nil {
		return nil, fmt.Errorf("request for requery: %w", err)
	}

	newReq.Header.Add("Content-Type", "application/json")
	logrus.
		WithField("url", url).
		WithField("method", "GET").
		WithField("body", string(reqDataBytes)).
		Info("sending request for inflow re-query")

	result, err := a.httpClient.Do(newReq)
	if err != nil {
		return nil, fmt.Errorf("inflow re-query response: %w", err)
	}

	if result.Body != nil {
		defer result.Body.Close()
	}

	resultBytes, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("spay response reading: %w", err)
	}

	logrus.
		WithField("status", result.Status).
		WithField("statusCode", result.StatusCode).
		WithField("statusCodeText", http.StatusText(result.StatusCode)).
		WithField("body", string(resultBytes)).
		Info("response result")

	if result.StatusCode < 200 || result.StatusCode > 299 {
		if len(resultBytes) <= 0 {
			return nil, fmt.Errorf("empty response received: %s", result.Status)
		}

		var errMsg ApiResponseErrorResult
		if err := json.Unmarshal(resultBytes, &errMsg); err != nil {
			return nil, fmt.Errorf("could not unmarshal error response to error obj: %w", err)
		}
		return nil, &errMsg
	}

	var resultStruct ListInflowForAccountResponse
	if err := json.Unmarshal(resultBytes, &resultStruct); err != nil {
		return nil, fmt.Errorf("could not unmarshal response to inflow result obj: %w", err)
	}

	resultStruct.Content = lo.Filter(resultStruct.Content, func(item InflowForAccountItem, i int) bool {
		return item.AccountNumber != a.config.FromAccount
	})

	return &resultStruct, nil
}

func (a *Api) QueryInflowsBySessionID(sessionID string, date time.Time) (*ListInflowForAccountResponse, error) {

	reqData := map[string]any{
		"SessionID":  sessionID,
		"StartDate":  date.Format("2006-01-02"),
		"pageNumber": 1,
	}
	reqDataBytes, err := json.Marshal(reqData)
	if err != nil {
		return nil, fmt.Errorf("list inflows for today failed: %w", err)
	}

	dataReader := bytes.NewReader(reqDataBytes)
	url := "https://epayments.sterling.ng/NIPrequeryV2/api/v1.0/NIP/FetchPreviousTransactionsStatus"
	newReq, err := http.NewRequest(http.MethodPost, url, dataReader)
	if err != nil {
		return nil, fmt.Errorf("request for requery: %w", err)
	}

	newReq.Header.Add("Content-Type", "application/json")
	logrus.
		WithField("url", url).
		WithField("method", "GET").
		WithField("body", string(reqDataBytes)).
		Info("sending request for inflow re-query")

	result, err := a.httpClient.Do(newReq)
	if err != nil {
		return nil, fmt.Errorf("inflow re-query response: %w", err)
	}

	if result.Body != nil {
		defer result.Body.Close()
	}

	resultBytes, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("spay response reading: %w", err)
	}

	logrus.
		WithField("status", result.Status).
		WithField("statusCode", result.StatusCode).
		WithField("statusCodeText", http.StatusText(result.StatusCode)).
		WithField("body", string(resultBytes)).
		Info("response result")

	if result.StatusCode < 200 || result.StatusCode > 299 {
		if len(resultBytes) <= 0 {
			return nil, fmt.Errorf("empty response received: %s", result.Status)
		}

		var errMsg ApiResponseErrorResult
		if err := json.Unmarshal(resultBytes, &errMsg); err != nil {
			return nil, fmt.Errorf("could not unmarshal error response to error obj: %w", err)
		}
		return nil, &errMsg
	}

	var resultStruct ListInflowForAccountResponse
	if err := json.Unmarshal(resultBytes, &resultStruct); err != nil {
		return nil, fmt.Errorf("could not unmarshal response to inflow result obj: %w", err)
	}

	resultStruct.Content = lo.Filter(resultStruct.Content, func(item InflowForAccountItem, i int) bool {
		return item.AccountNumber != a.config.FromAccount
	})

	return &resultStruct, nil
}

func (a *Api) request(uri, method string, data []byte) ([]byte, error) {

	var dataReader io.Reader
	if len(data) > 0 {
		dataReader = bytes.NewReader(data)
	}
	url := fmt.Sprintf("%s%s", a.config.baseUrl, uri)
	newReq, err := http.NewRequest(method, url, dataReader)
	if err != nil {
		return nil, fmt.Errorf("spay request: %w", err)
	}

	newReq.Header.Add("AppId", fmt.Sprintf("%d", a.config.appId))
	logrus.
		WithField("url", url).
		WithField("method", method).
		WithField("body", string(data)).
		WithField("AppId", a.config.appId).
		Info("sending request")

	result, err := a.httpClient.Do(newReq)
	if err != nil {
		return nil, fmt.Errorf("spay response: %w", err)
	}

	if result.Body != nil {
		defer result.Body.Close()
	}

	resultBytes, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("spay response reading: %w", err)
	}

	logrus.
		WithField("status", result.Status).
		WithField("statusCode", result.StatusCode).
		WithField("statusCodeText", http.StatusText(result.StatusCode)).
		WithField("body", string(resultBytes)).
		Info("response result")

	if result.StatusCode < 200 || result.StatusCode > 299 {

		logrus.
			WithField("status", result.Status).
			WithField("statusCode", result.StatusCode).
			WithField("statusCodeText", http.StatusText(result.StatusCode)).
			WithField("body", string(resultBytes)).
			Error("spay error response result")

		if len(resultBytes) <= 0 {
			return nil, fmt.Errorf("empty response received: %s", result.Status)
		}

		var errMsg ApiResponseErrorResult
		if err := json.Unmarshal(resultBytes, &errMsg); err != nil {
			return nil, fmt.Errorf("could not unmarshal error response to error obj: %w", err)
		}
		return nil, &errMsg
	}

	return resultBytes, nil
}

func (a *Api) encrypt(val string) (string, error) {
	sharedKeyVal, err := a.config.sharedKey.AsByteSlice()
	if err != nil {
		return "", err
	}

	sharedVectorVal, err := a.config.sharedVector.AsByteSlice()
	if err != nil {
		return "", err
	}

	logrus.WithField("sharedKeyVal", fmt.Sprintf("%x", sharedKeyVal)).
		WithField("sharedVectorVal", fmt.Sprintf("%x", sharedVectorVal)).
		Info("binary conversion to string")
	encryptedResult, err := TripleDESCBCEncrypt(val, sharedKeyVal, sharedVectorVal)
	if err != nil {
		return "", err
	}

	decrypedResult, err := TripleDESCBCDecrypt(encryptedResult, sharedKeyVal, sharedVectorVal)
	logrus.WithField("decryptedResult", decrypedResult).WithError(err).Info("decryption test")
	return encryptedResult, nil
}

func (a *Api) decrypt(val string) (string, error) {
	sharedKeyVal, err := a.config.sharedKey.AsByteSlice()
	if err != nil {
		return "", err
	}
	sharedVectorVal, err := a.config.sharedVector.AsByteSlice()
	if err != nil {
		return "", err
	}
	return TripleDESCBCEncrypt(val, sharedKeyVal, sharedVectorVal)
}

func (a *Api) GetTransferCost() float64 {
	return a.config.transferCost
}

func (a *Api) GetOriginAccount() string {
	return a.config.FromAccount
}

func (a *Api) GetBankCode() string {
	return sterlingBankCbnCode
}
