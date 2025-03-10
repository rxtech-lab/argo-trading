package main

import (
	"fmt"

	datasource "github.com/sirily11/argo-trading-go/src/engine/data_source"
)

func main() {
	csvIterator := datasource.CSVIterator{
		FilePath: "/Users/qiweili/Desktop/argo-trading/data/AAPL_2025-01-01_2025-01-31.csv",
	}

	for marketData := range csvIterator.Iterator() {
		fmt.Println(marketData)
	}
}
