package simba

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

func initHttpClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
	}
}

func GenerateBuzzWords() []string {
	buzzWords, exists := os.LookupEnv("APP_BUZZ_WORDS")
	defaultWords := []string{"care", "love", "support", "hug", "strong"}
	if !exists {
		log.Printf("Warning since APP_BUZZ_WORDS has not been specified, we are using default ones : %+v", defaultWords)
		return defaultWords
	}
	results := strings.Split(buzzWords, ",")
	if len(results) == 0 {
		log.Printf("Warning since APP_BUZZ_WORDS has been specified, however no word has been found, using default ones : %+v", defaultWords)
		return defaultWords
	}
	return results
}

func GenerateRandomIndexBuzzWord(words []string) int {
	lenWords := len(words)
	if lenWords == 0 {
		log.Printf("given words slice has len %d", lenWords)
		return -1
	}

	s1 := rand.NewSource(time.Now().UnixNano())

	min := 0
	max := lenWords
	return (rand.New(s1).Intn(max-min) + min)
}

//@params buzzWord: string
//@returns title:string, url:string, err:error
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
