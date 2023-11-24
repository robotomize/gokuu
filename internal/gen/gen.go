package gen

import (
	"bytes"
	"embed"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/hashicorp/go-multierror"
	"github.com/robotomize/gokuu/internal/hashio"
	"github.com/robotomize/gokuu/internal/strutil"
)

const (
	AssetsCurrencyCodesFile = "currency_codes.xml"
	AssetsCurrencyNamesFile = "currency_names.json"
	CurrencyGenFileName     = "currency"
	SymbolGenFileName       = "symbol"
	CountryGenFileName      = "country"
	NameGenFileName         = "name"
	FuncGenFileName         = "func"
)

const SuffixGenFileName = "_gen.go"

const (
	symbolTemplate     = "symbol.tmpl"
	countryTemplate    = "country.tmpl"
	currenciesTemplate = "currency.tmpl"
	nameTemplate       = "name.tmpl"
	funcTemplate       = "func.tmpl"
)

var ErrHashingContentEqual = errors.New("hash of the generated file is equivalent to the previous version")

var defaultHashTypeFunc = hashio.MD5()

var (
	//go:embed templates/*.tmpl
	templates embed.FS
	//go:embed assets
	assets embed.FS
)

type AssetsMapFunc func(b []byte, filename string) error

func ReadAssets(path string) func(AssetsMapFunc) error {
	return func(mapFunc AssetsMapFunc) error {
		entries, err := assets.ReadDir(path)
		if err != nil {
			return fmt.Errorf("read dir: %w", err)
		}

		for _, entry := range entries {
			b, err := assets.ReadFile(filepath.Join(path, entry.Name()))
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			if err := mapFunc(b, entry.Name()); err != nil {
				return fmt.Errorf("call mapFunc: %w", err)
			}
		}

		return nil
	}
}

func Funcs() template.FuncMap {
	return template.FuncMap{
		"removeBrackets":    strutil.RemoveContentIntoBrackets,
		"toCamelCase":       strutil.CamelCase,
		"removeNonAlphaNum": strutil.RemoveNonAlphaNum,
	}
}

func Template() *template.Template {
	tmpl := template.New("templates").Funcs(Funcs())
	tmpl = template.Must(tmpl.ParseFS(templates, "templates/*.tmpl"))
	return tmpl
}

func Generate(pathTo string, hasherFunc func() hash.Hash) error {
	var (
		codes CurrencyCodes
		names CurrencyNames
	)

	var multiErr multierror.Group

	if hasherFunc == nil {
		hasherFunc = defaultHashTypeFunc
	}

	iterFunc := ReadAssets("assets")

	if err := iterFunc(func(b []byte, filename string) error {
		switch filename {
		case AssetsCurrencyCodesFile:
			if err := xml.Unmarshal(b, &codes); err != nil {
				return fmt.Errorf("xml unmarshal: %w", err)
			}
		case AssetsCurrencyNamesFile:
			if err := json.Unmarshal(b, &names); err != nil {
				return fmt.Errorf("json unmarshal: %w", err)
			}
		default:
		}

		return nil
	}); err != nil {
		return fmt.Errorf("iterate func: %w", err)
	}

	currencies := make(map[string]Ccy, len(codes.CcyTbl.CcyEntries))
	countries := make(map[string]string, len(codes.CcyTbl.CcyEntries))
	countrySymbols := make(map[string][]string, len(codes.CcyTbl.CcyEntries))
	symCountries := make(map[string][]string, len(codes.CcyTbl.CcyEntries))

	for _, entry := range codes.CcyTbl.CcyEntries {
		if len(entry.Ccy) == 0 {
			continue
		}

		cntry := entry.CtryNm
		cntry = strutil.RemoveNonAlphaNum(cntry)
		cntry = strutil.RemoveContentIntoBrackets(cntry)
		cntry = strutil.RemoveExtraSpaces(cntry)
		cntry = strutil.CamelCase(cntry)

		countrySymbols[cntry] = append(countrySymbols[cntry], entry.Ccy)
		symCountries[entry.Ccy] = append(symCountries[entry.Ccy], cntry)

		if _, ok := countries[cntry]; !ok {
			countries[cntry] = entry.CtryNm
		}

		units, err := strconv.Atoi(entry.CcyMnrUnts)
		if err != nil {
			var numErr *strconv.NumError
			if errors.As(err, &numErr) {
				units = 0
			}
		}

		ccy, ok := names.Main.EnVersion.Numbers.Currencies[entry.Ccy]
		if !ok {
			continue
		}

		sign, ok := names.Main.EnVersion.Numbers.Currencies[entry.Ccy]
		if !ok {
			continue
		}

		if _, ok := currencies[entry.Ccy]; !ok {
			currencies[entry.Ccy] = Ccy{
				Number:       entry.CcyNbr,
				MinRateUnits: units,
				Name:         ccy.DisplayName,
				Symbol:       entry.Ccy,
				Sign:         sign.Sign,
			}
		}
	}

	multiErr.Go(func() error {
		fileName := filepath.Join(pathTo, fmt.Sprintf("%s%s", SymbolGenFileName, SuffixGenFileName))

		oldHash, err := hashingFile(fileName, hasherFunc)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("hashingFile: %w", err)
			}
		}

		w := newWriter(bytes.NewBuffer(make([]byte, 0, 512)), Template())

		if err := w.generate(symbolTemplate, struct {
			Currencies interface{}
		}{
			Currencies: currencies,
		}); err != nil {
			return fmt.Errorf("symbol tmpl generate error: %w", err)
		}

		if len(oldHash) != 0 {
			newHash, err := hashio.ReadAll(bytes.NewReader(w.buf.Bytes()), hasherFunc())
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}
			// @TODO hashed is broken
			if bytes.Equal(oldHash, newHash) {
				return fmt.Errorf("warning: %w, file: %s", ErrHashingContentEqual, fileName)
			}
		}

		if err := w.flush(fileName); err != nil {
			return fmt.Errorf("save the generated template to a file: %w", err)
		}

		return nil
	})

	multiErr.Go(func() error {
		fileName := filepath.Join(pathTo, fmt.Sprintf("%s%s", CurrencyGenFileName, SuffixGenFileName))

		oldHash, err := hashingFile(fileName, hasherFunc)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("hashingFile: %w", err)
			}
		}

		w := newWriter(bytes.NewBuffer(make([]byte, 0, 512)), Template())

		if err := w.generate(currenciesTemplate, struct {
			Currencies interface{}
		}{
			Currencies: currencies,
		}); err != nil {
			return fmt.Errorf("symbol tmpl generate error: %w", err)
		}

		if len(oldHash) != 0 {
			newHash, err := hashio.ReadAll(bytes.NewReader(w.buf.Bytes()), hasherFunc())
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			if bytes.Equal(oldHash, newHash) {
				return fmt.Errorf("warning: %w, file: %s", ErrHashingContentEqual, fileName)
			}
		}

		if err := w.flush(fileName); err != nil {
			return fmt.Errorf("save the generated template to a file: %w", err)
		}

		return nil
	})

	multiErr.Go(func() error {
		fileName := filepath.Join(pathTo, fmt.Sprintf("%s%s", CountryGenFileName, SuffixGenFileName))

		oldHash, err := hashingFile(fileName, hasherFunc)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("hashingFile: %w", err)
			}
		}

		w := newWriter(bytes.NewBuffer(make([]byte, 0, 512)), Template())

		if err := w.generate(countryTemplate, struct {
			Countries       interface{}
			CountrySymbols  map[string][]string
			SymbolCountries map[string][]string
		}{
			Countries:       countries,
			CountrySymbols:  countrySymbols,
			SymbolCountries: symCountries,
		}); err != nil {
			return fmt.Errorf("country tmpl generate error: %w", err)
		}

		if len(oldHash) != 0 {
			newHash, err := hashio.ReadAll(bytes.NewReader(w.buf.Bytes()), hasherFunc())
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			if bytes.Equal(oldHash, newHash) {
				return fmt.Errorf("warning: %w, file: %s", ErrHashingContentEqual, fileName)
			}
		}

		if err := w.flush(fileName); err != nil {
			return fmt.Errorf("save the generated template to a file: %w", err)
		}

		return nil
	})

	multiErr.Go(func() error {
		fileName := filepath.Join(pathTo, fmt.Sprintf("%s%s", NameGenFileName, SuffixGenFileName))

		oldHash, err := hashingFile(fileName, hasherFunc)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("hashingFile: %w", err)
			}
		}

		w := newWriter(bytes.NewBuffer(make([]byte, 0, 512)), Template())

		if err := w.generate(nameTemplate, struct {
			Currencies interface{}
		}{
			Currencies: currencies,
		}); err != nil {
			return fmt.Errorf("symbol tmpl generate error: %w", err)
		}

		if len(oldHash) != 0 {
			newHash, err := hashio.ReadAll(bytes.NewReader(w.buf.Bytes()), hasherFunc())
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			if bytes.Equal(oldHash, newHash) {
				return fmt.Errorf("warning: %w, file: %s", ErrHashingContentEqual, fileName)
			}
		}

		if err := w.flush(fileName); err != nil {
			return fmt.Errorf("save the generated template to a file: %w", err)
		}

		return nil
	})

	multiErr.Go(func() error {
		fileName := filepath.Join(pathTo, fmt.Sprintf("%s%s", FuncGenFileName, SuffixGenFileName))

		oldHash, err := hashingFile(fileName, hasherFunc)
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				return fmt.Errorf("hashingFile: %w", err)
			}
		}

		w := newWriter(bytes.NewBuffer(make([]byte, 0, 512)), Template())

		if err := w.generate(funcTemplate, nil); err != nil {
			return fmt.Errorf("symbol tmpl generate error: %w", err)
		}

		if len(oldHash) != 0 {
			newHash, err := hashio.ReadAll(bytes.NewReader(w.buf.Bytes()), hasherFunc())
			if err != nil {
				return fmt.Errorf("read file: %w", err)
			}

			if bytes.Equal(oldHash, newHash) {
				return fmt.Errorf("warning: %w, file: %s", ErrHashingContentEqual, fileName)
			}
		}

		if err := w.flush(fileName); err != nil {
			return fmt.Errorf("save the generated template to a file: %w", err)
		}

		return nil
	})

	if err := multiErr.Wait(); err != nil {
		return fmt.Errorf("generate: %w", err)
	}

	return nil
}

func newWriter(buf *bytes.Buffer, t *template.Template) *writer {
	return &writer{buf: buf, t: t}
}

func hashingFile(fileName string, hasherFunc func() hash.Hash) ([]byte, error) {
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	b, err := hashio.ReadAll(file, hasherFunc())
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	return b, nil
}

type writer struct {
	buf *bytes.Buffer
	t   *template.Template
}

func (w *writer) flush(fileName string) error {
	if err := os.WriteFile(fileName, w.buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	return nil
}

func (w *writer) generate(tmplName string, data interface{}) error {
	if err := w.t.ExecuteTemplate(w.buf, tmplName, data); err != nil {
		return fmt.Errorf("execute template: %w", err)
	}

	return nil
}
