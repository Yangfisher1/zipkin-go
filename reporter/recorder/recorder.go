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

/*
Package recorder implements a reporter to record spans in v2 format.
*/
package recorder

import (
	"sync"

	"github.com/Yangfisher1/zipkin-go/model"
)

// ReporterRecorder records Zipkin spans.
type ReporterRecorder struct {
	mtx   sync.Mutex
	spans []model.SpanModel
}

// NewReporter returns a new recording reporter.
func NewReporter() *ReporterRecorder {
	return &ReporterRecorder{}
}

// Send adds the provided span to the span list held by the recorder.
func (r *ReporterRecorder) Send(span model.SpanModel) {
	r.mtx.Lock()
	r.spans = append(r.spans, span)
	r.mtx.Unlock()
}

// Flush returns all recorded spans and clears its internal span storage
func (r *ReporterRecorder) Flush() []model.SpanModel {
	r.mtx.Lock()
	spans := r.spans
	r.spans = nil
	r.mtx.Unlock()
	return spans
}

// Close flushes the reporter
func (r *ReporterRecorder) Close() error {
	r.Flush()
	return nil
}
