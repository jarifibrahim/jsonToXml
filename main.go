package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// jsonData is the json format that the tool understands. Please update update
// this type if your json data has different format.
type jsonData struct {
	Id        int
	FirstName string `json:"first_name" xml:"name>first"`
	LastName  string `json:"last_name" xml:"name>last"`
	City      string
	State     string
}

// IsEmpty returns true if all attributes of jsonData are empty (zero valued).
func (p *jsonData) IsEmpty() bool {
	return p.Id == 0 && len(p.FirstName) == 0 && len(p.LastName) == 0 &&
		len(p.City) == 0 && len(p.State) == 0
}

var (
	rootCmd = &cobra.Command{
		Use:   "jsonToXml",
		Short: "jsonToXml is a fast jsonToXml converter",
		Long: `jsonToXml is fast jsonToXml converter. The tool is capable of concurrenly fetching` +
			` multiple URLs and converting them to XML`,
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
	urls, output   string
	ErrUnknownJSON = errors.New("JSON is valid but it is not of type jsonData")
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&urls, "urls", "u", "",
		"Comma separated list of URLs to process.")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "./out",
		"Output directory to store xml files. One file per url will be created.")
}
func run() {
	if len(strings.TrimSpace(urls)) == 0 {
		log.Fatal("--urls flag cannot be empty.")
	}
	if len(strings.TrimSpace(output)) == 0 {
		log.Fatal("--output flag cannot be empty.")
	}
	log.Printf("Started Processing")

	start := time.Now()
	urlList := strings.Split(urls, ",")

	checkAndCreateDir()

	var eg errgroup.Group
	// Process all the urls in the flag.
	// TODO(ibrahim): In case the urlList is too large, this could cause
	// performance degradation. Consider throttling the go routines.
	for i, u := range urlList {
		u := strings.TrimSpace(u)
		resFile := filepath.Join(output, fmt.Sprintf("%d.xml", i))
		// Process concurrently.
		eg.Go(func() error {
			w := newDefaultWorker(resFile)
			defer w.close()
			err := w.fetchAndProcess(u)
			if err != nil {
				log.Printf("Failed processing url: %q err: %s", u, err)
				return nil
			}
			log.Printf("Finished processing url: %q output: %q", u, resFile)
			return nil
		})
	}
	// Wait for all go routines to complete.
	if err := eg.Wait(); err != nil {
		log.Fatal(err)
	}
	log.Printf("Processed %d urls in %s", len(urlList), time.Since(start))
}

func checkAndCreateDir() {
	dirExists, err := exists(output)
	if err != nil {
		log.Fatal(err)
	}
	if dirExists {
		return
	}
	if err = os.MkdirAll(output, 0700); err != nil {
		log.Fatalf("Error Creating Dir: %q", output)
	}
}

// Getter interface is used to mock the client in tests.
type Getter interface {
	Get(url string) (*http.Response, error)
}

// Worker encapsulates the client and writer. Multiple workers can run
// concurrently for fetch and process urls.
type worker struct {
	client Getter
	writer io.WriteCloser
}

func newDefaultWorker(output string) *worker {
	file, err := os.Create(output)
	if err != nil {
		log.Fatal(err)
	}
	return &worker{
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		writer: file,
	}

}
func (w *worker) close() error {
	return w.writer.Close()
}

// fetchAndProcess will fetch the provided URL. If the data is json, it will convert it to xml.
func (w *worker) fetchAndProcess(url string) error {
	resp, err := w.client.Get(url)
	if err != nil {
		return errors.Wrap(err, "get failed")

	}
	defer resp.Body.Close()
	header := resp.Header.Get("Content-Type")
	if header != "application/json" {
		return errors.Errorf("Invalid Content-Type header. Expected application/json, received %q",
			header)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return jsonToXml(body, w.writer)
}

// jsonToXml converts the json data in "data" to xml and writes it to the writer.
func jsonToXml(data []byte, w io.Writer) error {
	var p jsonData
	if err := json.Unmarshal(data, &p); err != nil {
		return errors.Wrap(err, "json.Unmarshal")
	}

	// Data could be valid json but not of type jsonData.
	if p.IsEmpty() {
		return ErrUnknownJSON
	}

	data, err := xml.MarshalIndent(p, " ", " ")
	if err != nil {
		return errors.Wrap(err, "xml.Marshal")
	}
	_, err = w.Write(data)
	return errors.Wrap(err, "write")

}

// exists checks if the "path" exists.
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
