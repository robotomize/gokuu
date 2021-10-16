# Gokuu
Library for getting the latest exchange rates

# Installation
```bash
go get github.com/robotomize/gokuu
```
# Features
* Currently supports 59 currency pairs
* Use your own data sources with provider.Source interface

# Usage

Get actual exchange rates
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/robotomize/gokuu"
	"github.com/robotomize/gokuu/label"
)

func main() {
	ctx := context.Background()
	g := gokuu.New(http.DefaultClient)

	latest := g.GetLatest(ctx)
	for _, r := range latest.Result {
		fmt.Printf("from: %s, to: %s, rate: %f, date: %v", r.From().Symbol, r.To().Symbol, r.Rate(), r.Time())
	}
}
```

Conversion from one currency to another
```go
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/robotomize/gokuu"
	"github.com/robotomize/gokuu/label"
)

func main() {
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
}
```
