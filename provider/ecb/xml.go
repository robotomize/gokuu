package ecb

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strconv"
	"sync"
	"time"

	"github.com/robotomize/gokuu/label"
)

var errXMLAttrNotFound = errors.New("node not found")

const xmlCubeElement = "Cube"

var xmlNodePool = sync.Pool{
	New: func() interface{} { return &XMLNode{} },
}

// decodeXML returns the decoding function. decodeXML parses xml in streaming mode and returns currency pairs by date
func decodeXML() decodeFunc {
	return func(b []byte, iterFunc func(rates euroLatestRates) error) error {
		if iterFunc == nil {
			return errMissingIterFunc
		}
		decoder := xml.NewDecoder(bytes.NewReader(b))
	TokenLoop:
		for {
			token, err := decoder.Token()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break TokenLoop
				}

				var syntaxErr *xml.SyntaxError
				if errors.As(err, &syntaxErr) {
					return fmt.Errorf("%w: %v", errDecodeToken, syntaxErr.Error())
				}

				return fmt.Errorf("decode token: %w", err)
			}

			switch tp := token.(type) {
			case xml.StartElement:
				// Check the presence of the Cube element with attributes
				if isXMLCubeElement(tp.Name.Local) {
					if len(tp.Attr) == 0 {
						continue TokenLoop
					}

					currNode := xmlNodePool.Get().(*XMLNode)

					// Decode a piece of the tree into an XMLNode element, which represents the exchange rate for the day
					if err := decoder.DecodeElement(&currNode, &tp); err != nil {
						var syntaxErr *xml.SyntaxError
						switch {
						case errors.As(err, &syntaxErr):
							return fmt.Errorf("%w: %v", errDecodeToken, syntaxErr.Error())
						case errors.Is(err, errCcyNotFound), errors.Is(err, errXMLAttrNotFound):
							continue TokenLoop
						case errors.Is(err, errAttributeNotValid):
							return errAttributeNotValid
						default:
							return fmt.Errorf("decode element: %w", err)
						}
					}

					dailyRate := euroLatestRates{
						time:  time.Time(currNode.Time),
						rates: make([]euroExchangeRate, 0, len(currNode.Rates)),
					}

					for _, r := range currNode.Rates {
						dailyRate.rates = append(dailyRate.rates, euroExchangeRate{
							symbol: label.Symbol(r.Currency),
							rate:   r.Rate.Float64(),
						})
					}

					if err := iterFunc(dailyRate); err != nil {
						return fmt.Errorf("handle func: %w", err)
					}

					currNode = nil
					xmlNodePool.Put(currNode)
				}
			case xml.EndElement:
			case xml.CharData:
			}
		}

		return nil
	}
}

func isXMLCubeElement(name string) bool {
	return name == xmlCubeElement
}

type XMLAttrTime time.Time

func (x *XMLAttrTime) UnmarshalXMLAttr(attr xml.Attr) error {
	t, err := time.Parse("2006-01-02", attr.Value)
	if err != nil {
		return fmt.Errorf("%w: %v", errAttributeNotValid, err)
	}

	*x = XMLAttrTime(t)

	return nil
}

var _ xml.UnmarshalerAttr = (*XMLCurrencyAttr)(nil)

type XMLCurrencyAttr string

func (i *XMLCurrencyAttr) UnmarshalXMLAttr(attr xml.Attr) error {
	currencySymbol := (label.Symbol)(attr.Value)
	if _, ok := label.Currencies[currencySymbol]; !ok {
		return errCcyNotFound
	}

	*i = XMLCurrencyAttr(currencySymbol)
	return nil
}

var _ xml.UnmarshalerAttr = (*XMLRateAttr)(nil)

type XMLRateAttr float64

func (i XMLRateAttr) Float64() float64 {
	return float64(i)
}

func (i *XMLRateAttr) UnmarshalXMLAttr(attr xml.Attr) error {
	rate, err := strconv.ParseFloat(attr.Value, 64)
	if err != nil {
		return fmt.Errorf("%w: %v", errAttributeNotValid, err)
	}

	if rate <= 0 {
		return errAttributeNotValid
	}

	*i = XMLRateAttr(rate)

	return nil
}

type XMLNode struct {
	Time  XMLAttrTime `xml:"time,attr"`
	Rates []struct {
		Currency XMLCurrencyAttr `xml:"currency,attr"`
		Rate     XMLRateAttr     `xml:"rate,attr"`
	} `xml:"Cube"`
}
