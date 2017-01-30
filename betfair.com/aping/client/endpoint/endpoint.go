package endpoint

type Endpoint struct {
	URL    string
	Method string
}

func AccauntAPI(s string) (r Endpoint) {
	r.URL = "https://api.betfair.com/exchange/account/json-rpc/v1"
	r.Method = "AccountAPING/v1.0/" + s
	return
}

func BettingAPI(s string) (r Endpoint) {
	r.URL = "https://api.betfair.com/exchange/betting/json-rpc/v1"
	r.Method = "SportsAPING/v1.0/" + s
	return

}
