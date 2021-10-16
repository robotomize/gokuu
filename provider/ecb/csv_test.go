package ecb

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/robotomize/gokuu/label"
)

func TestParseCSV(t *testing.T) {
	t.Parallel()

	decodeFn := decodeCSV()
	if err := decodeFn([]byte("nothing"), nil); err != nil {
		if !errors.Is(err, errMissingIterFunc) {
			t.Errorf("iterate throw error: %v", err)
		}
	}
}

func TestMarkDecodeCSV(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		err   error
		bytes []byte
	}{
		{
			name: "test_invalid_markup_0",
			err:  errDecodeToken,
			bytes: []byte(`Date; CZK, DKK, GBP,
18 June 2021, 1, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 2,
`),
		},
		{
			name: "test_invalid_markup_1",
			err:  errDecodeToken,
			bytes: []byte(`Date, CZK, DKK, GBP,18 June 2021, 0, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 2,
`),
		},
		{
			name: "test_invalid_markup_2",
			err:  errDecodeToken,
			bytes: []byte(`Date, CZK, DKK, GBP,
18 June 2021, 20, 7.4364, 0.85785,
19 June 2021; 25.0, 1.23, 34,
`),
		},
		{
			name: "test_invalid_markup_3",
			err:  errDecodeToken,
			bytes: []byte(`Date, USD, JPY, 
					18 June 2021, 1.1898, 131.12, 
				`),
		},
		{
			name: "test_invalid_header_field_0",
			err:  errAttributeNotValid,
			bytes: []byte(`, CZK, DKK, GBP,
18 June 2021, 2, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 3,
`),
		},
		{
			name: "test_invalid_header_field_1",
			err:  errAttributeNotValid,
			bytes: []byte(`;, CZK, DKK, GBP,
18 June 2021, 11, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 32,
`),
		},
		{
			name: "test_invalid_rate_0",
			err:  errAttributeNotValid,
			bytes: []byte(`Date, CZK, DKK, GBP,
18 June 2021, 0, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 0,
`),
		},
		{
			name: "test_invalid_rate_1",
			err:  errAttributeNotValid,
			bytes: []byte(`Date, CZK, DKK, GBP,
18 June 2021, -1, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, -1,
`),
		},
		{
			name: "test_invalid_rate_2",
			err:  errAttributeNotValid,
			bytes: []byte(`Date, CZK, DKK, GBP,
18 June 2021, dgsd, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 11,
`),
		},
		{
			name: "test_invalid_date_0",
			err:  errAttributeNotValid,
			bytes: []byte(`Date, CZK, DKK, GBP,
wrong type, 1, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 1,
`),
		},
		{
			name: "test_invalid_extra_symbol",
			bytes: []byte(`Date, GPK, DKK, GBP,
19 June 2021, 1, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 1,
`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decodeFn := decodeCSV()

			err := decodeFn(tc.bytes, func(rate euroLatestRates) error {
				return nil
			})

			if !errors.Is(err, tc.err) {
				diff := cmp.Diff(tc.err, err, cmpopts.EquateErrors())
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestDataMatchingDecodeCSV(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		data  map[string]map[label.Symbol]float64
		bytes []byte
	}{
		{
			name: "test_data_matching_0",
			data: map[string]map[label.Symbol]float64{
				"2021-06-18": {
					"USD": 1.1898,
					"JPY": 131.12,
					"BGN": 1.9558,
				},
			},
			bytes: []byte(`Date, USD, JPY, BGN,
18 June 2021, 1.1898, 131.12, 1.9558, 
`),
		},
		{
			name: "test_data_matching_1",
			data: map[string]map[label.Symbol]float64{
				"2021-06-18": {
					"CZK": 25.519,
					"DKK": 7.4364,
					"GBP": 0.85785,
				},
				"2021-06-19": {
					"CZK": 25.0,
					"DKK": 1.23,
					"GBP": 0.81,
				},
			},
			bytes: []byte(`Date, CZK, DKK, GBP,
18 June 2021, 25.519, 7.4364, 0.85785,
19 June 2021, 25.0, 1.23, 0.81,
`),
		},
		{
			name: "test_data_matching_2",
			data: map[string]map[label.Symbol]float64{
				"2021-06-18": {
					"USD": 1.1898,
					"JPY": 131.12,
				},
			},
			bytes: []byte("Date, USD, JPY\n 18 June 2021, 1.1898, 131.12"),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decodeFn := decodeCSV()

			var rates []euroLatestRates
			if err := decodeFn(tc.bytes, func(rate euroLatestRates) error {
				rates = append(rates, rate)
				return nil
			}); err != nil {
				t.Fatalf("iterate throw error: %v", err)
			}

			if diff := cmp.Diff(len(tc.data), len(rates)); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}

			for _, rate := range rates {
				dateTime := rate.time.Format("2006-01-02")
				testPairs, ok := tc.data[dateTime]
				if !ok {
					t.Errorf("unknown datetime in test dataset")
				}

				for _, pair := range rate.rates {
					rate, ok := testPairs[pair.symbol]
					if !ok {
						t.Errorf("unknown currency symbol in test dataset")
					}

					if diff := cmp.Diff(rate, pair.rate); diff != "" {
						t.Errorf("mismatch (-want, +got):\n%s", diff)
					}
				}
			}
		})
	}
}
