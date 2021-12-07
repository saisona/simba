package simba

import (
	"errors"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
)

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string, overide bool) error {

	if _, err := os.Stat(filepath); errors.Is(err, fs.ErrNotExist) || overide {
		// Get the data
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// Create the file
		out, err := os.Create(filepath)
		if err != nil {
			return err
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}

		return nil
	} else {
		//File exists then not need to re download it
		log.Printf("Warning %s already exists ! overide=%v", filepath, overide)
		return nil
	}
}
