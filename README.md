<!--
Copyright 2022 The Yangfisher1 Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# Zipkin Library for Go

[![GHA](https://github.com/Yangfisher1/zipkin-go/actions/workflows/ci.yml/badge.svg?event=push)](https://github.com/Yangfisher1/zipkin-go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Yangfisher1/zipkin-go/branch/master/graph/badge.svg?token=gXdWofFlsq)](https://codecov.io/gh/Yangfisher1/zipkin-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/Yangfisher1/zipkin-go)](https://goreportcard.com/report/github.com/Yangfisher1/zipkin-go)
[![GoDoc](https://godoc.org/github.com/Yangfisher1/zipkin-go?status.svg)](https://godoc.org/github.com/Yangfisher1/zipkin-go)
[![Gitter chat](https://badges.gitter.im/Yangfisher1/zipkin.svg)](https://gitter.im/Yangfisher1/zipkin?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)
[![Sourcegraph](https://sourcegraph.com/github.com/Yangfisher1/zipkin-go/-/badge.svg)](https://sourcegraph.com/github.com/Yangfisher1/zipkin-go?badge)

Zipkin Go is the official Go Tracer / Tracing implementation for Zipkin, 
supported by the Yangfisher1 community.

## package organization
`zipkin-go` is built with interoperability in mind within the Yangfisher1
community and even 3rd parties, the library consists of several packages.

The main tracing implementation can be found in the root folder of this
repository. Reusable parts not considered core implementation or deemed
beneficiary for usage by others are placed in their own packages within this
repository.

### model
This library implements the Zipkin V2 Span Model which is available in the model
package. It contains a Go data model compatible with the Zipkin V2 API and can
automatically sanitize, parse and (de)serialize to and from the required JSON
representation as used by the official Zipkin V2 Collectors.

### propagation
The propagation package and B3 subpackage hold the logic for propagating
SpanContext (span identifiers and sampling flags) between services participating
in traces. Currently Zipkin B3 Propagation is supported for HTTP and GRPC.

### middleware
The middleware subpackages contain officially supported middleware handlers and
tracing wrappers.

#### http
An easy to use http.Handler middleware for tracing server side requests is
provided. This allows one to use this middleware in applications using
standard library servers as well as most available higher level frameworks. Some
frameworks will have their own instrumentation and middleware that maps better
for their ecosystem.

For HTTP client operations `NewTransport` can return a `http.RoundTripper`
implementation that can either wrap the standard http.Client's Transport or a
custom provided one and add per request tracing. Since HTTP Requests can have
one or multiple redirects it is advisable to always enclose HTTP Client calls
with a `Span` either around the `*http.Client` call level or parent function
level.

For convenience `NewClient` is provided which returns a HTTP Client which embeds
`*http.Client` and provides an `application span` around the HTTP calls when
calling the `DoWithAppSpan()` method.

#### grpc
Easy to use grpc.StatsHandler middleware are provided for tracing gRPC server
and client requests. 

For a server, pass `NewServerHandler` when calling `NewServer`, e.g.,

```go
import (
	"google.golang.org/grpc"
	zipkingrpc "github.com/Yangfisher1/zipkin-go/middleware/grpc"
)

server = grpc.NewServer(grpc.StatsHandler(zipkingrpc.NewServerHandler(tracer)))
```

For a client, pass `NewClientHandler` when calling `Dial`, e.g.,

```go
import (
	"google.golang.org/grpc"
	zipkingrpc "github.com/Yangfisher1/zipkin-go/middleware/grpc"
)

conn, err = grpc.Dial(addr, grpc.WithStatsHandler(zipkingrpc.NewClientHandler(tracer)))
```

### reporter
The reporter package holds the interface which the various Reporter
implementations use. It is exported into its own package as it can be used by
3rd parties to use these Reporter packages in their own libraries for exporting
to the Zipkin ecosystem. The `zipkin-go` tracer also uses the interface to
accept 3rd party Reporter implementations.

#### HTTP Reporter
Most common Reporter type used by Zipkin users transporting Spans to the Zipkin
server using JSON over HTTP. The reporter holds a buffer and reports to the
backend asynchronously.

#### Kafka Reporter
High performance Reporter transporting Spans to the Zipkin server using a Kafka
Producer digesting JSON V2 Spans. The reporter uses the
[Sarama async producer](https://pkg.go.dev/github.com/IBM/sarama#AsyncProducer)
underneath.

## usage and examples
[HTTP Server Example](examples/httpserver_test.go)
