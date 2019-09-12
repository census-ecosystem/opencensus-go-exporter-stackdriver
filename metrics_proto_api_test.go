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

package stackdriver_test

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	//resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/option"

	//distributionpb "google.golang.org/genproto/googleapis/api/distribution"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	googlemetricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"

	sd "contrib.go.opencensus.io/exporter/stackdriver"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	//"github.com/golang/protobuf/ptypes/wrappers"
	//"go.opencensus.io/resource/resourcekeys"
)

type testCases struct {
	name     string
	inMetric []*metricspb.Metric
	outTSR   []*monitoringpb.CreateTimeSeriesRequest
	outMDR   []*monitoringpb.CreateMetricDescriptorRequest
}

var (
	// project
	projectID = "metrics_proto_test"

	// default exporter options
	defaultOpts = sd.Options{
		ProjectID: projectID,
		// Set empty labels to avoid the opencensus-task
		DefaultMonitoringLabels: &sd.Labels{},
	}

	// resources
	outGlobalResource = &monitoredrespb.MonitoredResource{
		Type: "global",
	}

	// timestamps
	startTimestamp = &timestamp.Timestamp{
		Seconds: 1543160298,
		Nanos:   100000090,
	}
	endTimestamp = &timestamp.Timestamp{
		Seconds: 1543160298,
		Nanos:   100000997,
	}
	outInterval = &monitoringpb.TimeInterval{
		StartTime: startTimestamp,
		EndTime:   endTimestamp,
	}

	// label keys and values
	inEmptyKey       = &metricspb.LabelKey{Key: "empty_key"}
	inOperTypeKey    = &metricspb.LabelKey{Key: "operation_type"}
	inNoValue        = &metricspb.LabelValue{Value: "", HasValue: false}
	inEmptyValue     = &metricspb.LabelValue{Value: "", HasValue: true}
	inOperTypeValue1 = &metricspb.LabelValue{Value: "test_1", HasValue: true}
	inOperTypeValue2 = &metricspb.LabelValue{Value: "test_2", HasValue: true}

	// Metric Descriptor
	inMetricNameCalls    = "ocagent.io/calls"
	outCreateMDNameCalls = "projects/" + projectID + "/metricDescriptors/custom.googleapis.com/opencensus/" + inMetricNameCalls
	outMetricTypeCalls   = "custom.googleapis.com/opencensus/" + inMetricNameCalls
	outDisplayNameCalls  = "OpenCensus/" + inMetricNameCalls

	inMetricDescCalls  = "Number of calls"
	outMetricDescCalls = "Number of calls"

	metricUnitCalls    = "1"
	outMetricUnitCalls = "1"

	inMDCall = createMetricPbDescriptor(inMetricNameCalls,
		inMetricDescCalls,
		metricUnitCalls,
		metricspb.MetricDescriptor_CUMULATIVE_INT64,
		inEmptyKey,
		inOperTypeKey)
	outMDCall = createGoogleMetricPbDescriptor(
		outCreateMDNameCalls,
		outMetricTypeCalls,
		outMetricDescCalls,
		outDisplayNameCalls,
		outMetricUnitCalls,
		googlemetricpb.MetricDescriptor_CUMULATIVE,
		googlemetricpb.MetricDescriptor_INT64,
		"empty_key",
		"operation_type")

	inMetricNameLatency    = "ocagent.io/latency"
	outCreateMDNameLatency = "projects/" + projectID + "/metricDescriptors/custom.googleapis.com/opencensus/" + inMetricNameLatency
	outMetricTypeLatency   = "custom.googleapis.com/opencensus/" + inMetricNameLatency
	outDisplayNameLatency  = "OpenCensus/" + inMetricNameLatency

	inMetricDescLatency  = "Sum of latency"
	outMetricDescLatency = "Sum of latency"

	metricUnitLatency    = "ms"
	outMetricUnitLatency = "ms"

	inMDLatency = createMetricPbDescriptor(inMetricNameLatency,
		inMetricDescLatency,
		metricUnitLatency,
		metricspb.MetricDescriptor_CUMULATIVE_INT64,
		inEmptyKey,
		inOperTypeKey)
	outMDLatency = createGoogleMetricPbDescriptor(
		outCreateMDNameLatency,
		outMetricTypeLatency,
		outMetricDescLatency,
		outDisplayNameLatency,
		outMetricUnitLatency,
		googlemetricpb.MetricDescriptor_CUMULATIVE,
		googlemetricpb.MetricDescriptor_INT64,
		"empty_key",
		"operation_type")

	// points
	inPointsInt64_1 = []*metricspb.Point{
		{
			Timestamp: endTimestamp,
			Value:     &metricspb.Point_Int64Value{Int64Value: int64(1)},
		},
	}
	outValueInt64_1 = &monitoringpb.TypedValue{
		Value: &monitoringpb.TypedValue_Int64Value{
			Int64Value: 1,
		},
	}
	outPointsInt64_1 = []*monitoringpb.Point{
		{
			Interval: outInterval,
			Value:    outValueInt64_1,
		},
	}
)

func TestExportLabels(t *testing.T) {
	server, addr, doneFn := createFakeServer(t)
	defer doneFn()

	// Now create a gRPC connection to the agent.
	conn := createConn(t, addr)
	defer conn.Close()

	// Finally create the OpenCensus stats exporter
	se := createExporter(t, conn, defaultOpts)

	tcs := []*testCases{
		{
			name: "Label Test: [empty,v1], [empty,v2], [noValue,v1], [empty,noValue]",
			inMetric: []*metricspb.Metric{
				{
					MetricDescriptor: inMDCall,
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inOperTypeValue1},
							Points:         inPointsInt64_1,
						},
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inOperTypeValue2},
							Points:         inPointsInt64_1,
						},
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inNoValue, inOperTypeValue1},
							Points:         inPointsInt64_1,
						},
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inNoValue},
							Points:         inPointsInt64_1,
						},
					},
				},
			},
			outTSR: []*monitoringpb.CreateTimeSeriesRequest{
				{
					Name: "projects/metrics_proto_test",
					TimeSeries: []*monitoringpb.TimeSeries{
						{
							Metric: &googlemetricpb.Metric{
								Type: outMetricTypeCalls,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_INT64,
							Points:     outPointsInt64_1,
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMetricTypeCalls,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_INT64,
							Points:     outPointsInt64_1,
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMetricTypeCalls,
								Labels: map[string]string{
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_INT64,
							Points:     outPointsInt64_1,
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMetricTypeCalls,
								Labels: map[string]string{
									"empty_key": "",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_INT64,
							Points:     outPointsInt64_1,
						},
					},
				},
			},
			outMDR: []*monitoringpb.CreateMetricDescriptorRequest{
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDCall,
				},
			},
		},
	}
	for _, tc := range tcs {
		executeTestCase(t, tc, se, server)
	}
}

func TestExportMaxTSPerRequest(t *testing.T) {
	server, addr, doneFn := createFakeServer(t)
	defer doneFn()

	// Now create a gRPC connection to the agent.
	conn := createConn(t, addr)
	defer conn.Close()

	// Finally create the OpenCensus stats exporter
	se := createExporter(t, conn, defaultOpts)

	tcs := []*testCases{
		{
			name: "Metric with 250 TimeSeries, splits into 2 requests",
			inMetric: []*metricspb.Metric{
				{
					MetricDescriptor: inMDCall,
					Timeseries:       []*metricspb.TimeSeries{},
				},
			},
			outTSR: []*monitoringpb.CreateTimeSeriesRequest{
				{
					Name:       "projects/metrics_proto_test",
					TimeSeries: []*monitoringpb.TimeSeries{},
				},
				{
					Name:       "projects/metrics_proto_test",
					TimeSeries: []*monitoringpb.TimeSeries{},
				},
			},
			outMDR: []*monitoringpb.CreateMetricDescriptorRequest{
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDCall,
				},
			},
		},
	}
	for i := 0; i < 250; i++ {
		v := fmt.Sprintf("value_%d", i)
		lv := &metricspb.LabelValue{Value: v, HasValue: true}
		ts := &metricspb.TimeSeries{
			StartTimestamp: startTimestamp,
			LabelValues:    []*metricspb.LabelValue{inEmptyValue, lv},
			Points:         inPointsInt64_1,
		}
		tcs[0].inMetric[0].Timeseries = append(tcs[0].inMetric[0].Timeseries, ts)

		j := i / 200
		outTS := &monitoringpb.TimeSeries{
			Metric: &googlemetricpb.Metric{
				Type: outMetricTypeCalls,
				Labels: map[string]string{
					"empty_key":      "",
					"operation_type": v,
				},
			},
			Resource:   outGlobalResource,
			MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
			ValueType:  googlemetricpb.MetricDescriptor_INT64,
			Points:     outPointsInt64_1,
		}
		tcs[0].outTSR[j].TimeSeries = append(tcs[0].outTSR[j].TimeSeries, outTS)
	}
	for _, tc := range tcs {
		executeTestCase(t, tc, se, server)
	}
}

func TestExportMaxTSPerRequestAcrossTwoMetrics(t *testing.T) {
	server, addr, doneFn := createFakeServer(t)
	defer doneFn()

	// Now create a gRPC connection to the agent.
	conn := createConn(t, addr)
	defer conn.Close()

	// Finally create the OpenCensus stats exporter
	se := createExporter(t, conn, defaultOpts)

	tcs := []*testCases{
		{
			name: "Two Metric with 250 TimeSeries each, splits into 3 TS requests and 2 MD request",
			inMetric: []*metricspb.Metric{
				{
					MetricDescriptor: inMDCall,
					Timeseries:       []*metricspb.TimeSeries{},
				},
				{
					MetricDescriptor: inMDLatency,
					Timeseries:       []*metricspb.TimeSeries{},
				},
			},
			outTSR: []*monitoringpb.CreateTimeSeriesRequest{
				{
					Name:       "projects/metrics_proto_test",
					TimeSeries: []*monitoringpb.TimeSeries{},
				},
				{
					Name:       "projects/metrics_proto_test",
					TimeSeries: []*monitoringpb.TimeSeries{},
				},
				{
					Name:       "projects/metrics_proto_test",
					TimeSeries: []*monitoringpb.TimeSeries{},
				},
			},
			outMDR: []*monitoringpb.CreateMetricDescriptorRequest{
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDCall,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDLatency,
				},
			},
		},
	}
	for i := 0; i < 500; i++ {
		k := i / 250
		v := fmt.Sprintf("value_%d", i)
		lv := &metricspb.LabelValue{Value: v, HasValue: true}
		ts := &metricspb.TimeSeries{
			StartTimestamp: startTimestamp,
			LabelValues:    []*metricspb.LabelValue{inEmptyValue, lv},
			Points:         inPointsInt64_1,
		}
		tcs[0].inMetric[k].Timeseries = append(tcs[0].inMetric[k].Timeseries, ts)

		j := i / 200
		mt := outMetricTypeCalls
		if k == 1 {
			// TimeSeries Belongs to Latency
			mt = outMetricTypeLatency
		}
		outTS := &monitoringpb.TimeSeries{
			Metric: &googlemetricpb.Metric{
				Type: mt,
				Labels: map[string]string{
					"empty_key":      "",
					"operation_type": v,
				},
			},
			Resource:   outGlobalResource,
			MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
			ValueType:  googlemetricpb.MetricDescriptor_INT64,
			Points:     outPointsInt64_1,
		}
		tcs[0].outTSR[j].TimeSeries = append(tcs[0].outTSR[j].TimeSeries, outTS)
	}
	for _, tc := range tcs {
		executeTestCase(t, tc, se, server)
	}
}

func createConn(t *testing.T, addr string) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to make a gRPC connection to the agent: %v", err)
	}
	return conn
}

func createExporter(t *testing.T, conn *grpc.ClientConn, opts sd.Options) *sd.Exporter {
	opts.MonitoringClientOptions = []option.ClientOption{option.WithGRPCConn(conn)}
	se, err := sd.NewExporter(opts)
	if err != nil {
		t.Fatalf("Failed to create the statsExporter: %v", err)
	}
	return se
}

func executeTestCase(t *testing.T, tc *testCases, se *sd.Exporter, server *fakeMetricsServer) {
	dropped, err := se.PushMetricsProto(context.Background(), nil, nil, tc.inMetric)
	if dropped != 0 || err != nil {
		t.Fatalf("Name: %s, Error pushing metrics, dropped:%d err:%v", tc.name, dropped, err)
	}

	var gotTimeSeries []*monitoringpb.CreateTimeSeriesRequest
	server.forEachStackdriverTimeSeries(func(sdt *monitoringpb.CreateTimeSeriesRequest) {
		gotTimeSeries = append(gotTimeSeries, sdt)
	})

	if diff, idx := requireTimeSeriesRequestEqual(t, gotTimeSeries, tc.outTSR); diff != "" {
		t.Errorf("Name[%s], TimeSeries[%d], Error: %s", tc.name, idx, diff)
	}

	var gotCreateMDRequest []*monitoringpb.CreateMetricDescriptorRequest
	server.forEachStackdriverMetricDescriptor(func(sdm *monitoringpb.CreateMetricDescriptorRequest) {
		gotCreateMDRequest = append(gotCreateMDRequest, sdm)
	})

	if diff, idx := requireMetricDescriptorRequestEqual(t, gotCreateMDRequest, tc.outMDR); diff != "" {
		t.Errorf("Name[%s], MetricDescriptor[%d], Error: %s", tc.name, idx, diff)
	}
	server.resetStackdriverMetricDescriptors()
	server.resetStackdriverTimeSeries()
}

func createMetricPbDescriptor(name, desc, unit string, mt metricspb.MetricDescriptor_Type, lblKeys ...*metricspb.LabelKey) *metricspb.MetricDescriptor {
	return &metricspb.MetricDescriptor{
		Name:        name,
		Description: desc,
		LabelKeys:   lblKeys,
		Unit:        unit,
		Type:        mt,
	}
}

func createGoogleMetricPbDescriptor(name, metricType, desc, disp, unit string, mk googlemetricpb.MetricDescriptor_MetricKind, vt googlemetricpb.MetricDescriptor_ValueType, lblKeys ...string) *googlemetricpb.MetricDescriptor {
	lbls := make([]*labelpb.LabelDescriptor, 0)
	for _, k := range lblKeys {
		lbl := &labelpb.LabelDescriptor{
			Key:       k,
			ValueType: labelpb.LabelDescriptor_STRING,
		}
		lbls = append(lbls, lbl)
	}

	return &googlemetricpb.MetricDescriptor{
		Name:        name,
		Type:        metricType,
		Labels:      lbls,
		MetricKind:  mk,
		ValueType:   vt,
		Unit:        unit,
		Description: desc,
		DisplayName: disp,
	}
}

type fakeMetricsServer struct {
	monitoringpb.MetricServiceServer
	mu                           sync.RWMutex
	stackdriverTimeSeries        []*monitoringpb.CreateTimeSeriesRequest
	stackdriverMetricDescriptors []*monitoringpb.CreateMetricDescriptorRequest
}

func createFakeServer(t *testing.T) (*fakeMetricsServer, string, func()) {
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to bind to an available address: %v", err)
	}
	server := new(fakeMetricsServer)
	srv := grpc.NewServer()
	monitoringpb.RegisterMetricServiceServer(srv, server)
	go func() {
		_ = srv.Serve(ln)
	}()
	stop := func() {
		srv.Stop()
		_ = ln.Close()
	}
	_, agentPortStr, _ := net.SplitHostPort(ln.Addr().String())
	return server, "localhost:" + agentPortStr, stop
}

func (server *fakeMetricsServer) forEachStackdriverTimeSeries(fn func(sdt *monitoringpb.CreateTimeSeriesRequest)) {
	server.mu.RLock()
	defer server.mu.RUnlock()

	for _, sdt := range server.stackdriverTimeSeries {
		fn(sdt)
	}
}

func (server *fakeMetricsServer) forEachStackdriverMetricDescriptor(fn func(sdmd *monitoringpb.CreateMetricDescriptorRequest)) {
	server.mu.RLock()
	defer server.mu.RUnlock()

	for _, sdmd := range server.stackdriverMetricDescriptors {
		fn(sdmd)
	}
}

func (server *fakeMetricsServer) resetStackdriverTimeSeries() {
	server.mu.Lock()
	server.stackdriverTimeSeries = server.stackdriverTimeSeries[:0]
	server.mu.Unlock()
}

func (server *fakeMetricsServer) resetStackdriverMetricDescriptors() {
	server.mu.Lock()
	server.stackdriverMetricDescriptors = server.stackdriverMetricDescriptors[:0]
	server.mu.Unlock()
}

func (server *fakeMetricsServer) CreateMetricDescriptor(ctx context.Context, req *monitoringpb.CreateMetricDescriptorRequest) (*googlemetricpb.MetricDescriptor, error) {
	server.mu.Lock()
	server.stackdriverMetricDescriptors = append(server.stackdriverMetricDescriptors, req)
	server.mu.Unlock()
	return req.MetricDescriptor, nil
}

func (server *fakeMetricsServer) CreateTimeSeries(ctx context.Context, req *monitoringpb.CreateTimeSeriesRequest) (*empty.Empty, error) {
	server.mu.Lock()
	server.stackdriverTimeSeries = append(server.stackdriverTimeSeries, req)
	server.mu.Unlock()
	return new(empty.Empty), nil
}

func requireTimeSeriesRequestEqual(t *testing.T, got, want []*monitoringpb.CreateTimeSeriesRequest) (string, int) {
	diff := ""
	i := 0
	if len(got) != len(want) {
		diff = fmt.Sprintf("Unexpected slice len got: %d want: %d", len(got), len(want))
		return diff, i
	}
	for i, g := range got {
		w := want[i]
		diff = cmp.Diff(g, w, cmpopts.IgnoreFields(timestamp.Timestamp{}, "XXX_sizecache"))
		if diff != "" {
			return diff, i
		}
	}
	return diff, i
}

func requireMetricDescriptorRequestEqual(t *testing.T, got, want []*monitoringpb.CreateMetricDescriptorRequest) (string, int) {
	diff := ""
	i := 0
	if len(got) != len(want) {
		diff = fmt.Sprintf("Unexpected slice len got: %d want: %d", len(got), len(want))
		return diff, i
	}
	for i, g := range got {
		w := want[i]
		diff = cmp.Diff(g, w,
			cmpopts.IgnoreFields(labelpb.LabelDescriptor{}, "XXX_sizecache"),
			cmpopts.IgnoreFields(googlemetricpb.MetricDescriptor{}, "XXX_sizecache"))
		if diff != "" {
			return diff, i
		}
	}
	return diff, i
}
