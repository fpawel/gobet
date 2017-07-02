package appkey

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/nu7hatch/gouuid"

	"github.com/user/gobet/betfair.com/aping"
	"github.com/user/gobet/betfair.com/aping/request"
)

type developerApp struct {
	AppName     string                `json:"appName"`
	AppId       int                   `json:"appId"`
	AppVersions []developerAppVersion `json:"appVersions"`
}

type developerAppVersion struct {
	Owner                string `json:"owner"`
	VersionId            int    `json:"versionId"`
	Version              string `json:"version"`
	ApplicationKey       string `json:"applicationKey"`
	DelayData            bool   `json:"delayData"`
	SubscriptionRequired bool   `json:"subscriptionRequired"`
	OwnerManaged         bool   `json:"ownerManaged"`
	Active               bool   `json:"active"`
	VendorId             string `json:"vendorId,omitempty"`
	VendorSecret         string `json:"vendorSecret,omitempty"`
}

func GetResponse( xAuthentication string, endpoint aping.Endpoint, params interface{}) (responseBody []byte, err error) {
	return request.GetResponse(xAuthentication, &appKeyValue, endpoint, params)
}

func GetResponseWithAdminLogin(endpoint aping.Endpoint, params interface{}) (responseBody []byte, err error) {
	return request.GetResponseWithAdminLogin(&appKeyValue, endpoint, params)
}

func extractApplicationKey1(bytes []byte, out *string) (result bool) {
	var x struct {
		Jsonrpc string         `json:"appName"`
		Result  []developerApp `json:"result"`
	}
	result = json.Unmarshal(bytes, &x) == nil && len(x.Result) > 0 && len(x.Result[0].AppVersions) > 0
	if result {
		*out = x.Result[0].AppVersions[0].ApplicationKey
	}
	return
}

func extractApplicationKey2(bytes []byte, out *string) (result bool) {
	var x struct {
		Jsonrpc string       `json:"appName"`
		Result  developerApp `json:"result"`
	}
	result = json.Unmarshal(bytes, &x) == nil && len(x.Result.AppVersions) > 0
	if result {
		*out = x.Result.AppVersions[0].ApplicationKey
	}
	return
}

func getAppKey() (appKey string, err error) {
	var responseBody []byte

	responseBody, err = request.GetResponseWithAdminLogin(nil, aping.AccauntAPI("getDeveloperAppKeys"), nil)
	if err != nil {
		return
	}

	if extractApplicationKey1(responseBody, &appKey) {
		return
	} else {
		log.Printf("getDeveloperAppKeys: %v\n", string(responseBody))
	}

	var u4 *uuid.UUID
	if u4, err = uuid.NewV4(); err != nil {
		return
	}
	params := struct {
		AppName string `json:"appName"`
	}{u4.String()}

	responseBody, err = request.GetResponseWithAdminLogin(nil, aping.AccauntAPI("createDeveloperAppKeys"), params)
	if err != nil {
		return
	}
	if !extractApplicationKey2(responseBody, &appKey) {
		log.Printf("createDeveloperAppKeys: %v\n", string(responseBody))
		err = errors.New("required fields missing")
	}

	return
}

// Get appKeyValue
func Get() string {
	return appKeyValue
}

// appKeyValue - the unqiue application key associated with this betfair's ApiNG application version
var appKeyValue string

func init() {

	var err error
	appKeyValue, err = getAppKey()
	if err != nil {
		log.Fatalln("can`t get application key:", err)
	}
	log.Printf("app key: %v", appKeyValue)

}
