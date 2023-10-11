package apitoolkit

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type roundTripper struct {
	base   http.RoundTripper
	ctx    context.Context
	client *Client
	cfg    *roundTripperConfig
}

func (rt *roundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	defer func() {
		ReportError(req.Context(), err)
	}()

	if rt.client == nil {
		log.Println("APIToolkit: outgoing rountripper has a nil Apitoolkit client.")
		return rt.base.RoundTrip(req)
	}

	// Capture the request body
	reqBodyBytes := []byte{}
	if req.Body != nil {
		reqBodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
	}

	// Add a header to all outgoing requests "X-APITOOLKIT-TRACE-PARENT-ID"
	start := time.Now()
	res, err = rt.base.RoundTrip(req)
	var errorList []ATError
	if err != nil {
		// Add the error for the given request payload
		errorList = append(errorList, buildError(err))
	}

	var payload Payload
	var parentMsgIDPtr *uuid.UUID
	parentMsgID, ok := req.Context().Value(CurrentRequestMessageID).(uuid.UUID)
	if ok {
		parentMsgIDPtr = &parentMsgID
	}

	// Capture the response body
	if res != nil {
		respBodyBytes, _ := io.ReadAll(res.Body)
		res.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))
		payload = rt.client.buildPayload(
			GoOutgoing,
			start, req, res.StatusCode, reqBodyBytes,
			respBodyBytes, res.Header, nil,
			req.URL.Path,
			rt.cfg.RedactHeaders, rt.cfg.RedactRequestBody, rt.cfg.RedactResponseBody,
			errorList,
			uuid.Must(uuid.NewRandom()),
			parentMsgIDPtr,
		)
	} else {
		payload = rt.client.buildPayload(
			GoOutgoing,
			start, req, 503, reqBodyBytes,
			nil, nil, nil,
			req.URL.Path,
			rt.cfg.RedactHeaders, rt.cfg.RedactRequestBody, rt.cfg.RedactResponseBody,
			errorList,
			uuid.Must(uuid.NewRandom()),
			parentMsgIDPtr,
		)
	}

	pErr := rt.client.PublishMessage(req.Context(), payload)
	if pErr != nil {
		ReportError(req.Context(), pErr)
		if rt.client.config.Debug {
			log.Println("APIToolkit: unable to publish outgoing request payload to pubsub.")
		}
	}
	return res, err
}

type roundTripperConfig struct {
	RedactHeaders      []string
	RedactRequestBody  []string
	RedactResponseBody []string
}

type RoundTripperOption func(*roundTripperConfig)

func WithRedactHeaders(headers []string) RoundTripperOption {
	return func(rc *roundTripperConfig) {
		rc.RedactHeaders = headers
	}
}

func WithRedactRequestBody(fields []string) RoundTripperOption {
	return func(rc *roundTripperConfig) {
		rc.RedactRequestBody = fields
	}
}

func WithRedactResponseBody(fields []string) RoundTripperOption {
	return func(rc *roundTripperConfig) {
		rc.RedactResponseBody = fields
	}
}

// WrapRoundTripper returns a new RoundTripper which traces all requests sent
// over the transport.
func (c *Client) WrapRoundTripper(ctx context.Context, rt http.RoundTripper, opts ...RoundTripperOption) http.RoundTripper {
	cfg := new(roundTripperConfig)
	for _, opt := range opts {
		opt(cfg)
	}

	// If no rt is passed in, then use the default standard library transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	return &roundTripper{
		base:   rt,
		ctx:    ctx,
		client: c,
		cfg:    cfg,
	}
}
