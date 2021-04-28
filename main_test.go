package main

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

type mockClient struct{}

// Ensure we don't break the interface.
var _ Getter = &mockClient{}

func (mc *mockClient) Get(url string) (*http.Response, error) {
	switch {
	case url == "valid": // Returns valid response.
		reader := bytes.NewReader([]byte(`{"first_name": "firstname", "last_name":"lastname"}`))
		body := ioutil.NopCloser(reader)
		header := make(http.Header)
		header["Content-Type"] = []string{"application/json"}
		return &http.Response{
			StatusCode: 200,
			Header:     header,
			Body:       body,
		}, nil
	case url == "invalid": // Returns invalid response.
		reader := bytes.NewReader([]byte(`"last_name":"lastname"}`))
		body := ioutil.NopCloser(reader)
		return &http.Response{
			StatusCode: 200,
			Body:       body,
		}, nil
	case url == "unknown": // Return unknown response.
		reader := bytes.NewReader([]byte(`{"foo":"bar"}`))
		body := ioutil.NopCloser(reader)
		header := make(http.Header)
		header["Content-Type"] = []string{"application/json"}
		return &http.Response{
			StatusCode: 200,
			Body:       body,
		}, nil
	default:
		return nil, errors.New("Unknown url")
	}
}

type mockWriter struct {
	io.Writer
}

var _ io.WriteCloser = mockWriter{}

func (mockWriter) Close() error {
	return nil
}

func TestWorker(t *testing.T) {
	tt := []struct {
		name      string
		output    string
		shouldErr bool
	}{
		{"valid", " <jsonData>\n  <Id>0</Id>\n  <name>\n   <first>firstname</first>\n   " +
			"<last>lastname</last>\n  </name>\n  <City></City>\n  <State></State>\n </jsonData>",
			false},
		{"invalid", "", true},
		{"unknown", "", true},
	}

	for _, ti := range tt {
		t.Run(ti.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &worker{
				client: new(mockClient),
				writer: mockWriter{&buf},
			}
			err := w.fetchAndProcess(ti.name)
			if ti.shouldErr {
				require.Error(t, err)
				require.Zero(t, buf.Bytes())
				return
			}
			require.NoError(t, err)
			require.Equal(t, ti.output, string(buf.Bytes()))
		})
	}

}

func TestJsonRespToXml(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		jdata := []byte(`{"id": 10, "first_name": "firstname", "last_name":"lastname"}`)
		buf := &bytes.Buffer{}
		require.NoError(t, jsonToXml(jdata, buf))
		res := ` <jsonData>
  <Id>10</Id>
  <name>
   <first>firstname</first>
   <last>lastname</last>
  </name>
  <City></City>
  <State></State>
 </jsonData>`
		require.Equal(t, res, string(buf.Bytes()))
	})
	t.Run("valid json but not jsonData", func(t *testing.T) {
		jdata := []byte(`{"foo":"lastname"}`)
		buf := &bytes.Buffer{}
		err := jsonToXml(jdata, buf)
		require.Error(t, err)
		require.ErrorIs(t, ErrUnknownJSON, err)
		require.Empty(t, buf)
	})
	t.Run("invalid json", func(t *testing.T) {
		jdata := []byte(`{"foo":"lastname"`)
		buf := &bytes.Buffer{}
		err := jsonToXml(jdata, buf)
		require.NotErrorIs(t, ErrUnknownJSON, err)
		require.Empty(t, buf)
	})
}

func TestDataIsEmpty(t *testing.T) {
	var p jsonData
	require.True(t, p.IsEmpty())
	p.FirstName = "foo"
	require.False(t, p.IsEmpty())
}

// goos: linux
// goarch: amd64
// pkg: jsonToXml
// BenchmarkWorkers
// BenchmarkWorkers-8   	  287504	      4744 ns/op
// PASS
// ok  	jsonToXml	2.314s
func BenchmarkWorkers(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf bytes.Buffer
			w := &worker{
				client: new(mockClient),
				writer: mockWriter{&buf},
			}
			err := w.fetchAndProcess("valid")
			if err != nil {
				panic(err)
			}
		}
	})
}
