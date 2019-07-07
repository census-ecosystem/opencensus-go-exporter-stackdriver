// Copyright 2018, OpenCensus Authors
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

package propagation

import (
	"net/http"
	"reflect"
	"testing"

	"go.opencensus.io/trace"
)

func TestHTTPFormat(t *testing.T) {
	format := &HTTPFormat{}
	emptySpanContext := trace.SpanContext{}
	tests := []struct {
		succesfulDecoding bool
		incoming          string
		wantSpanContext   trace.SpanContext
		outgoing          string
	}{
		{
			succesfulDecoding: true,
			incoming:          "105445aa7843bc8bf206b12000100000/105445aa7843bc8b;o=1",
			wantSpanContext: trace.SpanContext{
				TraceID:      [16]byte{16, 84, 69, 170, 120, 67, 188, 139, 242, 6, 177, 32, 0, 16, 0, 0},
				SpanID:       [8]byte{16, 84, 69, 170, 120, 67, 188, 139},
				TraceOptions: 1,
			},
			outgoing: "105445aa7843bc8bf206b12000100000/105445aa7843bc8b;o=1",
		},
		{
			succesfulDecoding: true,
			incoming:          "105445aa7843bc8bf206b12000100000/307349a6a1f76af6;o=0",
			wantSpanContext: trace.SpanContext{
				TraceID:      [16]byte{16, 84, 69, 170, 120, 67, 188, 139, 242, 6, 177, 32, 0, 16, 0, 0},
				SpanID:       [8]byte{48, 115, 73, 166, 161, 247, 106, 246},
				TraceOptions: 0,
			},
			outgoing: "105445aa7843bc8bf206b12000100000/307349a6a1f76af6;o=0",
		},
		{
			// Optional trace options are not present
			succesfulDecoding: true,
			incoming:          "105445aa7843bc8bf206b12000100000/105445aa7843bc8b",
			wantSpanContext: trace.SpanContext{
				TraceID:      [16]byte{16, 84, 69, 170, 120, 67, 188, 139, 242, 6, 177, 32, 0, 16, 0, 0},
				SpanID:       [8]byte{16, 84, 69, 170, 120, 67, 188, 139},
				TraceOptions: 0,
			},
			outgoing: "105445aa7843bc8bf206b12000100000/105445aa7843bc8b;o=0",
		},
		{
			// Non-integer option
			succesfulDecoding: false,
			incoming:          "105445aa7843bc8bf206b12000100000/307349a6a1f76af6;o=a",
			wantSpanContext:   emptySpanContext,
			outgoing:          "00000000000000000000000000000000/0000000000000000;o=0",
		},
		{
			// Odd-length trace id
			succesfulDecoding: false,
			incoming:          "105445aa7843bc8bf206b1200010000/307349a6a1f76af6;o=0",
			wantSpanContext:   emptySpanContext,
			outgoing:          "00000000000000000000000000000000/0000000000000000;o=0",
		},
		{
			// Odd-length span id
			succesfulDecoding: false,
			incoming:          "105445aa7843bc8bf206b12000100000/307349a6a1f76af;o=0",
			wantSpanContext:   emptySpanContext,
			outgoing:          "00000000000000000000000000000000/0000000000000000;o=0",
		},
		{
			// No trace id, random text as header content
			succesfulDecoding: false,
			incoming:          "105445aa7843bc8bf206b12000100",
			wantSpanContext:   emptySpanContext,
			outgoing:          "00000000000000000000000000000000/0000000000000000;o=0",
		},
		{
			// No trace id, random text as header content
			succesfulDecoding: false,
			incoming:          "",
			wantSpanContext:   emptySpanContext,
			outgoing:          "00000000000000000000000000000000/0000000000000000;o=0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.incoming, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			req.Header.Add(httpHeader, tt.incoming)
			sc, ok := format.SpanContextFromRequest(req)
			if ok != tt.succesfulDecoding {
				t.Errorf("exporter.SpanContextFromRequest() = false; want true")
			}
			if got, want := sc, tt.wantSpanContext; !reflect.DeepEqual(got, want) {
				t.Errorf("exporter.SpanContextFromRequest() returned span context %v; want %v", got, want)
			}

			req, _ = http.NewRequest("GET", "http://example.com", nil)
			format.SpanContextToRequest(sc, req)
			if got, want := req.Header.Get(httpHeader), tt.outgoing; got != want {
				t.Errorf("exporter.SpanContextToRequest() returned header %q; want %q", got, want)
			}
		})
	}
}
