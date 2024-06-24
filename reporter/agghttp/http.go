package agghttp

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
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
	url           string
	client        HTTPDoer
	logger        *log.Logger
	batchInterval time.Duration
	batchSize     int
	maxBacklog    int
	batchMtx      *sync.Mutex
	batch         []*model.AggregatedSpan
	spanC         chan *model.AggregatedSpan
	sendC         chan struct{}
	quit          chan struct{}
	shutdown      chan error
	reqCallback   RequestCallbackFn
	reqTimeout    time.Duration
	serializer    reporter.SpanSerializer
}

// Send implements reporter
func (r *httpReporter) Send(s model.AggregatedSpan) {
	go func(span model.AggregatedSpan) {
		r.spanC <- &span
	}(s)
}

// Close implements reporter
func (r *httpReporter) Close() error {
	close(r.quit)
	return <-r.shutdown
}

func (r *httpReporter) loop() {
	var (
		nextSend   = time.Now().Add(r.batchInterval)
		ticker     = time.NewTicker(r.batchInterval / 10)
		tickerChan = ticker.C
	)
	defer ticker.Stop()

	for {
		select {
		case span := <-r.spanC:
			currentBatchSize := r.append(span)
			if currentBatchSize >= r.batchSize {
				nextSend = time.Now().Add(r.batchInterval)
				r.enqueueSend()
			}
		case <-tickerChan:
			if time.Now().After(nextSend) {
				nextSend = time.Now().Add(r.batchInterval)
				r.enqueueSend()
			}
		case <-r.quit:
			close(r.sendC)
			return
		}
	}
}

func (r *httpReporter) sendLoop() {
	for range r.sendC {
		_ = r.sendBatch()
	}
	r.shutdown <- r.sendBatch()
}

func (r *httpReporter) enqueueSend() {
	select {
	case r.sendC <- struct{}{}:
	default:
		// Do nothing if there's a pending send request already
	}
}

func (r *httpReporter) append(span *model.AggregatedSpan) (newBatchSize int) {
	r.batchMtx.Lock()

	r.batch = append(r.batch, span)
	if len(r.batch) > r.maxBacklog {
		dispose := len(r.batch) - r.maxBacklog
		// r.logger.Printf("backlog too long, disposing %d spans", dispose)
		r.batch = r.batch[dispose:]
	}
	newBatchSize = len(r.batch)

	r.batchMtx.Unlock()
	return
}

func (r *httpReporter) sendBatch() error {
	// Select all current spans in the batch to be sent
	r.batchMtx.Lock()
	sendBatch := r.batch[:]
	r.batchMtx.Unlock()

	if len(sendBatch) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(sendBatch)
	if err != nil {
		r.logger.Printf("failed when marshalling the spans batch: %s\n", err.Error())
		return err
	}

	req, err := http.NewRequest("POST", r.url, bytes.NewBuffer(jsonData))
	if err != nil {
		r.logger.Printf("failed when creating the request: %s\n", err.Error())
		return err
	}

	ctx, cancel := context.WithTimeout(req.Context(), r.reqTimeout)
	defer cancel()

	resp, err := r.client.Do(req.WithContext(ctx))
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
	}

	// Remove sent spans from the batch even if they were not saved
	r.batchMtx.Lock()
	r.batch = r.batch[len(sendBatch):]
	r.batchMtx.Unlock()

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

// BatchSize sets the maximum batch size, after which a collect will be
// triggered. The default batch size is 100 traces.
func BatchSize(n int) ReporterOption {
	return func(r *httpReporter) { r.batchSize = n }
}

// MaxBacklog sets the maximum backlog size. When batch size reaches this
// threshold, spans from the beginning of the batch will be disposed.
func MaxBacklog(n int) ReporterOption {
	return func(r *httpReporter) { r.maxBacklog = n }
}

// BatchInterval sets the maximum duration we will buffer traces before
// emitting them to the collector. The default batch interval is 1 second.
func BatchInterval(d time.Duration) ReporterOption {
	return func(r *httpReporter) { r.batchInterval = d }
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
		url:           url,
		logger:        log.New(os.Stderr, "", log.LstdFlags),
		client:        &http.Client{},
		batchInterval: defaultBatchInterval,
		batchSize:     defaultBatchSize,
		maxBacklog:    defaultMaxBacklog,
		batch:         []*model.AggregatedSpan{},
		spanC:         make(chan *model.AggregatedSpan),
		sendC:         make(chan struct{}, 1),
		quit:          make(chan struct{}, 1),
		shutdown:      make(chan error, 1),
		batchMtx:      &sync.Mutex{},
		serializer:    reporter.JSONSerializer{},
		reqTimeout:    defaultTimeout,
	}

	for _, opt := range opts {
		opt(&r)
	}

	go r.loop()
	go r.sendLoop()

	return &r
}
