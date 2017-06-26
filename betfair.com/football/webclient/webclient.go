package webclient

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"log"
	"github.com/PuerkitoBio/goquery"
	"github.com/user/gobet/betfair.com/football"
	"github.com/user/gobet/betfair.com/football/parse"
	"github.com/user/gobet/utils"
	"github.com/user/gobet/config"
	"io/ioutil"
	"github.com/user/gobet/traficControl"
	"encoding/json"
)

const (
	BetfairURL = "https://www.betfair.com"
	ruCoockie = `vid=39ad9e9d-12e6-487e-8f0c-fb89d881c015; bucket=2~0~test_search; wsid=13f06991-5f1a-11e6-862c-90e2ba0fa6a0; betexPtk=betexCurrency%3DGBP%7EbetexTimeZone%3DEurope%2FLondon%7EbetexRegion%3DGBR%7EbetexLocale%3Dru; mEWJSESSIONID=AA42CA3984085031B4C9F344A940BACB; betexPtkSess=betexCurrencySessionCookie%3DGBP%7EbetexRegionSessionCookie%3DGBR%7EbetexTimeZoneSessionCookie%3DEurope%2FLondon%7EbetexLocaleSessionCookie%3Dru%7EbetexSkin%3Dstandard%7EbetexBrand%3Dbetfair; PI=61999; pi=partner61999; UI=0; spi=0; bfsd=ts=1470847622600|st=p; _qst_s=1; _qsst_s=1470847831600; betfairSSC=lsSSC%3D1%3Bcookie-policy%3D1; _ga=GA1.2.783247160.1470847628; _gat=1; _qubitTracker=1470847628500.818360; _qubitTracker_s=1470847628500.818360; _qPageNum_betfair=1; _qst=%5B1%2C0%5D; _qsst=1470847840800; qb_ss_status=BOA5:Ma&OsT|BOBI:Ik&OsY|BOBQ:D0&Osa|BOBk:OP&Osd|BOKN:J&Ot6; _qb_se=BOA5:OsT&VZ1XPKE|BOBI:OsY&VZ1XPKE|BOBQ:Osa&VZ1WbWc|BOBk:Osd&VZ1XPKE|BOKN:Ot6&VZ1XPKE; qb_permanent=:0:0:0:0:0::0:1:0::::::::::::::::::::K6M&OMN&OsY&Osd&OuS&OsT&Osa&Ot6:VZ1XPKE; _q_geo=%5B%222%22%2C%2293.115.95.202%22%2C%22RO%22%2C%2212072%22%2C%22unknown%22%2C%2217843%22%2C%2244.4599%22%2C%2226.1333%22%5D; qb_cc=RO; update-browser=Wed%20Aug%2010%202016%2016%3A47%3A39%20GMT%2B0000%20(UTC); exp=ex; pref_md_pers_0="{\"com-es-info\":{\"spainRedirectNotification\":\"false\"}}"; ss_opts=BOA5:C&C|BOBI:C&C|BOBQ:B&B|BOBk:C&C|BOKN:C&C|_g:VZ1WbU4&VZ1XPIg&B&C`
)

var header1 = http.Header{
	"Accept-Language": {"ru-RU,ru;q=0.8,en-US;q=0.5,en;q=0.3"},
	"Accept-Encoding": {"gzip,deflate,sdch"},

	"User-Agent": {"Mozilla/5.0 (Windows NT 6.1; rv:45.0) Gecko/20100101 Firefox/45.0"},

	"Accept":  {"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"},
	"Cookie":  {ruCoockie},
	"Referer": {"http://www.betfair.com/ru/"},
}




func downloadURL(srcURL string, header http.Header) (*goquery.Document, error) {
	fail := func(fileLine string, e error) (*goquery.Document, error) {
		return nil, fmt.Errorf("betfairPage, DownloadURL, %q, %s, %s", srcURL, fileLine, e.Error())
	}
	URL := srcURL
	if config.Get().UseBetfairProxi {
		// https://betproxi.herokuapp.com/test/proxi/https%3A%2F%2Fwww.betfair.com%2Fexchange%2Ffootball
		//URL = fmt.Sprintf("http://betproxi.herokuapp.com/test/proxi/%s",
		URL = fmt.Sprintf("http://gobet.herokuapp.com/proxi/%s",
			utils.QueryEscape(URL))
	}
	request, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return fail(utils.FuncFileLine(), err)
	}

	for k, v := range header1 {
		request.Header[k] = v
	}
	for k, v := range header {
		request.Header[k] = v
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fail(utils.FuncFileLine(), err)
	}

	if config.Get().ControlTraffic {
		log.Printf("Control traffic: %v", srcURL)
	}

	var reader io.Reader = response.Body

	if response.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(response.Body)
		if err != nil {
			return fail(utils.FuncFileLine(), err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return fail(utils.FuncFileLine(), err)
	}

	return doc, nil
}

func ReadFirstPageURL() (string, error) {
	const (
		URL = "https://www.betfair.com/exchange/football"
	)

	var header = http.Header{
		"Accept-Language": {"en-US"},
		"Cookie":          {},
	}

	doc, err := downloadURL(URL, header)
	if err != nil {
		return "", utils.ErrorWithInfo(err)
	}

	return parse.FirstPageURL(doc)
}

func ReadPage(URL string) (games []football.Match, nextPage *string, err error) {

	doc, errDoc := downloadURL(URL, http.Header{})
	if errDoc != nil {
		err = utils.ErrorWithInfo(errDoc)
		return
	}
	return parse.Page(doc)
}


func ReadMatches() (readedGames []football.Match, err error) {
	var firstPageURL string
	firstPageURL, err = ReadFirstPageURL()
	if err != nil {
		return
	}

	ptrNextPage := &firstPageURL
	for page := 0; ptrNextPage != nil && err == nil; page++ {
		var gamesPage []football.Match
		gamesPage, ptrNextPage, err = ReadPage(BetfairURL + *ptrNextPage)
		if err != nil {
			return
		}
		for _, game := range gamesPage {
			game.Page = page
			readedGames = append(readedGames, game)
		}
	}

	return
}

func ReadMatchesFromHerokuApp() (readedGames []football.Match, err error) {

	var resp *http.Response
	url := "http://gobet.herokuapp.com/football/footballMatches"
	resp, err = http.Get(url)
	if err != nil {
		err = fmt.Errorf("http error of %v: %v", url, err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	traficControl.AddTotalBytesReaded(len(body), "HEROKU APP")

	var data struct {
		Ok  []football.Match `json:"ok,omitempty"`
		Err error            `json:"error,omitempty"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		err = fmt.Errorf("data error of %v: %v", url, err)
		return
	}
	readedGames = data.Ok
	return
}
