package gen

type Ccy struct {
	Number       int
	MinRateUnits int
	Name         string
	Symbol       string
	Sign         string
}

type CurrencyCodes struct {
	CcyTbl struct {
		CcyEntries []struct {
			CtryNm     string `xml:"CtryNm"`
			CcyNm      string `xml:"CcyNm"`
			Ccy        string `xml:"Ccy"`
			CcyNbr     int    `xml:"CcyNbr"`
			CcyMnrUnts string `xml:"CcyMnrUnts"`
		} `xml:"CcyNtry"`
	} `xml:"CcyTbl"`
}

type CurrencyNames struct {
	Main struct {
		EnVersion struct {
			Identity struct {
				Version struct {
					ClDRVersion string `json:"_cldrVersion"`
				} `json:"version"`
				Language  string `json:"language"`
				Territory string `json:"territory"`
			} `json:"identity"`
			Numbers struct {
				Currencies map[string]struct {
					DisplayName string `json:"displayName"`
					Symbol      string `json:"symbol"`
					Sign        string `json:"symbol-alt-narrow"`
				} `json:"currencies"`
			} `json:"numbers"`
		} `json:"en-001"`
	} `json:"main"`
}
