package rcb

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/robotomize/gokuu/label"
)

func TestDataMatchingDecodeXML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		expected struct {
			Date  string
			Rates map[label.Symbol]float64
		}
		bytes []byte
	}{
		{
			name: "test_set_0",
			expected: struct {
				Date  string
				Rates map[label.Symbol]float64
			}{
				Date: "30.07.2021",
				Rates: map[label.Symbol]float64{
					label.AUD: 54.1609,
					label.AZN: 43.0785,
					label.GBP: 102.1811,
				},
			},
			bytes: []byte(`<ValCurs Date="30.07.2021" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>AUD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>AZN</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>GBP</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>102,1811</Value>
    </Valute>
</ValCurs>`),
		},
		{
			name: "test_set_0",
			expected: struct {
				Date  string
				Rates map[label.Symbol]float64
			}{
				Date: "30.08.2021",
				Rates: map[label.Symbol]float64{
					label.USD: 54.1609,
					label.EUR: 43.0785,
					label.JPY: 102.1811,
				},
			},
			bytes: []byte(`<ValCurs Date="30.08.2021" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>USD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>EUR</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>JPY</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>102,1811</Value>
    </Valute>
</ValCurs>`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			result, err := decodeXML(tc.bytes)
			if err != nil {
				t.Fatalf("decoding XML: %v", err)
			}

			if diff := cmp.Diff(len(tc.expected.Rates), len(result.rates)); diff != "" {
				t.Errorf("bad csv (-want, +got): %s", diff)
			}

			dateTime := result.time.Format("02.01.2006")
			if diff := cmp.Diff(tc.expected.Date, dateTime); diff != "" {
				t.Errorf("bad csv (-want, +got): %s", diff)
			}

			for _, r := range result.rates {
				v, ok := tc.expected.Rates[r.symbol]
				if !ok {
					t.Errorf("can not find symbol %s in result set", r.symbol.String())
				}

				if diff := cmp.Diff(v, r.rate); diff != "" {
					t.Errorf("bad csv (-want, +got): %s", diff)
				}
			}
		})
	}
}

func TestAttrValidationDecodeXML(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		err   error
		bytes []byte
	}{
		{
			name: "test_invalid_rate_0",
			err:  errAttributeNotValid,
			bytes: []byte(`<ValCurs Date="30.08.2021" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>USD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>EUR</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>JPY</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>0.0</Value>
    </Valute>
</ValCurs>`),
		},
		{
			name: "test_invalid_rate_1",
			err:  errAttributeNotValid,
			bytes: []byte(`<ValCurs Date="30.08.2021" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>USD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>EUR</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>JPY</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>-1</Value>
    </Valute>
</ValCurs>`),
		},
		{
			name: "test_value_type_2",
			err:  errAttributeNotValid,
			bytes: []byte(`<ValCurs Date="30.08.2021" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>USD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>EUR</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>JPY</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>hello world</Value>
    </Valute>
</ValCurs>`),
		},
		{
			name: "test_value_type_0",
			err:  errAttributeNotValid,
			bytes: []byte(`<ValCurs Date="hello world" name="Foreign Currency Market">
<Valute ID="R01010">
        <NumCode>036</NumCode>
        <CharCode>USD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
    <Valute ID="R01020A">
        <NumCode>944</NumCode>
        <CharCode>EUR</CharCode>
        <Nominal>1</Nominal>
        <Name>Азербайджанский манат</Name>
        <Value>43,0785</Value>
    </Valute>
    <Valute ID="R01035">
        <NumCode>826</NumCode>
        <CharCode>JPY</CharCode>
        <Nominal>1</Nominal>
        <Name>Фунт стерлингов Соединенного королевства</Name>
        <Value>hello world</Value>
    </Valute>
</ValCurs>`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := decodeXML(tc.bytes)

			if !errors.Is(err, tc.err) {
				diff := cmp.Diff(tc.err, err, cmpopts.EquateErrors())
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}

func TestXMLMarkupDecodeXML(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		err   error
		bytes []byte
	}{
		{
			name: "test_extra_attr",
			bytes: []byte(`<ValCurs Date="30.07.1996" name="Foreign Currency Market">
<Valute ID="R01010" time="2017-01-02">
        <NumCode>036</NumCode>
        <CharCode>USD</CharCode>
        <Nominal>1</Nominal>
        <Name>Австралийский доллар</Name>
        <Value>54,1609</Value>
    </Valute>
</ValCurs>`),
		},
		{
			name: "test_invalid_syntax_0",
			err:  errDecodeToken,
			bytes: []byte(`<ValCurs Date="30.07.1996" name="Foreign Currency Market">
	<Valute ID="R01010" time="2017-01-02">
	       <NumCode>036</NumCode>
	       <CharCode>USD</CharCode>
	       <Nominal>1</Nominal>
	       <Name>Австралийский доллар</Name>
	       <Value>54,1609</Value>
	   </Valute
	</ValCurs>`),
		},
		{
			name: "test_invalid_syntax_1",
			err:  errDecodeToken,
			bytes: []byte(`<ValCurs Date="30.07.1996" name="Foreign Currency Market">
	<Valute ID="R01010" time="2017-01-02">
	       <NumCode>036</NumCode>
	       <CharCode>USD</CharCode>
	       <Nominal>1</Nominal>
	       <Name>Австралийский доллар</Name>
	       <Value>54,1609</Value>
	   </Valute>`),
		},
		{
			name: "test_unknown_attr_field",
			bytes: []byte(`<ValCurs Date="30.07.1996" name="Foreign Currency Market">
	<Valute ID="R01010" time="2017-01-02" weather="sunny">
	       <NumCode>036</NumCode>
	       <Num>036</Num>
	       <CharCode>USD</CharCode>
	       <Nominal>1</Nominal>
	       <Name>Австралийский доллар</Name>
	       <Value>54,1609</Value>
	   </Valute>
	</ValCurs>`),
		},
		{
			name: "test_unknown_symbol",
			bytes: []byte(`<ValCurs Date="30.07.1996" name="Foreign Currency Market">
	<Valute ID="R01010" time="2017-01-02" weather="sunny">
	       <NumCode>036</NumCode>
	       <Num>036</Num>
	       <CharCode>USDT</CharCode>
	       <Nominal>1</Nominal>
	       <Name>Австралийский доллар</Name>
	       <Value>54,1609</Value>
	   </Valute>
	</ValCurs>`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := decodeXML(tc.bytes)

			if !errors.Is(err, tc.err) {
				diff := cmp.Diff(tc.err, err, cmpopts.EquateErrors())
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
