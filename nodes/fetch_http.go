package nodes

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"axiom-official/generic-echo/axiom"
	gen "axiom-official/generic-echo/gen"
)

// FetchHTTP is a GENERIC (ADR-122) node: unlike Echo's field-less ports
// (EchoInputPlaceholder/EchoOutputPlaceholder), its input/output are
// ordinary, concrete, named messages (HTTPRequest/HTTPResponse) with
// GENERALIZED fields — url/method/body and status_code/body — broad enough
// to cover any HTTP call, read and written exactly like any other node's
// generated struct fields. No wire-format knowledge required — this is the
// pattern axiom-package-authoring's "Generic nodes (kind: generic)" section
// documents. Only placeable via an Instance that maps a flow's real,
// narrower shape onto these generalized fields.
func FetchHTTP(ctx context.Context, ax axiom.Context, input *gen.HTTPRequest) (*gen.HTTPResponse, error) {
	method := input.Method
	if method == "" {
		method = http.MethodGet
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var body io.Reader
	if input.Body != "" {
		body = strings.NewReader(input.Body)
	}
	req, err := http.NewRequestWithContext(reqCtx, method, input.Url, body)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Cap the body we read back — a demo fixture has no business buffering an
	// unbounded response into memory.
	const maxBody = 1 << 20 // 1 MiB
	buf, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, err
	}

	return &gen.HTTPResponse{
		StatusCode: int32(resp.StatusCode),
		Body:       string(buf),
	}, nil
}
