package http

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

type serverlessReporter struct {
	errorReporter *httpReporter
	batchedUrl    string
	batch         []*model.AggregatedSpan
	batchInterval time.Duration
	batchSize     int
	maxBacklog    int
	batchMtx      *sync.Mutex
	spanC         chan *model.AggregatedSpan
	sendC         chan struct{}
	quit          chan struct{}
	shutdown      chan error
}

func (r *serverlessReporter) SendAggregatedSpans(spans model.AggregatedSpan) {
	r.spanC <- &spans
}

func (r *serverlessReporter) Send(s model.SpanModel) {
	r.errorReporter.spanC <- &s
}

func (r *serverlessReporter) Close() error {
	close(r.errorReporter.quit)
	<-r.errorReporter.shutdown
	close(r.quit)
	return <-r.shutdown
}

func (r *serverlessReporter) loop() {
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

func (r *serverlessReporter) sendLoop() {
	for range r.sendC {
		_ = r.sendBatch()
	}
	r.shutdown <- r.sendBatch()
}

func (r *serverlessReporter) enqueueSend() {
	select {
	case r.sendC <- struct{}{}:
	default:
		// Do nothing if there's a pending send request already
	}
}

func (r *serverlessReporter) append(span *model.AggregatedSpan) (newBatchSize int) {
	r.batchMtx.Lock()

	r.batch = append(r.batch, span)
	if len(r.batch) > r.maxBacklog {
		dispose := len(r.batch) - r.maxBacklog
		// TODO: count drop rate %1 here
		r.batch = r.batch[dispose:]
	}
	newBatchSize = len(r.batch)

	r.batchMtx.Unlock()
	return
}

func (r *serverlessReporter) sendBatch() error {
	// Select all current spans in the batch to be sent
	r.batchMtx.Lock()
	sendBatch := r.batch[:]
	r.batchMtx.Unlock()

	if len(sendBatch) == 0 {
		return nil
	}

	jsonData, err := json.Marshal(sendBatch)
	if err != nil {
		r.errorReporter.logger.Printf("failed to marshal the batch: %s\n", err.Error())
		return err
	}

	req, err := http.NewRequest("POST", r.batchedUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		r.errorReporter.logger.Printf("failed when creating the request: %s\n", err.Error())
		return err
	}

	ctx, cancel := context.WithTimeout(req.Context(), r.errorReporter.reqTimeout)
	defer cancel()

	resp, err := r.errorReporter.client.Do(req.WithContext(ctx))
	if err != nil {
		r.errorReporter.logger.Printf("failed to send the request: %s\n", err.Error())
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		r.errorReporter.logger.Printf("failed the request with status code %d\n", resp.StatusCode)
	}

	// Remove sent spans from the batch even if they were not saved
	r.batchMtx.Lock()
	r.batch = r.batch[len(sendBatch):]
	r.batchMtx.Unlock()

	return nil
}

// NewServerlessReporter returns a new ServerlessReporter implementation.
func NewServerlessReporter(batchedUrl string, url string, opts ...ReporterOption) reporter.ServerlessReporter {
	r := serverlessReporter{
		errorReporter: &httpReporter{
			url:           url,
			logger:        log.New(os.Stderr, "", log.LstdFlags),
			client:        &http.Client{},
			batchInterval: defaultBatchInterval,
			batchSize:     defaultBatchSize,
			maxBacklog:    defaultMaxBacklog,
			batch:         []*model.SpanModel{},
			spanC:         make(chan *model.SpanModel),
			sendC:         make(chan struct{}, 1),
			quit:          make(chan struct{}, 1),
			shutdown:      make(chan error, 1),
			batchMtx:      &sync.Mutex{},
			serializer:    reporter.JSONSerializer{},
			reqTimeout:    defaultTimeout,
		},
		batchedUrl:    batchedUrl,
		batch:         []*model.AggregatedSpan{},
		batchInterval: 1 * time.Second,
		batchSize:     100,
		maxBacklog:    1000,
		batchMtx:      &sync.Mutex{},
		spanC:         make(chan *model.AggregatedSpan),
		sendC:         make(chan struct{}, 1),
		quit:          make(chan struct{}, 1),
		shutdown:      make(chan error, 1),
	}

	for _, opt := range opts {
		opt(r.errorReporter)
	}

	go r.errorReporter.loop()
	go r.errorReporter.sendLoop()
	go r.loop()
	go r.sendLoop()

	return &r
}
