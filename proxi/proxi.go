package proxi

import (
	"fmt"
	"github.com/go-chi/chi"
	"gobet/utils"
	"io/ioutil"
	"net/http"
	"strings"
)

func createProxiRequest(urlStr string, r *http.Request) (*http.Request, error) {
	req, err := http.NewRequest(r.Method, urlStr, r.Body)

	if err != nil {
		return nil, fmt.Errorf("can`t create http request - %s", err.Error())
	}

	for key, value := range r.Header {
		req.Header.Set(key, strings.Join(value, "; "))
	}

	return req, nil
}

// Proxi отправляет исходный http запрос /get/:password/*url на целевой адрес из поля url
// и записывает полученный http ответ в веб-контекст
func Proxi(w http.ResponseWriter, r *http.Request) {

	decodedURL := chi.URLParam(r, "*")
	if decodedURL == "" {
		http.Error(w, "target url is empty", http.StatusBadRequest)
	}

	encodedURL, err := utils.QueryUnescape(decodedURL)

	rreq, err := createProxiRequest(encodedURL, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	response, err := client.Do(rreq)
	if err != nil {
		http.Error(w, fmt.Sprintf("can't send request an get response in http.Client.Do - %s", err.Error()),
			http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()

	for key, value := range response.Header {
		w.Header().Set(key, strings.Join(value, "; "))
	}

	w.WriteHeader(response.StatusCode)
	body, err := ioutil.ReadAll(response.Body)
	if err == nil {
		_, err := w.Write(body[:])
		if err != nil {
			http.Error(w, fmt.Sprintf("can't write data to response body - %s", err.Error()),
				http.StatusInternalServerError)
		}
	}
}
