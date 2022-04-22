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
	"errors"
	"fmt"
	"testing"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	"google.golang.org/api/option"
	googlemetricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	tss := makeTs(500, false) // make 500 time series, should be split to 3 reqs

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
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return monitoring.NewMetricClient(context.Background(), option.WithGRPCConn(conn))
}

// makeTs returns a list of n *monitoringpb.TimeSeries. The metric type (service/non-service)
// is determined by serviceMetric
func makeTs(n int, serviceMetric bool) []*monitoringpb.TimeSeries {
	var tsl []*monitoringpb.TimeSeries
	for i := 0; i < n; i++ {
		metricType := fmt.Sprintf("custom.googleapis.com/opencensus/test/metric/%v", i)
		if serviceMetric {
			metricType = fmt.Sprintf("kubernetes.io/opencensus/test/metric/%v", i)
		}
		tsl = append(tsl, &monitoringpb.TimeSeries{
			Metric: &googlemetricpb.Metric{
				Type: metricType,
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
		})
	}
	return tsl
}

func TestSendReqAndParseDropped(t *testing.T) {
	type testCase struct {
		name                        string
		nonServiceTimeSeriesCount   int
		serviceTimeSeriesCount      int
		createTimeSeriesFunc        func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error
		createServiceTimeSeriesFunc func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error
		expectedErr                 bool
		expectedDropped             int
	}

	testCases := []testCase{
		{
			name:                      "No error",
			serviceTimeSeriesCount:    75,
			nonServiceTimeSeriesCount: 75,
			createTimeSeriesFunc: func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
				return nil
			},
			createServiceTimeSeriesFunc: func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
				return nil
			},
			expectedErr:     false,
			expectedDropped: 0,
		},
		{
			name:                      "Partial error",
			serviceTimeSeriesCount:    75,
			nonServiceTimeSeriesCount: 75,
			createTimeSeriesFunc: func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
				return errors.New("One or more TimeSeries could not be written: Internal error encountered. Please retry after a few seconds. If internal errors persist, contact support at https://cloud.google.com/support/docs.: timeSeries[0-16,25-44,46-74]; Unknown metric: agent.googleapis.com/system.swap.page_faults: timeSeries[45]")
			},
			createServiceTimeSeriesFunc: func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
				return errors.New("One or more TimeSeries could not be written: Internal error encountered. Please retry after a few seconds. If internal errors persist, contact support at https://cloud.google.com/support/docs.: timeSeries[0-16,25-44,46-74]; Unknown metric: agent.googleapis.com/system.swap.page_faults: timeSeries[45]")
			},
			expectedErr:     true,
			expectedDropped: 67 * 2,
		},
		{
			name:                      "Incorrectly formatted error",
			nonServiceTimeSeriesCount: 75,
			serviceTimeSeriesCount:    75,
			createTimeSeriesFunc: func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
				return errors.New("One or more TimeSeries could not be written: Internal error encountered. Please retry after a few seconds. If internal errors persist, contact support at https://cloud.google.com/support/docs.: timeSeries[0-16,25-44,,46-74]; Unknown metric: agent.googleapis.com/system.swap.page_faults: timeSeries[45x]")
			},
			createServiceTimeSeriesFunc: func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
				return nil
			},
			expectedErr:     true,
			expectedDropped: 75,
		},
		{
			name:                      "Unexpected error format",
			nonServiceTimeSeriesCount: 75,
			serviceTimeSeriesCount:    75,
			createTimeSeriesFunc: func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
				return nil
			},
			createServiceTimeSeriesFunc: func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
				return errors.New("err1")
			},
			expectedErr:     true,
			expectedDropped: 75,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			persistedCreateTimeSeries := createTimeSeries
			persistedCreateServiceTimeSeries := createServiceTimeSeries
			createTimeSeries = test.createTimeSeriesFunc
			createServiceTimeSeries = test.createServiceTimeSeriesFunc
			defer func() {
				createTimeSeries = persistedCreateTimeSeries
				createServiceTimeSeries = persistedCreateServiceTimeSeries
			}()

			mc, _ := monitoring.NewMetricClient(context.Background())
			var tsl []*monitoringpb.TimeSeries
			tsl = append(tsl, makeTs(test.serviceTimeSeriesCount, true)...)
			tsl = append(tsl, makeTs(test.nonServiceTimeSeriesCount, false)...)
			d, errors := sendReq(context.Background(), mc, &monitoringpb.CreateTimeSeriesRequest{TimeSeries: tsl})
			if !test.expectedErr && len(errors) > 0 {
				t.Fatalf("Expected no errors, got %v", errors)
			}
			if test.expectedErr && len(errors) == 0 {
				t.Fatalf("Expected errors, got %v", errors)
			}
			if d != test.expectedDropped {
				t.Fatalf("Want %v dropped, got %v", test.expectedDropped, d)
			}
		})
	}
}
