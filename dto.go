package spay

import (
	"fmt"
	"time"
)

var (
	ErrInsufficientFunds = &ApiResponseErrorResult{Response: "x51", Data: ApiResponseErrorResultData{
		ResponseText: "Insufficient Funds",
	}}
	ErrToAccountNotAllowed = &ApiResponseErrorResult{Response: "03x", Data: ApiResponseErrorResultData{
		ResponseText: "To Account is not Allowed for this Operation",
	}}
)

type BaseApiReq struct {
	Referenceid   string `json:"Referenceid"`
	RequestType   int    `json:"RequestType"`
	Translocation string `json:"Translocation"`
}

type ListOfBankResponse []BankResponse

type BankResponse struct {
	BankName string `json:"BANKNAME"`
	BankCode string `json:"BANKCODE"`
}

type ApiOperationResponseData struct {
	Response string `json:"response"`
	Status   string `json:"status"`
}

type ApiOperationResponse[T any] struct {
	Message      string `json:"message"`
	Response     string `json:"response"`
	Responsedata any    `json:"Responsedata"`
	Data         T      `json:"data"`
}

type ApiResponseErrorResultData struct {
	ResponseText string `json:"ResponseText"`
	Status       any    `json:"status"`
}

type ApiResponseErrorResult struct {
	Message      string                     `json:"message"`
	Response     string                     `json:"response"`
	Responsedata any                        `json:"Responsedata"`
	Data         ApiResponseErrorResultData `json:"data"`
}

func (a ApiResponseErrorResult) Is(target error) bool {
	return target.Error() == a.Error()
}

func (a ApiResponseErrorResult) Error() string {
	return a.String()
}

func (a ApiResponseErrorResult) String() string {
	return fmt.Sprintf("message=%s code=%s", a.Data.ResponseText, a.Response)
}

type ApiDebugErrorResult struct {
	Message          string `json:"Message"`
	ExceptionMessage string `json:"ExceptionMessage"`
	ExceptionType    string `json:"ExceptionType"`
	StackTrace       string `json:"StackTrace"`
}

func (a ApiDebugErrorResult) String() string {
	return fmt.Sprintf("message: %s, exception: %s, type: %s, stack: %s", a.Message, a.ExceptionMessage, a.ExceptionType, a.StackTrace)
}

func (a ApiDebugErrorResult) Error() string {
	return a.String()
}

type SterlingToSterlingTransferResultData struct {
	Status string `json:"status"`
}

type SterlingToSterlingTransferResult struct {
	Message      string                               `json:"message"`
	Response     string                               `json:"response"`
	Responsedata interface{}                          `json:"Responsedata"`
	Data         SterlingToSterlingTransferResultData `json:"data"`
}

type SterlingToSterlingTransferRequest struct {
	ReferenceId   string  `json:"Referenceid"`
	Translocation string  `json:"Translocation"`
	PaymentRef    string  `json:"paymentRef"`
	Amt           float64 `json:"amt"`
	ToAcct        string  `json:"toacct"`
	Remarks       string  `json:"remarks"`
	Tellerid      string  `json:"tellerid"`
}

type sterlingToSterlingTransfer struct {
	BaseApiReq
	Amt        string `json:"amt"`
	Tellerid   string `json:"tellerid"`
	Frmacct    string `json:"frmacct"`
	Toacct     string `json:"toacct"`
	PaymentRef string `json:"paymentRef"`
	Remarks    string `json:"remarks"`
}

type SterlingNameEnquiryResponse struct {
	AccountName   string `json:"AccountName"`
	AccountNumber string `json:"AccountNumber"`
	Status        string `json:"status"`
	BVN           string `json:"BVN"`
}

type sterlingNameEnquiryReq struct {
	BaseApiReq
	NUBAN string `json:"NUBAN"`
}

type interBankNameEnquiryReq struct {
	BaseApiReq
	ToAccount           string `json:"ToAccount"`
	DestinationBankCode string `json:"DestinationBankCode"`
}

type ListBanksRequest struct {
	BaseApiReq
}

type interBankTransferRequest struct {
	BaseApiReq
	SessionID           string `json:"SessionID"`
	FromAccount         string `json:"FromAccount"`
	ToAccount           string `json:"ToAccount"`
	Amount              string `json:"Amount"`
	DestinationBankCode string `json:"DestinationBankCode"`
	NEResponse          string `json:"NEResponse"`
	BenefiName          string `json:"BenefiName"`
	PaymentReference    string `json:"PaymentReference"`
	Tellerid            string `json:"tellerid"`
	Remarks             string `json:"remarks"`
}
type InterBankTransferResultData struct {
	ResponseText interface{} `json:"ResponseText"`
	Status       string      `json:"status"`
}

type InterBankTransferResult struct {
	Message      string                      `json:"message"`
	Response     string                      `json:"response"`
	Responsedata interface{}                 `json:"Responsedata"`
	Data         InterBankTransferResultData `json:"data"`
}

type InterBankTransferRequest struct {
	Translocation        string `json:"Translocation"`
	PaymentReference     string `json:"PaymentReference"`
	Reference            string `json:"reference"`
	ToAccount            string `json:"ToAccount"`
	Amount               string `json:"Amount"`
	DestinationBankCode  string `json:"DestinationBankCode"`
	NEResponse           string `json:"NEResponse"`
	BenefiName           string `json:"BenefiName"`
	Tellerid             string `json:"tellerid"`
	Remarks              string `json:"remarks"`
	NameEnquirySessionID string `json:"-"`
}

type InterbankNameEnquiryResponse struct {
	Message      string                           `json:"message"`
	Response     string                           `json:"response"`
	Responsedata any                              `json:"Responsedata"`
	Data         InterbankNameEnquiryResponseData `json:"data"`
}

type InterbankNameEnquiryResponseData struct {
	AccountName   string      `json:"AccountName"`
	SessionID     string      `json:"sessionID"`
	AccountNumber string      `json:"AccountNumber"`
	Status        string      `json:"status"`
	BVN           string      `json:"BVN"`
	ResponseText  interface{} `json:"ResponseText"`
}

type InflowNotificationResult struct {
	AccountNumber               string `json:"AccountNumber"`
	ResponseCode                string `json:"ResponseCode"`
	Amount                      string `json:"Amount"`
	SourceCustomerName          string `json:"SourceCustomerName"`
	SourceCustomerAccountNumber string `json:"SourceCustomerAccountNumber"`
	Dateposted                  string `json:"Dateposted"`
	SenderBank                  string `json:"SenderBank"`
	PaymentRef                  string `json:"PaymentRef"`
	Requery                     string `json:"Requery"`
	FTReference                 string `json:"FTReference"`
	SessionID                   string `json:"SessionID"`
	Remark                      string `json:"Remark"`
}

type ListInflowResponse struct {
	Success bool                       `json:"Success"`
	Data    []InflowNotificationResult `json:"Data"`
	Message string                     `json:"Message"`
}

type InflowForAccountItem struct {
	AccountNumber               string  `json:"accountNumber"`
	ResponseCode                string  `json:"responseCode"`
	Amount                      string  `json:"amount"`
	SourceCustomerName          string  `json:"sourceCustomerName"`
	SourceCustomerAccountNumber string  `json:"sourceCustomerAccountNumber"`
	Dateposted                  string  `json:"dateposted"`
	SenderBank                  string  `json:"senderBank"`
	PaymentRef                  string  `json:"paymentRef"`
	Requery                     *string `json:"requery"`
	FtReference                 *string `json:"ftReference"`
	SessionID                   string  `json:"sessionID"`
	Remark                      string  `json:"remark"`
}

type ListInflowForAccountResponse struct {
	Content      []InflowForAccountItem `json:"content"`
	Error        interface{}            `json:"error"`
	HasError     bool                   `json:"hasError"`
	ErrorMessage string                 `json:"errorMessage"`
	Message      string                 `json:"message"`
	RequestId    string                 `json:"requestId"`
	IsSuccess    bool                   `json:"isSuccess"`
	RequestTime  time.Time              `json:"requestTime"`
	ResponseTime time.Time              `json:"responseTime"`
}
