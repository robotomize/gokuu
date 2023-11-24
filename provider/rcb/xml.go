package rcb

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robotomize/gokuu/label"
	"golang.org/x/text/encoding/charmap"
)

var (
	errDecodeToken       = errors.New("decoding of the markup failed")
	errAttributeNotValid = errors.New("attr is not valid")
)

var xmlNodePool = sync.Pool{
	New: func() interface{} { return &XMLNode{} },
}

// decodeXML returns the decoding function. decodeXML parses xml in streaming mode and returns currency pairs by date
func decodeXML(b []byte) (rubLatestRates, error) {
	var dailyRates rubLatestRates
	decoder := xml.NewDecoder(bytes.NewReader(b))
	decoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		switch charset {
		case "windows-1251":
			return charmap.Windows1251.NewDecoder().Reader(input), nil
		}

		return nil, fmt.Errorf("charset is not defined")
	}

TokenLoop:
	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break TokenLoop
			}

			var syntaxErr *xml.SyntaxError
			if errors.As(err, &syntaxErr) {
				return dailyRates, fmt.Errorf("%w: %v", errDecodeToken, syntaxErr.Error())
			}

			return dailyRates, fmt.Errorf("decode token: %w", err)
		}

		switch tp := token.(type) {
		case xml.StartElement:
			// Check the presence of the Cube element with attributes
			if tp.Name.Local == "ValCurs" {
				currNode := xmlNodePool.Get().(*XMLNode)

				// Decode a piece of the tree into an XMLNode element, which represents the exchange rate for the day
				if err := decoder.DecodeElement(&currNode, &tp); err != nil {
					var syntaxErr *xml.SyntaxError
					switch {
					case errors.As(err, &syntaxErr):
						return dailyRates, fmt.Errorf("%w: %v", errDecodeToken, syntaxErr.Error())
					case errors.Is(err, errAttributeNotValid):
						return dailyRates, errAttributeNotValid
					default:
						return dailyRates, fmt.Errorf("decode element: %w", err)
					}
				}

				dailyRates.rates = make([]rubExchangeRate, 0, len(currNode.Rates))
				dailyRates.time = time.Time(currNode.Time)

				for _, r := range currNode.Rates {
					v, err := strconv.ParseFloat(strings.Replace(r.Value, ",", ".", -1), 64)
					if err != nil {
						return dailyRates, fmt.Errorf("strconv.ParseFloat: %w", err)
					}

					if v <= 0 {
						return dailyRates, errAttributeNotValid
					}

					if _, ok := label.Currencies[r.Currency]; !ok {
						continue
					}

					dailyRates.rates = append(
						dailyRates.rates, rubExchangeRate{
							symbol: r.Currency,
							rate:   v,
						},
					)
				}

				currNode = nil
				xmlNodePool.Put(currNode)
			}
		}
	}

	return dailyRates, nil
}

type XMLAttrTime time.Time

func (x *XMLAttrTime) UnmarshalXMLAttr(attr xml.Attr) error {
	t, err := time.Parse("02.01.2006", attr.Value)
	if err != nil {
		return fmt.Errorf("time.Parse: %w", err)
	}

	*x = XMLAttrTime(t)

	return nil
}

type XMLCcyRate struct {
	Currency label.Symbol `xml:"CharCode"`
	Value    string       `xml:"Value"`
	Rate     float64
}

type XMLNode struct {
	Time  XMLAttrTime  `xml:"Date,attr"`
	Rates []XMLCcyRate `xml:"Valute"`
}
