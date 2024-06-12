package sync

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Yangfisher1/zipkin-go/model"
	"github.com/Yangfisher1/zipkin-go/reporter"
)

// HTTPDoer will do a request to the Zipkin HTTP Collector
type HTTPDoer interface { // nolint: revive // keep as is, we don't want to break dependendants
	Do(req *http.Request) (*http.Response, error)
}

type syncReporter struct {
	url         string
	client      HTTPDoer
	logger      *log.Logger
	reqCallback RequestCallbackFn
	reqTimeout  time.Duration
	serializer  reporter.SpanSerializer
}

func (r *syncReporter) Send(s model.SpanModel) {
	var batch []*model.SpanModel
	batch = append(batch, &s)

	body, err := r.serializer.Serialize(batch)
	if err != nil {
		r.logger.Printf("failed when marshalling the spans batch: %s\n", err.Error())
		return
	}

	req, err := http.NewRequest("POST", r.url, bytes.NewReader(body))
	if err != nil {
		r.logger.Printf("failed when creating the request: %s\n", err.Error())
		return
	}

	// By default we send b3:0 header to mitigate trace reporting amplification in
	// service mesh environments where the sidecar proxies might trace the call
	// we do here towards the Zipkin collector.
	req.Header.Set("b3", "0")

	req.Header.Set("Content-Type", r.serializer.ContentType())
	if r.reqCallback != nil {
		r.reqCallback(req)
	}

	ctx, cancel := context.WithTimeout(req.Context(), r.reqTimeout)
	defer cancel()

	resp, err := r.client.Do(req.WithContext(ctx))
	if err != nil {
		r.logger.Printf("failed to send the request: %s\n", err.Error())
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		r.logger.Printf("failed the request with status code %d\n", resp.StatusCode)
	}

}

func (r *syncReporter) Close() error {
	return nil // nothing to do
}

// RequestCallbackFn receives the initialized request from the Collector before
// sending it over the wire. This allows one to plug in additional headers or
// do other customization.
type RequestCallbackFn func(*http.Request)

// ReporterOption sets a parameter for the HTTP Reporter
type ReporterOption func(r *syncReporter)

// Timeout sets maximum timeout for the http request through its context.
func Timeout(duration time.Duration) ReporterOption {
	return func(r *syncReporter) { r.reqTimeout = duration }
}

// Client sets a custom http client to use under the interface HTTPDoer
// which includes a `Do` method with same signature as the *http.Client
func Client(client HTTPDoer) ReporterOption {
	return func(r *syncReporter) { r.client = client }
}

// RequestCallback registers a callback function to adjust the reporter
// *http.Request before it sends the request to Zipkin.
func RequestCallback(rc RequestCallbackFn) ReporterOption {
	return func(r *syncReporter) { r.reqCallback = rc }
}

// Logger sets the logger used to report errors in the collection
// process.
func Logger(l *log.Logger) ReporterOption {
	return func(r *syncReporter) { r.logger = l }
}

// Serializer sets the serialization function to use for sending span data to
// Zipkin.
func Serializer(serializer reporter.SpanSerializer) ReporterOption {
	return func(r *syncReporter) {
		if serializer != nil {
			r.serializer = serializer
		}
	}
}

func NewSyncReporter(url string, opts ...ReporterOption) reporter.Reporter {
	r := syncReporter{
		url:        url,
		logger:     log.New(os.Stderr, "", log.LstdFlags),
		client:     &http.Client{},
		serializer: reporter.JSONSerializer{},
		reqTimeout: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(&r)
	}

	return &r
}
