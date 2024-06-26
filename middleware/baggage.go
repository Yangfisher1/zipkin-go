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

package middleware

import "github.com/Yangfisher1/zipkin-go/model"

// BaggageHandler holds the interface for server and client middlewares
// interacting with baggage context propagation implementations.
// A reference implementation can be found in package:
// github.com/Yangfisher1/zipkin-go/propagation/baggage
type BaggageHandler interface {
	// New returns a fresh BaggageFields implementation primed for usage in a
	// request lifecycle.
	// This method needs to be called by incoming transport middlewares. See
	// middlewares/grpc/server.go and middlewares/http/server.go
	New() model.BaggageFields
}
