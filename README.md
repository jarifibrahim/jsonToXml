## jsonToXml

jsonToXml is a command line utility tool that can be used to fetch json from a URL and conver that json into xml. The tool currently support only the following json format.

```
type jsonData {
  id 			int
  first_name 	string
  last_name 	string
  city 			string
  state 		string
}
```

To use a different json object, please edit the jsonData type in main.go.

## Usage
```
$ go run main.go --help
jsonToXml is fast jsonToXml converter. The tool is capable of concurrenly fetching multiple URLs and converting them to XML.

Usage:
  jsonToXml [flags]

Flags:
  -h, --help            help for jsonToXml
  -o, --output string   Output directory to store xml files. One per url. (default "./out")
  -u, --urls string     List of URLs to process.
```

## Example
Step 1: Start a local http server
```
cd sample-data
sudo python -m SimpleHTTPServer 80
```
Step 2: Run the tool
```
go run main.go --urls "http://localhost/sample1.json,http://localhost/sample2.json"

or

go build .
./jsonToXml urls "http://localhost/sample.json,http://localhost/sample1.json, http://localhost/sample2.json, http://localhost/invalid.json"
```
