package ecb

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/robotomize/gokuu/label"
)

func decodeCSV() decodeFunc {
	return func(b []byte, iterFunc func(rates euroLatestRates) error) error {
		if iterFunc == nil {
			return errMissingIterFunc
		}
		decoder := csv.NewReader(bytes.NewReader(b))
		idx := 0
		var header []string
	TokenLoop:
		for {
			line, err := decoder.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break TokenLoop
				}

				var parseError *csv.ParseError
				if errors.As(err, &parseError) {
					return fmt.Errorf("%w: %v", errDecodeToken, parseError.Error())
				}

				return fmt.Errorf("csv decoder read: %w", err)
			}

			if idx == 0 {
				for n, column := range line {
					token := strings.Trim(column, " \t")
					if n == 0 && token == "" {
						return errAttributeNotValid
					}
					if token == "" {
						continue
					}
					header = append(header, strings.Trim(column, " "))
				}
				idx += 1
				continue TokenLoop
			}

			var dailyRate euroLatestRates

			for n, column := range line {
				token := strings.Trim(column, " \t")
				if token == "" {
					continue
				}

				if n == 0 && header[n] != "Date" {
					return fmt.Errorf("%w: %v", errAttributeNotValid, err)
				}

				if header[n] == "Date" {
					t, err := time.Parse("02 January 2006", token)
					if err != nil {
						return fmt.Errorf("%w: %v", errAttributeNotValid, err)
					}

					dailyRate.time = t
					continue
				}

				symbol := header[n]
				currencySymbol := (label.Symbol)(symbol)
				if _, ok := label.Currencies[currencySymbol]; !ok {
					continue
				}

				r, err := strconv.ParseFloat(token, 64)
				if err != nil {
					return fmt.Errorf("%w: %v", errAttributeNotValid, err)
				}

				if r <= 0 {
					return errAttributeNotValid
				}

				dailyRate.rates = append(dailyRate.rates, euroExchangeRate{
					symbol: currencySymbol,
					rate:   r,
				})
			}

			if err := iterFunc(dailyRate); err != nil {
				return fmt.Errorf("handle func: %w", err)
			}
		}

		return nil
	}
}
