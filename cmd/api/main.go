package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/arthurschuelter/finances/cache"
	"github.com/joho/godotenv"
)

type Asset struct {
	Symbol      string  `json:"symbol"`
	Quantity    int     `json:"quantity"`
	TotalCost   float64 `json:"totalCost"`
	AverageCost float64 `json:"averageCost"`
}

type Quote struct {
	Symbol             string
	LongName           string
	RegularMarketPrice float64
	Currency           string
	LogoUrl            string
}

type QuoteResponse struct {
	Results []Quote `json:"results"`
}

var API_URL = "https://api.hgbrasil.com/finance/stock_price?key=%s&symbol=%s"

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system environment")
	}

	apiToken := os.Getenv("API_TOKEN")
	if apiToken == "" {
		log.Fatal("API_TOKEN is not set")
	}

	personalAssets, err := LoadAssets("assets.json")
	if err != nil {
		log.Fatal("Error loading assets: ", err)
	}

	symbols, err := GetSymbols(apiToken, personalAssets)
	if err != nil {
		log.Fatal("Error getting symbols: ", err)
	}
	prices := make(map[string]float64, len(symbols))
	for _, symbol := range symbols {
		prices[symbol.Name] = symbol.Price
	}

	for _, curAsset := range personalAssets {
		curAsset.AverageCost = curAsset.TotalCost / float64(curAsset.Quantity)
		price, ok := prices[curAsset.Symbol]
		if !ok {
			price, ok = prices[strings.ToUpper(curAsset.Symbol)]
		}
		if !ok {
			fmt.Printf("Couldn't find %s\n", curAsset.Symbol)
			continue
		}
		PrintSymbol(Quote{Symbol: strings.ToUpper(curAsset.Symbol), RegularMarketPrice: price}, curAsset)
	}

	s := http.Server{
		Addr:         ":8080",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  120 * time.Second,
		// Handler:      HelloHandler{},
	}

	err = s.ListenAndServe()
	if err != nil {
		if err != http.ErrServerClosed {
			panic(err)
		}
	}
}

func GetSymbols(apiToken string, assets []Asset) ([]cache.Symbol, error) {
	if symbols, fresh, err := cache.ReadFreshCache("cache.json", time.Now()); err != nil {
		log.Printf("Ignoring invalid cache: %v", err)
	} else if fresh {
		return symbols, nil
	}

	symbols := make([]cache.Symbol, 0, len(assets))
	client := &http.Client{Timeout: 10 * time.Second}
	for _, asset := range assets {
		url := fmt.Sprintf("https://brapi.dev/api/quote/%s?token=%s", asset.Symbol, apiToken)
		resp, err := client.Get(url)
		if err != nil {
			return nil, fmt.Errorf("fetch %s: %w", asset.Symbol, err)
		}
		var data QuoteResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&data)
		resp.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode %s response: %w", asset.Symbol, decodeErr)
		}
		if len(data.Results) == 0 {
			return nil, fmt.Errorf("API returned no results for %s", asset.Symbol)
		}
		quote := data.Results[0]
		symbols = append(symbols, cache.Symbol{Name: quote.Symbol, Price: quote.RegularMarketPrice})
	}
	if err := cache.WriteCache("cache.json", symbols); err != nil {
		return nil, fmt.Errorf("write cache: %w", err)
	}
	return symbols, nil
}

func PrintSymbol(quote Quote, asset Asset) {
	// fmt.Printf("Name: %s (%s)\n", quote.LongName, quote.Symbol)
	fmt.Printf("Symbol: %s\n", quote.Symbol)
	// fmt.Printf("Currency: %s\n", quote.Currency)
	fmt.Printf("Quantity: %.d\n", asset.Quantity)
	fmt.Printf("Average Price: %.2f\n", asset.AverageCost)
	fmt.Printf("Stock Price: %.2f\n", quote.RegularMarketPrice)
	fmt.Printf("Variation: R$ %.2f (%.4f)\n", quote.RegularMarketPrice*float64(asset.Quantity)-asset.TotalCost, quote.RegularMarketPrice/asset.AverageCost)
	// fmt.Printf("Logo URL: %s\n", quote.LogoUrl)

	fmt.Println()
}

// LoadAssets loads the user's portfolio from a JSON file.
func LoadAssets(path string) ([]Asset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var assets []Asset
	if err := json.Unmarshal(data, &assets); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return assets, nil
}
