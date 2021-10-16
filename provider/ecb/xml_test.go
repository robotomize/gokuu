package ecb

import (
	"errors"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/robotomize/gokuu/label"
)

func TestMain(m *testing.M) {
	m.Run()
	os.Exit(0)
}

func TestDecodeXML(t *testing.T) {
	t.Parallel()

	decodeFn := decodeXML()
	if err := decodeFn([]byte("nothing"), nil); err != nil {
		if !errors.Is(err, errMissingIterFunc) {
			t.Errorf("iterate throw error: %v", err)
		}
	}
}

func TestDataMatchingDecodeXML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected map[string]map[label.Symbol]float64
		bytes    []byte
	}{
		{
			name: "test_single_set",
			expected: map[string]map[label.Symbol]float64{
				"2021-06-18": {
					"USD": 1.1898,
					"JPY": 131.12,
					"BGN": 1.9558,
				},
			},
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube currency="USD" rate="1.1898"/>
            <Cube currency="JPY" rate="131.12"/>
            <Cube currency="BGN" rate="1.9558"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
		{
			name: "test_multiple_set",
			expected: map[string]map[label.Symbol]float64{
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
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube currency="CZK" rate="25.519"/>
            <Cube currency="DKK" rate="7.4364"/>
            <Cube currency="GBP" rate="0.85785"/>
        </Cube>
        <Cube time="2021-06-19">
            <Cube currency="CZK" rate="25.0"/>
            <Cube currency="DKK" rate="1.23"/>
            <Cube currency="GBP" rate="0.81"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decodeFn := decodeXML()

			var rates []euroLatestRates
			if err := decodeFn(tc.bytes, func(rate euroLatestRates) error {
				rates = append(rates, rate)
				return nil
			}); err != nil {
				t.Errorf("iterate throw error: %v", err)
			}

			if len(rates) != len(tc.expected) {
				diff := cmp.Diff(len(tc.expected), len(rates))
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}

			for _, rate := range rates {
				dateTime := rate.time.Format("2006-01-02")
				testPairs, ok := tc.expected[dateTime]
				if !ok {
					t.Errorf("unknown datetime in test dataset")
				}

				if len(rate.rates) != len(testPairs) {
					diff := cmp.Diff(len(tc.expected), len(rate.rates))
					t.Errorf("mismatch (-want, +got):\n%s", diff)
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

func TestAttrValidationAttrDecodeXML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		err   error
		bytes []byte
	}{
		{
			name: "test_invalid_rate_0",
			err:  errAttributeNotValid,
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01"
                 xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube currency="USD" rate="1.1898"/>
            <Cube currency="ZAR" rate="0.0"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
		{
			name: "test_invalid_rate_1",
			err:  errAttributeNotValid,
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01"
                 xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube currency="USD" rate="1.1898"/>
            <Cube currency="ZAR" rate="-1"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
		{
			name: "test_invalid_rate_2",
			err:  errAttributeNotValid,
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01"
                 xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube currency="USD" rate="1.1898"/>
            <Cube currency="ZAR" rate="wrong type"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
		{
			name: "test_value_type_1",
			err:  errAttributeNotValid,
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01"
                 xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="wrong type">
            <Cube currency="USD" rate="1.1898"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decodeFn := decodeXML()

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

func TestMarkupDecodeXML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		err   error
		bytes []byte
	}{
		{
			name: "test_extra_attr",
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01"
                 xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2017-01-02">
            <Cube time="2017-01-02" currency="USD" rate="1.1898"/>
            <Cube currency="JPY" rate="131.12" time="2017-01-02"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
		{
			name: "test_invalid_syntax_0",
			err:  errDecodeToken,
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube currency="USD" rate="1.1898"/>
        </Cube>
    </Cube
</gesmes:Envelope>`),
		},
		{
			name: "test_invalid_syntax_1",
			err:  errDecodeToken,
			bytes: []byte(` <Cube>
        <Cube time="2021-06-18">
            <Cube currency="USD" rate="1.1898"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
		{
			name: "test_invalid_syntax_2",
			err:  errDecodeToken,
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube currency="USD" rate="1.1898">
            <Cube currency="JPY" rate="131.12"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
		{
			name: "test_unknown_attr",
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube weather="sunny"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
		{
			name: "test_unknown_symbol",
			bytes: []byte(`<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
    <gesmes:subject>Reference rates</gesmes:subject>
    <gesmes:Sender>
        <gesmes:name>European Central Bank</gesmes:name>
    </gesmes:Sender>
    <Cube>
        <Cube time="2021-06-18">
            <Cube currency="USD" rate="1.1898"/>
            <Cube currency="JPYDF" rate="131.12"/>
        </Cube>
    </Cube>
</gesmes:Envelope>`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			decodeFn := decodeXML()

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
