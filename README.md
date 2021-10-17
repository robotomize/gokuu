# Gokuu

[![Go Report](https://goreportcard.com/badge/github.com/robotomize/gokuu)](https://goreportcard.com/report/github.com/robotomize/gokuu)

Gokuu is a library for getting up-to-date exchange rates and converting them on Go. Gokuu is a controller and plug-in data sources that work with a specific financial institution. 

Right now, Gokuu can retrieve data from the European central bank, the Russian central bank and the central bank of the United Arab Emirates. Altogether, these sources cover exchange rates for 59 currencies

## Installation
```bash
go get github.com/robotomize/gokuu
```
## Features
* Currently supports 59 currency pairs
* Use your own data sources with provider.Source interface

## Usage

If you want to get all current exchange rates, you can use the following GetLatest method.
When you match currency pairs from different providers, Gokuu by default saves the data from the provider who gave it faster. 

```go
ctx := context.Background()
g := gokuu.New(http.DefaultClient)

latest := g.GetLatest(ctx)
for _, r := range latest.Result {
	fmt.Printf("from: %s, to: %s, rate: %f, date: %v", 
		r.From().Symbol, r.To().Symbol, r.Rate(), r.Time(),
	)
}
```

With options you can change the strategy for merging data from multiple providers. For example, a merger based on priorities

```go
g := gokuu.New(http.DefaultClient, gokuu.WithPriorityMergeStrategy())
```

Calculate the arithmetic average of the currency pair from all providers

```go
g := gokuu.New(http.DefaultClient, gokuu.WithAverageMergeStrategy())
```

You can also use the conversion function
```go
ctx := context.Background()
g := gokuu.New(http.DefaultClient)

conv, err := g.Convert(
	ctx, gokuu.ConvOpt{
		From:  label.USD,
		To:    label.RUB,
		Value: 10,
	},
)
if err != nil {
	log.Fatalln(err)
}
fmt.Println(conv)
```

Use your caching function for better performance. For example like this
```go
g := gokuu.New(http.DefaultClient, gokuu.WithAverageMergeStrategy())
latest := g.GetLatest(ctx)
conv, err := g.Convert(
	ctx, gokuu.ConvOpt{
		From:  label.USD,
		To:    label.RUB,
		Value: 10,
		CacheFn: func(ctx context.Context) gokuu.LatestResponse {
			return latest
		},
	},
)
if err != nil {
	log.Fatalln(err)
}
fmt.Println(conv)
```

You can also use the helper functions from the package github.com/robotomize/gokuu/label
```go
label.GetSymbols()
label.GetCountries()
label.GetCurrencies()
label.GetCountriesUsingCurrency("currency-symbol")
label.GetCurrenciesUsedCountry("countryname")
```

## License

SOD is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
