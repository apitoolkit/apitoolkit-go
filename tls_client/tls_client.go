package apitoolkit_tlsclient

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	apitoolkit "github.com/apitoolkit/apitoolkit-go"
	fhttp "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/google/uuid"
)

type InstrumentedHttpClient struct {
	ctx              context.Context
	client           tls_client.HttpClient // Assuming tlsclient is the package name
	apitoolkitClient *apitoolkit.Client
}

func NewHttpClient(ctx context.Context, client tls_client.HttpClient, apitoolkitClient *apitoolkit.Client) *InstrumentedHttpClient {
	return &InstrumentedHttpClient{
		ctx:              ctx,
		client:           client,
		apitoolkitClient: apitoolkitClient,
	}
}

func (c *InstrumentedHttpClient) GetCookies(u *url.URL) []*fhttp.Cookie {
	return c.client.GetCookies(u)
}

func (c *InstrumentedHttpClient) SetCookies(u *url.URL, cookies []*fhttp.Cookie) {
	c.client.SetCookies(u, cookies)
}

func (c *InstrumentedHttpClient) SetCookieJar(jar fhttp.CookieJar) {
	c.client.SetCookieJar(jar)
}

func (c *InstrumentedHttpClient) GetCookieJar() fhttp.CookieJar {
	return c.client.GetCookieJar()
}

func (c *InstrumentedHttpClient) SetProxy(proxyUrl string) error {
	return c.client.SetProxy(proxyUrl)
}

func (c *InstrumentedHttpClient) GetProxy() string {
	return c.client.GetProxy()
}

func (c *InstrumentedHttpClient) SetFollowRedirect(followRedirect bool) {
	c.client.SetFollowRedirect(followRedirect)
}

func (c *InstrumentedHttpClient) GetFollowRedirect() bool {
	return c.client.GetFollowRedirect()
}

func (c *InstrumentedHttpClient) CloseIdleConnections() {
	c.client.CloseIdleConnections()
}

func (c *InstrumentedHttpClient) Get(url string) (*fhttp.Response, error) {
	req, err := fhttp.NewRequest(fhttp.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *InstrumentedHttpClient) Head(url string) (*fhttp.Response, error) {
	req, err := fhttp.NewRequest(fhttp.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *InstrumentedHttpClient) Post(url, contentType string, body io.Reader) (*fhttp.Response, error) {
	req, err := fhttp.NewRequest(fhttp.MethodPost, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return c.Do(req)
}

func (c *InstrumentedHttpClient) Do(req *fhttp.Request) (res *fhttp.Response, err error) {
	defer func() {
		if err != nil {
			apitoolkit.ReportError(c.ctx, err)
		}
	}()

	if c.client == nil {
		log.Println("APIToolkit: outgoing rountripper has a nil Apitoolkit client.")
		return c.client.Do(req)
	}

	reqBodyBytes := []byte{}
	if req.Body != nil {
		reqBodyBytes, _ = io.ReadAll(req.Body)
		req.Body = io.NopCloser(bytes.NewBuffer(reqBodyBytes))
	}

	// Add a header to all outgoing requests "X-APITOOLKIT-TRACE-PARENT-ID"
	start := time.Now()

	res, err = c.client.Do(req)

	var errorList []apitoolkit.ATError
	if err != nil {
		errorList = append(errorList, apitoolkit.BuildError(err))
	}

	var payload apitoolkit.Payload
	var parentMsgIDPtr *uuid.UUID
	parentMsgID, ok := c.ctx.Value(apitoolkit.CurrentRequestMessageID).(uuid.UUID)
	if ok {
		parentMsgIDPtr = &parentMsgID
	}

	// Capture the response body
	if res != nil {
		respBodyBytes, _ := io.ReadAll(res.Body)
		res.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))
		payload = BuildPayload(
			c.apitoolkitClient,
			apitoolkit.GoOutgoing,
			start, req, res.StatusCode, reqBodyBytes,
			respBodyBytes, res.Header, nil,
			req.URL.Path,
			nil, nil, nil,
			errorList,
			uuid.Must(uuid.NewRandom()),
			parentMsgIDPtr,
		)
	} else {
		payload = BuildPayload(
			c.apitoolkitClient,
			apitoolkit.GoOutgoing,
			start, req, 503, reqBodyBytes,
			nil, nil, nil,
			req.URL.Path,
			nil, nil, nil,
			errorList,
			uuid.Must(uuid.NewRandom()),
			parentMsgIDPtr,
		)
	}

	pErr := c.apitoolkitClient.PublishMessage(req.Context(), payload)
	if pErr != nil {
		c.apitoolkitClient.ReportError(c.ctx, pErr)
		config := c.apitoolkitClient.GetConfig()

		if config.Debug {
			log.Println("APIToolkit: unable to publish outgoing request payload to pubsub.")
		}
	}
	return res, err
}

func BuildPayload(c *apitoolkit.Client, SDKType string, trackingStart time.Time, req *fhttp.Request,
	statusCode int, reqBody []byte, respBody []byte, respHeader map[string][]string,
	pathParams map[string]string, urlPath string,
	redactHeadersList,
	redactRequestBodyList, redactResponseBodyList []string,
	errorList []apitoolkit.ATError,
	msgID uuid.UUID,
	parentID *uuid.UUID,
) apitoolkit.Payload {
	config := c.GetConfig()
	metadata := c.GetMetadata()

	if req == nil || c == nil || req.URL == nil {
		// Early return with empty payload to prevent any nil pointer panics
		if config.Debug {
			log.Println("APIToolkit: nil request or client or url while building payload.")
		}
		return apitoolkit.Payload{}
	}
	projectId := ""
	if metadata != nil {
		projectId = metadata.ProjectId
	}

	redactedHeaders := []string{"password", "Authorization", "Cookies"}
	for _, v := range redactHeadersList {
		redactedHeaders = append(redactedHeaders, strings.ToLower(v))
	}

	since := time.Since(trackingStart)
	var parentIDVal *string
	if parentID != nil {
		parentIDStr := (*parentID).String()
		parentIDVal = &parentIDStr
	}
	var serviceVersion *string
	if config.ServiceVersion != "" {
		serviceVersion = &config.ServiceVersion
	}
	return apitoolkit.Payload{
		Duration:        since,
		Host:            req.Host,
		Method:          req.Method,
		PathParams:      pathParams,
		ProjectID:       projectId,
		ProtoMajor:      req.ProtoMajor,
		ProtoMinor:      req.ProtoMinor,
		QueryParams:     req.URL.Query(),
		RawURL:          req.URL.RequestURI(),
		Referer:         req.Referer(),
		RequestBody:     apitoolkit.RedactJSON(reqBody, redactRequestBodyList),
		RequestHeaders:  apitoolkit.RedactHeaders(req.Header, redactedHeaders),
		ResponseBody:    apitoolkit.RedactJSON(respBody, redactResponseBodyList),
		ResponseHeaders: apitoolkit.RedactHeaders(respHeader, redactedHeaders),
		SdkType:         SDKType,
		StatusCode:      statusCode,
		Timestamp:       time.Now(),
		URLPath:         urlPath,
		Errors:          errorList,
		ServiceVersion:  serviceVersion,
		Tags:            config.Tags,
		MsgID:           msgID.String(),
		ParentID:        parentIDVal,
	}
}
