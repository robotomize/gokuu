package cae

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/robotomize/gokuu/label"
)

func TestDataMatchingParseHTML(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		html     []byte
		expected struct {
			Date  string
			Rates map[label.Symbol]float64
		}
	}{
		{
			name: "test_set_0",
			expected: struct {
				Date  string
				Rates map[label.Symbol]float64
			}{
				Date: "12-08-2021",
				Rates: map[label.Symbol]float64{
					label.USD: 3.6725,
					label.ARS: 0.03783,
				},
			},
			html: []byte(`<!DOCTYPE html>
<html lang="en" dir="ltr"
      prefix="content: http://purl.org/rss/1.0/modules/content/  dc: http://purl.org/dc/terms/  foaf: http://xmlns.com/foaf/0.1/  og: http://ogp.me/ns#  rdfs: http://www.w3.org/2000/01/rdf-schema#  schema: http://schema.org/  sioc: http://rdfs.org/sioc/ns#  sioct: http://rdfs.org/sioc/types#  skos: http://www.w3.org/2004/02/skos/core#  xsd: http://www.w3.org/2001/XMLSchema# ">
<body class="path-fx-rates">
<main role="main" class="outer-wrapper">
    <div class="container">
        <div class="row">
            <div class="col-12">
                <!-- Add you custom twig html here -->
                <div class="pb-4">
                    <div class="dropdown" id="ratesDatePickerDropDown">
                        <a href="#" id="ratesDatePicker" data-toggle="dropdown" aria-haspopup="true"
                           aria-expanded="false">
                            <h3 class="m-0">
                <span class="badge badge-light d-inline-flex align-items-center py-0">
                  <span>Date12-08-2021</span>
                  <i class="icon-arrow-down-2" style="font-size: 8px; margin-left: 8px;"></i>
                </span>
                            </h3>
                        </a>
                        <div class="dropdown-menu p-0" aria-labelledby="ratesDatePicker">
                            <div class="h-100 calendar__placeholder">
                                <div id="ratesPageCalendar"></div>
                            </div>
                        </div>
                    </div>
                    <div class="text-muted"><small>Last updated <span class="dir-ltr">12 Aug 2021 6:00PM</span></small>
                    </div>
                </div>

                <table id="ratesDateTable" class="table table-striped table-bordered table-eibor text-center">
                    <thead>
                    <tr>
                        <th>Currency</th>
                        <th>Rate</th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr>
                        <td>US Dollar</td>
                        <td>3.672500</td>
                    </tr>
                    <tr>
                        <td>Argentine Peso</td>
                        <td>0.037830</td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</main>
</body>
</html>
`),
		},
		{
			name: "test_set_1",
			expected: struct {
				Date  string
				Rates map[label.Symbol]float64
			}{
				Date: "12-08-2021",
				Rates: map[label.Symbol]float64{
					label.AUD: 2.696006,
					label.BDT: 0.043328,
				},
			},
			html: []byte(`<!DOCTYPE html>
<html lang="en" dir="ltr"
      prefix="content: http://purl.org/rss/1.0/modules/content/  dc: http://purl.org/dc/terms/  foaf: http://xmlns.com/foaf/0.1/  og: http://ogp.me/ns#  rdfs: http://www.w3.org/2000/01/rdf-schema#  schema: http://schema.org/  sioc: http://rdfs.org/sioc/ns#  sioct: http://rdfs.org/sioc/types#  skos: http://www.w3.org/2004/02/skos/core#  xsd: http://www.w3.org/2001/XMLSchema# ">
<body class="path-fx-rates">
<main role="main" class="outer-wrapper">
    <div class="container">
        <div class="row">
            <div class="col-12">
                <!-- Add you custom twig html here -->
                <div class="pb-4">
                    <div class="dropdown" id="ratesDatePickerDropDown">
                        <a href="#" id="ratesDatePicker" data-toggle="dropdown" aria-haspopup="true"
                           aria-expanded="false">
                            <h3 class="m-0">
                <span class="badge badge-light d-inline-flex align-items-center py-0">
                  <span>Date12-08-2021</span>
                  <i class="icon-arrow-down-2" style="font-size: 8px; margin-left: 8px;"></i>
                </span>
                            </h3>
                        </a>
                        <div class="dropdown-menu p-0" aria-labelledby="ratesDatePicker">
                            <div class="h-100 calendar__placeholder">
                                <div id="ratesPageCalendar"></div>
                            </div>
                        </div>
                    </div>
                    <div class="text-muted"><small>Last updated <span class="dir-ltr">12 Aug 2021 6:00PM</span></small>
                    </div>
                </div>

                <table id="ratesDateTable" class="table table-striped table-bordered table-eibor text-center">
                    <thead>
                    <tr>
                        <th>Currency</th>
                        <th>Rate</th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr>
                        <td>Australian Dollar</td>
                        <td>2.696006</td>
                    </tr>
                    <tr>
                        <td>Bangladesh Taka</td>
                        <td>0.043328</td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</main>
</body>
</html>
`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := parseHTML(tc.html)
			if err != nil {
				t.Fatalf("parse HTML: %v", err)
			}

			if diff := cmp.Diff(tc.expected.Date, result.time.Format("02-01-2006")); diff != "" {
				t.Errorf("bad csv (-want, +got): %s", diff)
			}

			for _, r := range result.rates {
				v, ok := tc.expected.Rates[r.symbol]
				if !ok {
					t.Errorf("can not find symbol %s in result set", r.symbol.String())
				}

				if diff := cmp.Diff(v, r.rate); diff != "" {
					t.Errorf("bad value (-want, +got): %s", diff)
				}
			}
		})
	}
}

func TestAttrValidationParseHTML(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		html []byte
		err  error
	}{
		{
			name: "test_validate_time_0",
			err:  errParseAttrNotValid,
			html: []byte(`<!DOCTYPE html>
<html lang="en" dir="ltr"
      prefix="content: http://purl.org/rss/1.0/modules/content/  dc: http://purl.org/dc/terms/  foaf: http://xmlns.com/foaf/0.1/  og: http://ogp.me/ns#  rdfs: http://www.w3.org/2000/01/rdf-schema#  schema: http://schema.org/  sioc: http://rdfs.org/sioc/ns#  sioct: http://rdfs.org/sioc/types#  skos: http://www.w3.org/2004/02/skos/core#  xsd: http://www.w3.org/2001/XMLSchema# ">
<body class="path-fx-rates">
<main role="main" class="outer-wrapper">
    <div class="container">
        <div class="row">
            <div class="col-12">
                <!-- Add you custom twig html here -->
                <div class="pb-4">
                    <div class="dropdown" id="ratesDatePickerDropDown">
                        <a href="#" id="ratesDatePicker" data-toggle="dropdown" aria-haspopup="true"
                           aria-expanded="false">
                            <h3 class="m-0">
                <span class="badge badge-light d-inline-flex align-items-center py-0">
                  <span>Datefds</span>
                  <i class="icon-arrow-down-2" style="font-size: 8px; margin-left: 8px;"></i>
                </span>
                            </h3>
                        </a>
                        <div class="dropdown-menu p-0" aria-labelledby="ratesDatePicker">
                            <div class="h-100 calendar__placeholder">
                                <div id="ratesPageCalendar"></div>
                            </div>
                        </div>
                    </div>
                    <div class="text-muted"><small>Last updated <span class="dir-ltr">dgds</span></small>
                    </div>
                </div>

                <table id="ratesDateTable" class="table table-striped table-bordered table-eibor text-center">
                    <thead>
                    <tr>
                        <th>Currency</th>
                        <th>Rate</th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr>
                        <td>US Dollar</td>
                        <td>3.672500</td>
                    </tr>
                    <tr>
                        <td>Argentine Peso</td>
                        <td>0.037830</td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</main>
</body>
</html>
`),
		},
		{
			name: "test_validate_time_1",
			err:  errParseAttrNotValid,
			html: []byte(`<!DOCTYPE html>
<html lang="en" dir="ltr"
      prefix="content: http://purl.org/rss/1.0/modules/content/  dc: http://purl.org/dc/terms/  foaf: http://xmlns.com/foaf/0.1/  og: http://ogp.me/ns#  rdfs: http://www.w3.org/2000/01/rdf-schema#  schema: http://schema.org/  sioc: http://rdfs.org/sioc/ns#  sioct: http://rdfs.org/sioc/types#  skos: http://www.w3.org/2004/02/skos/core#  xsd: http://www.w3.org/2001/XMLSchema# ">
<body class="path-fx-rates">
<main role="main" class="outer-wrapper">
    <div class="container">
        <div class="row">
            <div class="col-12">
                <!-- Add you custom twig html here -->
                <div class="pb-4">
                    <div class="dropdown" id="ratesDatePickerDropDown">
                        <a href="#" id="ratesDatePicker" data-toggle="dropdown" aria-haspopup="true"
                           aria-expanded="false">
                            <h3 class="m-0">
                <span class="badge badge-light d-inline-flex align-items-center py-0">
                  <span>Dat</span>
                  <i class="icon-arrow-down-2" style="font-size: 8px; margin-left: 8px;"></i>
                </span>
                            </h3>
                        </a>
                        <div class="dropdown-menu p-0" aria-labelledby="ratesDatePicker">
                            <div class="h-100 calendar__placeholder">
                                <div id="ratesPageCalendar"></div>
                            </div>
                        </div>
                    </div>
                    <div class="text-muted"><small>Last updated <span class="dir-ltr">dgds</span></small>
                    </div>
                </div>

                <table id="ratesDateTable" class="table table-striped table-bordered table-eibor text-center">
                    <thead>
                    <tr>
                        <th>Currency</th>
                        <th>Rate</th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr>
                        <td>US Dollar</td>
                        <td>3.672500</td>
                    </tr>
                    <tr>
                        <td>Argentine Peso</td>
                        <td>0.037830</td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</main>
</body>
</html>
`),
		},
		{
			name: "test_validate_time_2",
			err:  errParseAttrNotValid,
			html: []byte(`<!DOCTYPE html>
<html lang="en" dir="ltr"
      prefix="content: http://purl.org/rss/1.0/modules/content/  dc: http://purl.org/dc/terms/  foaf: http://xmlns.com/foaf/0.1/  og: http://ogp.me/ns#  rdfs: http://www.w3.org/2000/01/rdf-schema#  schema: http://schema.org/  sioc: http://rdfs.org/sioc/ns#  sioct: http://rdfs.org/sioc/types#  skos: http://www.w3.org/2004/02/skos/core#  xsd: http://www.w3.org/2001/XMLSchema# ">
<body class="path-fx-rates">
<main role="main" class="outer-wrapper">
    <div class="container">
        <div class="row">
            <div class="col-12">
                <!-- Add you custom twig html here -->
                <div class="pb-4">
                    <div class="dropdown" id="ratesDatePickerDropDown">
                        <a href="#" id="ratesDatePicker" data-toggle="dropdown" aria-haspopup="true"
                           aria-expanded="false">
                            <h3 class="m-0">
                <span class="badge badge-light d-inline-flex align-items-center py-0">
                  <span></span>
                  <i class="icon-arrow-down-2" style="font-size: 8px; margin-left: 8px;"></i>
                </span>
                            </h3>
                        </a>
                        <div class="dropdown-menu p-0" aria-labelledby="ratesDatePicker">
                            <div class="h-100 calendar__placeholder">
                                <div id="ratesPageCalendar"></div>
                            </div>
                        </div>
                    </div>
                    <div class="text-muted"><small>Last updated <span class="dir-ltr">dgds</span></small>
                    </div>
                </div>

                <table id="ratesDateTable" class="table table-striped table-bordered table-eibor text-center">
                    <thead>
                    <tr>
                        <th>Currency</th>
                        <th>Rate</th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr>
                        <td>US Dollar</td>
                        <td>3.672500</td>
                    </tr>
                    <tr>
                        <td>Argentine Peso</td>
                        <td>0.037830</td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</main>
</body>
</html>
`),
		},
		{
			name: "test_validate_rate",
			err:  errParseAttrNotValid,
			html: []byte(`<!DOCTYPE html>
<html lang="en" dir="ltr"
      prefix="content: http://purl.org/rss/1.0/modules/content/  dc: http://purl.org/dc/terms/  foaf: http://xmlns.com/foaf/0.1/  og: http://ogp.me/ns#  rdfs: http://www.w3.org/2000/01/rdf-schema#  schema: http://schema.org/  sioc: http://rdfs.org/sioc/ns#  sioct: http://rdfs.org/sioc/types#  skos: http://www.w3.org/2004/02/skos/core#  xsd: http://www.w3.org/2001/XMLSchema# ">
<body class="path-fx-rates">
<main role="main" class="outer-wrapper">
    <div class="container">
        <div class="row">
            <div class="col-12">
                <!-- Add you custom twig html here -->
                <div class="pb-4">
                    <div class="dropdown" id="ratesDatePickerDropDown">
                        <a href="#" id="ratesDatePicker" data-toggle="dropdown" aria-haspopup="true"
                           aria-expanded="false">
                            <h3 class="m-0">
                <span class="badge badge-light d-inline-flex align-items-center py-0">
                  <span>Date12-08-2021</span>
                  <i class="icon-arrow-down-2" style="font-size: 8px; margin-left: 8px;"></i>
                </span>
                            </h3>
                        </a>
                        <div class="dropdown-menu p-0" aria-labelledby="ratesDatePicker">
                            <div class="h-100 calendar__placeholder">
                                <div id="ratesPageCalendar"></div>
                            </div>
                        </div>
                    </div>
                    <div class="text-muted"><small>Last updated <span class="dir-ltr">12 Aug 2021 6:00PM</span></small>
                    </div>
                </div>

                <table id="ratesDateTable" class="table table-striped table-bordered table-eibor text-center">
                    <thead>
                    <tr>
                        <th>Currency</th>
                        <th>Rate</th>
                    </tr>
                    </thead>
                    <tbody>
                    <tr>
                        <td>Australian Dollar</td>
                        <td>fdsds</td>
                    </tr>
                    <tr>
                        <td>Bangladesh Taka</td>
                        <td>0.043328</td>
                    </tr>
                    </tbody>
                </table>
            </div>
        </div>
    </div>
</main>
</body>
</html>
`),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := parseHTML(tc.html)
			if err != nil {
				if !errors.Is(err, tc.err) {
					diff := cmp.Diff(tc.err, err, cmpopts.EquateErrors())
					t.Errorf("mismatch (-want, +got):\n%s", diff)
				}

				return
			}

			if diff := cmp.Diff(tc.err, err, cmpopts.EquateErrors()); diff != "" {
				t.Errorf("mismatch (-want, +got):\n%s", diff)
			}
		})
	}
}
