package cae

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/robotomize/gokuu/label"
	"golang.org/x/net/html"
)

var (
	errParseAttrNotValid = errors.New("attr is not valid")
	errHTMLNotValid      = errors.New("html not valid")
)

func parseHTML(b []byte) (aedLatestRates, error) {
	var dailyRates aedLatestRates
	root, err := html.Parse(bytes.NewReader(b))
	if err != nil {
		return dailyRates, fmt.Errorf("%w: html parse: %v", errHTMLNotValid, err)
	}

	doc := goquery.NewDocumentFromNode(root)

	date := doc.Find("#ratesDatePicker > h3 > span > span").Text()
	if date == "" || len(date) < 4 {
		return dailyRates, errParseAttrNotValid
	}

	date = date[4:]
	dt, err := time.Parse("02-01-2006", date)
	if err != nil {
		return dailyRates, errParseAttrNotValid
	}

	dailyRates.time = dt

	var buf bytes.Buffer

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			// Keep newlines and spaces, like jQuery
			buf.WriteString(n.Data)
		}
		if n.FirstChild != nil {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}
	}

	rateNodes := doc.Find("#ratesDateTable tbody tr td").Nodes
	for i := 0; i < len(rateNodes); i += 2 {
		f(rateNodes[i])

		name := buf.String()
		buf.Reset()
		if name == "" {
			return dailyRates, errParseAttrNotValid
		}

		symbol, ok := label.Names[name]
		if !ok {
			continue
		}

		f(rateNodes[i+1])

		rateStr := buf.String()
		buf.Reset()

		rate, err := strconv.ParseFloat(rateStr, 64)
		if err != nil {
			return aedLatestRates{}, errParseAttrNotValid
		}

		dailyRates.rates = append(dailyRates.rates, aedExchangeRate{
			symbol: symbol,
			rate:   rate,
		})
	}

	return dailyRates, nil
}
