package proxi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/user/gobet/utils"
)

func createProxiRequest(urlStr string, r *http.Request) (*http.Request, error) {
	req, err := http.NewRequest(r.Method, urlStr, r.Body)

	if err != nil {
		return nil, fmt.Errorf("can not create http request - %s", err.Error())
	}

	for key, value := range r.Header {
		req.Header.Set(key, strings.Join(value, "; "))
	}

	return req, nil
}

// Proxi отправляет исходный http запрос /get/:password/*url на целевой адрес из поля url
// и записывает полученный http ответ в веб-контекст
func Proxi(c *gin.Context) {

	internalServerError := func(e error) {
		c.String(http.StatusInternalServerError, e.Error())
	}
	url, urlParsed := utils.QueryUnescape(c.Param("url"))
	if !urlParsed {
		c.String(http.StatusBadRequest, "can't parse target url")
		return
	}

	rreq, err := createProxiRequest(url, c.Request)
	if err != nil {
		internalServerError(err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(rreq)
	if err != nil {
		internalServerError(fmt.Errorf("can't send request an get response in http.Client.Do - %s", err.Error()))
		return
	}
	defer response.Body.Close()

	for key, value := range response.Header {
		c.Header(key, strings.Join(value, "; "))
	}

	c.Writer.WriteHeader(response.StatusCode)
	body, err := ioutil.ReadAll(response.Body)
	if err == nil {
		_, err := c.Writer.Write(body[:])
		if err != nil {
			internalServerError(fmt.Errorf("can't write data to response body - %s", err.Error()))
		}
	}

}
