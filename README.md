# spay

```go

package main

import (
	"fmt"
	"time"

	"github.com/akacokafor/spay"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
)

func main() {
	sharedKeyValue := ""  //put your really long binary string here
	sharedKeyVector := "" //put the short binary string here
	defaultAppId := int32(11111) // your app id
	defaultFromAccount := "0000000000" //your sterling bank NPS settlement or collection account number
	prodUrl := "https://webapps.sterling.ng/Spay_Statement"

	shouldDecryptResponse := false

	spayApi, err := spay.NewApi(
		spay.BitString(sharedKeyVector),
		spay.BitString(sharedKeyValue),
		defaultAppId,
		defaultFromAccount,
		shouldDecryptResponse,
		prodUrl,
	)
	if err != nil {
		logrus.Fatal(err)
	}
	//
	banksList, err := spayApi.ListBanks()
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.WithField("banks", banksList).Infoln("banks gotten")

	acctName, err := spayApi.SterlingNameEnquiry("0000000000") //sterling bank account number
	logrus.WithError(err).WithField("acctName", acctName).Infoln("account name for sterling")

	r, err := spayApi.SterlingTransfer(&spay.SterlingToSterlingTransferRequest{
		ReferenceId:   fmt.Sprintf("%d", time.Now().UnixMilli()),
		Translocation: "100,100",
		PaymentRef:    ksuid.New().String(),
		Amt:           100.0,
		ToAcct:        "0000000000", //"any sterling bank account",
		Remarks:       "Test",
		Tellerid:      "1111",
	})
	logrus.WithError(err).WithField("result", r).Infoln("sterling to sterling transfer")

	otherBank, err := spayApi.OtherBanksNameEnquiry("0000000000", "000014") //000014 is access bank

	logrus.WithError(err).WithField("otherBank", otherBank).Infoln("account name for access")
	transferToOther, err := spayApi.InitiateInterBankTransfer(&spay.InterBankTransferRequest{
		PaymentReference:     ksuid.New().String(),
		Reference:            ksuid.New().String(),
		ToAccount:            "0000000000", //account number
		Amount:               "101.00",
		Tellerid:             "1111",
		DestinationBankCode:  "000014", //this is access bank
		Translocation:        "6.44,3.53",
		NEResponse:           otherBank.AccountName,
		BenefiName:           otherBank.AccountName,
		NameEnquirySessionID: otherBank.SessionID,
		Remarks:              "test",
	})

	if err != nil {
		logrus.Fatal(err)
	}
	//
	logrus.WithField("transferToOther", transferToOther).Infoln("ended transfer attempt")

}

```
