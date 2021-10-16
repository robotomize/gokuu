package ecb

import (
	"errors"
	"time"

	"github.com/robotomize/gokuu/label"
)

var (
	errDecodeToken       = errors.New("decoding of the markup failed")
	errAttributeNotValid = errors.New("attr is not valid")
	errCcyNotFound       = errors.New("currency symbol not found")
	errMissingIterFunc   = errors.New("missing iter function")
)

// decodeFunc for parsing data and processing it in streaming mode
type decodeFunc func([]byte, func(rates euroLatestRates) error) error

type euroLatestRates struct {
	time  time.Time
	rates []euroExchangeRate
}

type euroExchangeRate struct {
	symbol label.Symbol
	rate   float64
}
