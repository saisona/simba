package simba

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	_ "github.com/joho/godotenv/autoload"
)

func initHttpClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

//@returns title, url, error
func FetchRelatedGif(buzzWord string) (string, string, error) {
	client := initHttpClient()
	apiKey, exists := os.LookupEnv("APP_GIPHY_TOKEN")
	if !exists {
		return "", "", fmt.Errorf("APP_GIPHY_TOKEN env is not set")
	}

	url := fmt.Sprintf("https://api.giphy.com/v1/gifs/search?api_key=%s&q=%s&limit=1&offset=0&rating=g&lang=en", apiKey, buzzWord)
	res, err := client.Get(url)
	if err != nil {
		return "", "", err
	}

	jsonDecoder := json.NewDecoder(res.Body)
	var giphyResponse GiphyResponse
	if err := jsonDecoder.Decode(&giphyResponse); err != nil {
		return "", "", err
	}

	if len(giphyResponse.Data) == 0 {
		return "", "", fmt.Errorf("No data available")
	}

	return giphyResponse.Data[0].Id, giphyResponse.Data[0].Images.Original.Url, nil
}
