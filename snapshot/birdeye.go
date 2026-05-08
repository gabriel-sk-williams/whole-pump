package snapshot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type BirdeyePriceResponse struct {
	Data struct {
		Value float64 `json:"value"`
	} `json:"data"`
	Success bool `json:"success"`
}

func GetPrice(mint string, timestamp int64) (float64, error) {
	apiKey := os.Getenv("BIRDEYE_API_KEY")
	if apiKey == "" {
		return 0, fmt.Errorf("BIRDEYE_API_KEY not set")
	}

	url := fmt.Sprintf("https://public-api.birdeye.so/defi/historical_price_unix?address=%s&unixtime=%d", mint, timestamp)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("x-chain", "solana")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	// fmt.Println(string(body))

	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	var result BirdeyePriceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success || result.Data.Value == 0 {
		return 0, fmt.Errorf("no price data for mint %s at timestamp %d", mint, timestamp)
	}

	return result.Data.Value, nil
}
