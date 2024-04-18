package plugins

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"

	extism "github.com/extism/go-sdk"
)

type HTTPOptions struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

var (
	HTTPRequest = extism.NewHostFunctionWithStack("httpRequest", func(ctx context.Context, p *extism.CurrentPlugin, stack []uint64) {
		// {
		//	"method": "GET",
		//	"url": "http://localhost:8080",
		//}
		opts, err := p.ReadString(stack[0])
		if err != nil {
			panic(err)
		}

		var httpOpts HTTPOptions
		if err := json.Unmarshal([]byte(opts), &httpOpts); err != nil {
			panic(err)
		}

		// body as JSON
		body, err := p.ReadString(stack[1])
		if err != nil {
			panic(err)
		}

		url, err := url.Parse(httpOpts.URL)
		if err != nil {
			panic(err)
		}

		_, err = http.DefaultClient.Do(&http.Request{
			Method: httpOpts.Method,
			URL:    url,
			Body:   io.NopCloser(strings.NewReader(body)),
		})

		if err != nil {
			panic(err)
		}
	}, []extism.ValueType{extism.ValueTypePTR, extism.ValueTypePTR}, []extism.ValueType{})
)
