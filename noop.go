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

package zipkin

import (
	"time"

	"github.com/Yangfisher1/zipkin-go/model"
)

type noopSpan struct {
	model.SpanContext
}

func (n *noopSpan) Context() model.SpanContext { return n.SpanContext }

func (n *noopSpan) SetName(string) {}

func (*noopSpan) SetRemoteEndpoint(*model.Endpoint) {}

func (*noopSpan) Annotate(time.Time, string) {}

func (*noopSpan) Tag(string, string) {}

func (*noopSpan) Finish() {}

func (*noopSpan) FinishedWithDuration(_ time.Duration) {}

func (*noopSpan) Flush() {}

// IsNoop tells whether the span is noop or not. Usually used to avoid resource misusage
// when customizing a span as data won't be recorded
func IsNoop(s Span) bool {
	_, ok := s.(*noopSpan)
	return ok
}
