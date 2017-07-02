package request

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"fmt"

	"github.com/user/gobet/betfair.com/aping"
	"github.com/user/gobet/betfair.com/login"
	"github.com/user/gobet/traficControl"
)

func GetResponse(xAuthentication string, appKey *string,
		endpoint aping.Endpoint, params interface{}) (responseBody []byte, err error) {

	jsonReq := struct {
		Jsonrpc string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params"`
		Id      int         `json:"id"`
	}{"2.0", endpoint.Method, params, 1}

	var reqbytes []byte
	if reqbytes, err = json.Marshal(&jsonReq); err != nil {
		return
	}

	var req *http.Request
	if req, err = http.NewRequest("POST", endpoint.URL, bytes.NewBuffer(reqbytes)); err != nil {
		return
	}
	req.ContentLength = int64(len(reqbytes))
	if appKey != nil {
		req.Header.Set("X-Application", *appKey)
	}

	req.Header.Set("X-Authentication", xAuthentication)
	req.Header.Set("ContentType", "application/json")
	req.Header.Set("AcceptCharset", "UTF-8")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	var resp *http.Response
	if resp, err = client.Do(req); err != nil {
		return
	}
	defer resp.Body.Close()

	responseBody, err = ioutil.ReadAll(resp.Body)

	traficControl.AddTotalBytesReaded(len(responseBody), "API-NG")

	return
}

func GetResponseWithAdminLogin(appKey *string, endpoint aping.Endpoint, params interface{}) (responseBody []byte, err error) {

	chLogin := make(chan login.Result)
	login.GetAdminAuth(chLogin)
	rLogin := <-chLogin
	if rLogin.Error != nil {
		err = fmt.Errorf("login error: %v", rLogin.Error)
		return
	}
	return GetResponse(rLogin.Token, appKey, endpoint, params )
}
