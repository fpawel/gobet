package parse

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/PuerkitoBio/goquery"
	"gobet/betfair.com/football"
	"gobet/utils"
)

func odd(node *goquery.Selection, nSelection int, side string) (pprise *float64) {
	pat := fmt.Sprintf(
		"td.odds.%s.selection-%d button.bet-button.%s.cta.cta-%s[data-bettype='%c']",
		side, nSelection, side, side, unicode.ToUpper([]rune(side)[0]))
	node = node.Find(pat)

	if v, err := strconv.ParseFloat(strings.TrimSpace(node.Text()), 64); err == nil {
		pprise = &v
	}

	return
}

func eventID(node *goquery.Selection) (int, error) {
	strEventID, ok := node.Attr("data-eventid")
	if !ok {
		return 0, utils.NewErrorWithInfo("data-eventid not found")
	}
	return strconv.Atoi(strEventID)

}

func marketID(node *goquery.Selection) (int, error) {
	str, ok := node.Attr("data-marketid")
	if !ok {
		return 0, utils.NewErrorWithInfo("data-marketid not found")
	}

	m := regexp.MustCompile("1.(\\d+)").FindStringSubmatch(str)
	if len(m) != 2 {
		return 0, utils.NewErrorWithInfo(fmt.Sprintf("unexpected data-marketid %v", str))
	}
	return strconv.Atoi(m[1])
}

func odds(node *goquery.Selection) (odds football.Odds) {

	odds.Win1 = odd(node, 1, "back")
	odds.Win2 = odd(node, 1, "lay")
	odds.Draw1 = odd(node, 2, "back")
	odds.Draw2 = odd(node, 2, "lay")
	odds.Lose1 = odd(node, 3, "back")
	odds.Lose2 = odd(node, 3, "lay")
	return
}

func game(node *goquery.Selection) (game football.Match, err error) {

	if game.EventID, err = eventID(node); err != nil {
		return
	}

	if game.MarketID, err = marketID(node); err != nil {
		return
	}

	strf := func(s string) string {
		return node.Find(s).Text()
	}

	game.Home = strf("span.home-team")
	game.Away = strf("span.away-team")
	game.Time = strf("span.start-time")
	game.Result = strf("span.result")
	game.Odds = odds(node)

	return
}

func FirstPageURL(doc *goquery.Document) (string, error) {
	const (
		pattern = "a.more-events[href]"
	)

	if v, ok := doc.Find(pattern).First().Attr("href"); ok {
		return v, nil
	}

	s, _ := doc.Html()

	return "", utils.ErrorWithInfo(fmt.Errorf("pattern %v not found, %s", pattern, s))
}

func Page(doc *goquery.Document) (games []football.Match, nextPage *string, err error) {

	doc.Find("tbody[data-marketid][data-eventid]").Each(func(i int, x *goquery.Selection) {
		game, errGame := game(x)
		if errGame != nil {
			err = utils.ErrorWithInfo(errGame)
			return
		}
		game.Live.Order = len(games)
		games = append(games, game)
	})

	nodeNextPage := doc.Find("a.next-page")
	if v, ok := nodeNextPage.Attr("href"); ok {
		nextPage = &v
	}

	return
}
