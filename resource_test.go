// Copyright 2019, OpenCensus Authors
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

package stackdriver // import "contrib.go.opencensus.io/exporter/stackdriver"

import (
	"fmt"
	"testing"

	"contrib.go.opencensus.io/resource/resourcekeys"
	"github.com/google/go-cmp/cmp"
	"go.opencensus.io/resource"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
)

func TestDefaultMapResource(t *testing.T) {
	cases := []struct {
		input *resource.Resource
		want  *monitoredrespb.MonitoredResource
	}{
		// Verify that the mapping works and that we skip over the
		// first mapping that doesn't apply.
		{
			input: &resource.Resource{
				Type: resourcekeys.GCPTypeGCEInstance,
				Labels: map[string]string{
					resourcekeys.GCPKeyGCEProjectID:  "proj1",
					resourcekeys.GCPKeyGCEInstanceID: "inst1",
					resourcekeys.GCPKeyGCEZone:       "zone1",
					"extra_key":                      "must be ignored",
				},
			},
			want: &monitoredrespb.MonitoredResource{
				Type: "gce_instance",
				Labels: map[string]string{
					"project_id":  "proj1",
					"instance_id": "inst1",
					"zone":        "zone1",
				},
			},
		},
		// No match due to missing key.
		{
			input: &resource.Resource{
				Type: resourcekeys.GCPTypeGCEInstance,
				Labels: map[string]string{
					resourcekeys.GCPKeyGCEProjectID:  "proj1",
					resourcekeys.GCPKeyGCEInstanceID: "inst1",
				},
			},
			want: nil,
		},
	}
	for i, c := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			got := defaultMapResource(c.input)
			if diff := cmp.Diff(got, c.want); diff != "" {
				t.Errorf("Values differ -got +want: %s", diff)
			}
		})
	}
}
