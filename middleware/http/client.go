// Copyright 2022 The Yangfisher1 Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Yangfisher1/zipkin-go"
	"github.com/Yangfisher1/zipkin-go/model"
)

// ErrValidTracerRequired error
var ErrValidTracerRequired = errors.New("valid tracer required")

// Client holds a Zipkin instrumented HTTP Client.
type Client struct {
	*http.Client
	tracer           *zipkin.Tracer
	httpTrace        bool
	defaultTags      map[string]string
	transportOptions []TransportOption
	remoteEndpoint   *model.Endpoint
}

// ClientOption allows optional configuration of Client.
type ClientOption func(*Client)

// WithClient allows one to add a custom configured http.Client to use.
func WithClient(client *http.Client) ClientOption {
	return func(c *Client) {
		if client == nil {
			client = &http.Client{}
		}
		c.Client = client
	}
}

// ClientTrace allows one to enable Go's net/http/httptrace.
func ClientTrace(enabled bool) ClientOption {
	return func(c *Client) {
		c.httpTrace = enabled
	}
}

// ClientTags adds default Tags to inject into client application spans.
func ClientTags(tags map[string]string) ClientOption {
	return func(c *Client) {
		c.defaultTags = tags
	}
}

// TransportOptions passes optional Transport configuration to the internal
// transport used by Client.
func TransportOptions(options ...TransportOption) ClientOption {
	return func(c *Client) {
		c.transportOptions = options
	}
}

// WithRemoteEndpoint will set the remote endpoint for all spans.
func WithRemoteEndpoint(remoteEndpoint *model.Endpoint) ClientOption {
	return func(c *Client) {
		c.remoteEndpoint = remoteEndpoint
	}
}

// NewClient returns an HTTP Client adding Zipkin instrumentation around an
// embedded standard Go http.Client.
func NewClient(tracer *zipkin.Tracer, options ...ClientOption) (*Client, error) {
	if tracer == nil {
		return nil, ErrValidTracerRequired
	}

	c := &Client{tracer: tracer, Client: &http.Client{}}
	for _, option := range options {
		option(c)
	}

	c.transportOptions = append(
		c.transportOptions,
		// the following Client settings override provided transport settings.
		RoundTripper(c.Client.Transport),
		TransportTrace(c.httpTrace),
		TransportRemoteEndpoint(c.remoteEndpoint),
	)
	tr, err := NewTransport(tracer, c.transportOptions...)
	if err != nil {
		return nil, err
	}
	c.Client.Transport = tr

	return c, nil
}

// DoWithAppSpan wraps http.Client's Do with tracing using an application span.
func (c *Client) DoWithAppSpan(req *http.Request, name string) (*http.Response, error) {
	var parentContext model.SpanContext

	if span := zipkin.SpanFromContext(req.Context()); span != nil {
		parentContext = span.Context()
	}

	appSpan := c.tracer.StartSpan(name, zipkin.Parent(parentContext), zipkin.RemoteEndpoint(c.remoteEndpoint))

	zipkin.TagHTTPMethod.Set(appSpan, req.Method)
	zipkin.TagHTTPPath.Set(appSpan, req.URL.Path)

	res, err := c.Do(
		req.WithContext(zipkin.NewContext(req.Context(), appSpan)),
	)
	if err != nil {
		zipkin.TagError.Set(appSpan, err.Error())
		appSpan.Finish()
		return res, err
	}

	if c.httpTrace {
		appSpan.Annotate(time.Now(), "wr")
	}

	if res.ContentLength > 0 {
		zipkin.TagHTTPResponseSize.Set(appSpan, strconv.FormatInt(res.ContentLength, 10))
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		statusCode := strconv.FormatInt(int64(res.StatusCode), 10)
		zipkin.TagHTTPStatusCode.Set(appSpan, statusCode)
		if res.StatusCode > 399 {
			zipkin.TagError.Set(appSpan, statusCode)
		}
	}

	res.Body = &spanCloser{
		ReadCloser:   res.Body,
		sp:           appSpan,
		traceEnabled: c.httpTrace,
	}
	return res, err
}
