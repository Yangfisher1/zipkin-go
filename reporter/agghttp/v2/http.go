package v2

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Yangfisher1/zipkin-go/model"
	"github.com/Yangfisher1/zipkin-go/reporter"
)

// defaults
const (
	defaultTimeout       = 5 * time.Second // timeout for http request in seconds
	defaultBatchInterval = 1 * time.Second // BatchInterval in seconds
	defaultBatchSize     = 100
	defaultMaxBacklog    = 1000
)

// HTTPDoer will do a request to the Zipkin HTTP Collector
type HTTPDoer interface { // nolint: revive // keep as is, we don't want to break dependendants
	Do(req *http.Request) (*http.Response, error)
}

// httpReporter will send spans to a Zipkin HTTP Collector using Zipkin V2 API.
type httpReporter struct {
	url         string
	client      HTTPDoer
	logger      *log.Logger
	reqCallback RequestCallbackFn
	reqTimeout  time.Duration
	serializer  reporter.SpanSerializer
}

// Send implements reporter
func (r *httpReporter) Send(s model.AggregatedSpan) {
	go func(span model.AggregatedSpan) {
		r.send(span)
	}(s)
}

// Async send the whole trace
func (r *httpReporter) send(s model.AggregatedSpan) {
	var batch []*model.AggregatedSpan
	batch = append(batch, &s)

	jsonData, err := json.Marshal(batch)
	if err != nil {
		// r.logger.Printf("failed when marshalling the spans batch: %s\n", err.Error())
	}

	req, err := http.NewRequest("POST", r.url, bytes.NewBuffer(jsonData))
	if err != nil {
		// r.logger.Printf("failed when creating the request: %s\n", err.Error())
	}

	ctx, cancel := context.WithTimeout(req.Context(), r.reqTimeout)
	defer cancel()

	resp, err := r.client.Do(req.WithContext(ctx))
	if err != nil {
		// r.logger.Printf("failed to send the request: %s\n", err.Error())
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		// r.logger.Printf("failed the request with status code %d\n", resp.StatusCode)
	}
}

// Close implements reporter
func (r *httpReporter) Close() error {
	return nil
}

// RequestCallbackFn receives the initialized request from the Collector before
// sending it over the wire. This allows one to plug in additional headers or
// do other customization.
type RequestCallbackFn func(*http.Request)

// ReporterOption sets a parameter for the HTTP Reporter
type ReporterOption func(r *httpReporter)

// Timeout sets maximum timeout for the http request through its context.
func Timeout(duration time.Duration) ReporterOption {
	return func(r *httpReporter) { r.reqTimeout = duration }
}

// Client sets a custom http client to use under the interface HTTPDoer
// which includes a `Do` method with same signature as the *http.Client
func Client(client HTTPDoer) ReporterOption {
	return func(r *httpReporter) { r.client = client }
}

// RequestCallback registers a callback function to adjust the reporter
// *http.Request before it sends the request to Zipkin.
func RequestCallback(rc RequestCallbackFn) ReporterOption {
	return func(r *httpReporter) { r.reqCallback = rc }
}

// Logger sets the logger used to report errors in the collection
// process.
func Logger(l *log.Logger) ReporterOption {
	return func(r *httpReporter) { r.logger = l }
}

// Serializer sets the serialization function to use for sending span data to
// Zipkin.
func Serializer(serializer reporter.SpanSerializer) ReporterOption {
	return func(r *httpReporter) {
		if serializer != nil {
			r.serializer = serializer
		}
	}
}

// NewReporter returns a new HTTP Reporter.
// url should be the endpoint to send the spans to, e.g.
// http://localhost:9411/api/v2/spans
func NewReporter(url string, opts ...ReporterOption) reporter.AggregatedReporter {
	r := httpReporter{
		url:        url,
		logger:     log.New(os.Stderr, "", log.LstdFlags),
		client:     &http.Client{},
		serializer: reporter.JSONSerializer{},
		reqTimeout: defaultTimeout,
	}

	for _, opt := range opts {
		opt(&r)
	}

	return &r
}
