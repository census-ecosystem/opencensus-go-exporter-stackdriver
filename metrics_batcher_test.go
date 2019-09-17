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

package stackdriver

import (
	"context"
	"fmt"
	"testing"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"google.golang.org/api/option"
	googlemetricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
)

func TestWorkers(t *testing.T) {
	server, addr, doneFn := createFakeServer(t)
	defer doneFn()
	ctx := context.Background()

	c1, err := makeClient(addr)
	if err != nil {
		t.Fatalf("Failed to create metric client %v", err)
	}
	m1 := newMetricsBatcher(ctx, "test", 1, c1, defaultTimeout) // batcher with 1 worker

	c2, err := makeClient(addr)
	if err != nil {
		t.Fatalf("Failed to create metric client %v", err)
	}
	m2 := newMetricsBatcher(ctx, "test", 2, c2, defaultTimeout) // batcher with 2 workers

	tss := make([]*monitoringpb.TimeSeries, 0, 500) // make 500 time series, should be split to 3 reqs
	for i := 0; i < 500; i++ {
		tss = append(tss, makeTs(i))
	}

	for _, ts := range tss {
		m1.addTimeSeries(ts)
	}
	if err := m1.close(ctx); err != nil {
		t.Fatalf("Want no error, got %v", err)
	}
	reqs1 := server.stackdriverTimeSeries
	server.resetStackdriverTimeSeries()
	server.resetStackdriverMetricDescriptors()

	for _, ts := range tss {
		m2.addTimeSeries(ts)
	}
	if err := m2.close(ctx); err != nil {
		t.Fatalf("Want no error, got %v", err)
	}
	reqs2 := server.stackdriverTimeSeries

	if len(reqs1) != 3 {
		t.Fatalf("Want 3 CreateTimeSeriesReqs, got %v", len(reqs1))
	}
	if len(reqs2) != 3 {
		t.Fatalf("Want 3 CreateTimeSeriesReqs, got %v", len(reqs2))
	}
	if m1.droppedTimeSeries != m2.droppedTimeSeries {
		t.Fatalf("Dropped time series counts don't match, FromOneWorker: %v, FromTwoWorkers: %v", m1.droppedTimeSeries, m2.droppedTimeSeries)
	}
	if diff := cmpTSReqs(reqs1, reqs2); diff != "" {
		t.Fatalf("CreateTimeSeriesRequests don't match -FromOneWorker +FromTwoWorkers: %s", diff)
	}
}

func makeClient(addr string) (*monitoring.MetricClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return monitoring.NewMetricClient(context.Background(), option.WithGRPCConn(conn))
}

func makeTs(i int) *monitoringpb.TimeSeries {
	return &monitoringpb.TimeSeries{
		Metric: &googlemetricpb.Metric{
			Type: fmt.Sprintf("custom.googleapis.com/opencensus/test/metric/%v", i),
			Labels: map[string]string{
				"key": fmt.Sprintf("test_%v", i),
			},
		},
		MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
		ValueType:  googlemetricpb.MetricDescriptor_INT64,
		Points: []*monitoringpb.Point{
			{
				Value: &monitoringpb.TypedValue{
					Value: &monitoringpb.TypedValue_Int64Value{
						Int64Value: int64(i),
					},
				},
			},
		},
	}
}
