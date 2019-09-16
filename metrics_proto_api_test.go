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

package stackdriver_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"

	"strings"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"google.golang.org/api/option"

	labelpb "google.golang.org/genproto/googleapis/api/label"
	googlemetricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"

	sd "contrib.go.opencensus.io/exporter/stackdriver"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"go.opencensus.io/resource/resourcekeys"
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

	// label keys and values
	inEmptyValue = &metricspb.LabelValue{Value: "", HasValue: true}
)

func TestVariousCasesFromFile(t *testing.T) {
	files := []string{
		"ExportLabels",
		"ExportMetricsOfAllTypes",
	}
	for _, file := range files {
		tc := readTestCaseFromFiles(file)
		server, addr, doneFn := createFakeServer(t)
		defer doneFn()

		// Now create a gRPC connection to the agent.
		conn := createConn(t, addr)
		defer conn.Close()

		// Finally create the OpenCensus stats exporter
		se := createExporter(t, conn, defaultOpts)
		executeTestCase(t, tc, se, server, nil)

	}
}

func TestExportMaxTSPerRequest(t *testing.T) {
	server, addr, doneFn := createFakeServer(t)
	defer doneFn()

	// Now create a gRPC connection to the server.
	conn := createConn(t, addr)
	defer conn.Close()

	// Finally create the OpenCensus stats exporter
	se := createExporter(t, conn, defaultOpts)

	tcFromFile := readTestCaseFromFiles("ExportMaxTSPerRequest")

	// update tcFromFile with additional input Time-series and expected Time-Series in
	// CreateTimeSeriesRequest(s). Replicate time-series with different label values.
	for i := 1; i < 250; i++ {
		v := fmt.Sprintf("value_%d", i)
		lv := &metricspb.LabelValue{Value: v, HasValue: true}

		ts := *tcFromFile.inMetric[0].Timeseries[0]
		ts.LabelValues = []*metricspb.LabelValue{inEmptyValue, lv}
		tcFromFile.inMetric[0].Timeseries = append(tcFromFile.inMetric[0].Timeseries, &ts)

		j := i / 200
		outTS := *(tcFromFile.outTSR[0].TimeSeries[0])
		outTS.Metric = &googlemetricpb.Metric{
			Type: tcFromFile.outMDR[0].MetricDescriptor.Type,
			Labels: map[string]string{
				"empty_key":      "",
				"operation_type": v,
			},
		}
		if j > len(tcFromFile.outTSR)-1 {
			newOutTSR := &monitoringpb.CreateTimeSeriesRequest{
				Name: tcFromFile.outTSR[0].Name,
			}
			tcFromFile.outTSR = append(tcFromFile.outTSR, newOutTSR)
		}
		tcFromFile.outTSR[j].TimeSeries = append(tcFromFile.outTSR[j].TimeSeries, &outTS)
	}
	executeTestCase(t, tcFromFile, se, server, nil)
}

func TestExportMaxTSPerRequestAcrossTwoMetrics(t *testing.T) {
	server, addr, doneFn := createFakeServer(t)
	defer doneFn()

	// Now create a gRPC connection to the server.
	conn := createConn(t, addr)
	defer conn.Close()

	// Finally create the OpenCensus stats exporter
	se := createExporter(t, conn, defaultOpts)

	// Read two metrics, one CreateTimeSeriesRequest and two CreateMetricDescriptorRequest.
	tcFromFile := readTestCaseFromFiles("ExportMaxTSPerRequestAcrossTwoMetrics")

	// update tcFromFile with additional input Time-series and expected Time-Series in
	// CreateTimeSeriesRequest(s).
	// Replicate time-series for both metrics.
	for k := 0; k < 2; k++ {
		for i := 1; i < 250; i++ {
			v := fmt.Sprintf("value_%d", i+k*250)
			ts := *tcFromFile.inMetric[k].Timeseries[0]
			lv := &metricspb.LabelValue{Value: v, HasValue: true}
			ts.LabelValues = []*metricspb.LabelValue{inEmptyValue, lv}
			tcFromFile.inMetric[k].Timeseries = append(tcFromFile.inMetric[k].Timeseries, &ts)
		}
	}

	// Replicate time-series in CreateTimeSeriesRequest
	for k := 0; k < 2; k++ {
		for i := 0; i < 250; i++ {
			v := i + k*250
			if v == 0 {
				// skip first TS, it is already there.
				continue
			}
			val := fmt.Sprintf("value_%d", v)

			j := v / 200

			// pick metric-1 for first 250 time-series and metric-2 for next 250 time-series.
			mt := tcFromFile.outMDR[k].MetricDescriptor.Type
			outTS := *(tcFromFile.outTSR[0].TimeSeries[0])
			outTS.Metric = &googlemetricpb.Metric{
				Type: mt,
				Labels: map[string]string{
					"empty_key":      "",
					"operation_type": val,
				},
			}
			if j > len(tcFromFile.outTSR)-1 {
				newOutTSR := &monitoringpb.CreateTimeSeriesRequest{
					Name: tcFromFile.outTSR[0].Name,
				}
				tcFromFile.outTSR = append(tcFromFile.outTSR, newOutTSR)
			}
			tcFromFile.outTSR[j].TimeSeries = append(tcFromFile.outTSR[j].TimeSeries, &outTS)
		}
	}
	executeTestCase(t, tcFromFile, se, server, nil)
}

func createConn(t *testing.T, addr string) *grpc.ClientConn {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to make a gRPC connection to the server: %v", err)
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

func executeTestCase(t *testing.T, tc *testCases, se *sd.Exporter, server *fakeMetricsServer, rsc *resourcepb.Resource) {
	dropped, err := se.PushMetricsProto(context.Background(), nil, rsc, tc.inMetric)
	if dropped != 0 || err != nil {
		t.Fatalf("Name: %s, Error pushing metrics, dropped:%d err:%v", tc.name, dropped, err)
	}

	gotTimeSeries := []*monitoringpb.CreateTimeSeriesRequest{}
	server.forEachStackdriverTimeSeries(func(sdt *monitoringpb.CreateTimeSeriesRequest) {
		gotTimeSeries = append(gotTimeSeries, sdt)
	})

	if diff, idx := requireTimeSeriesRequestEqual(t, gotTimeSeries, tc.outTSR); diff != "" {
		t.Errorf("Name[%s], TimeSeries[%d], Error: -got +want %s\n", tc.name, idx, diff)
	}

	gotCreateMDRequest := []*monitoringpb.CreateMetricDescriptorRequest{}
	server.forEachStackdriverMetricDescriptor(func(sdm *monitoringpb.CreateMetricDescriptorRequest) {
		gotCreateMDRequest = append(gotCreateMDRequest, sdm)
	})

	if diff, idx := requireMetricDescriptorRequestEqual(t, gotCreateMDRequest, tc.outMDR); diff != "" {
		t.Errorf("Name[%s], MetricDescriptor[%d], Error: -got +want %s\n", tc.name, idx, diff)
	}
	server.resetStackdriverMetricDescriptors()
	server.resetStackdriverTimeSeries()
}

func createResourcePbUnknown() *resourcepb.Resource {
	return &resourcepb.Resource{
		Type: "Unknown",
		Labels: map[string]string{
			resourcekeys.K8SKeyClusterName:   "cluster1",
			resourcekeys.K8SKeyPodName:       "pod1",
			resourcekeys.K8SKeyNamespaceName: "namespace1",
			resourcekeys.ContainerKeyName:    "container-name1",
			resourcekeys.CloudKeyZone:        "zone1",
		},
	}
}

func createResourcePbContainer() *resourcepb.Resource {
	return &resourcepb.Resource{
		Type: resourcekeys.ContainerType,
		Labels: map[string]string{
			resourcekeys.K8SKeyClusterName:   "cluster1",
			resourcekeys.K8SKeyPodName:       "pod1",
			resourcekeys.K8SKeyNamespaceName: "namespace1",
			resourcekeys.ContainerKeyName:    "container-name1",
			resourcekeys.CloudKeyZone:        "zone1",
		},
	}
}

func createMonitoredResourcePbK8sContainer() *monitoredrespb.MonitoredResource {
	return &monitoredrespb.MonitoredResource{
		Type: "k8s_container",
		Labels: map[string]string{
			"location":       "zone1",
			"cluster_name":   "cluster1",
			"namespace_name": "namespace1",
			"pod_name":       "pod1",
			"container_name": "container-name1",
		},
	}

}

func createResourcePbK8sPodType() *resourcepb.Resource {
	return &resourcepb.Resource{
		Type: resourcekeys.K8SType,
		Labels: map[string]string{
			resourcekeys.K8SKeyClusterName:   "cluster1",
			resourcekeys.K8SKeyPodName:       "pod1",
			resourcekeys.K8SKeyNamespaceName: "namespace1",
			resourcekeys.ContainerKeyName:    "container-name1",
			resourcekeys.CloudKeyZone:        "zone1",
		},
	}
}

func createMonitoredResourcePbK8sPodType() *monitoredrespb.MonitoredResource {
	return &monitoredrespb.MonitoredResource{
		Type: "k8s_pod",
		Labels: map[string]string{
			"location":       "zone1",
			"cluster_name":   "cluster1",
			"namespace_name": "namespace1",
			"pod_name":       "pod1",
		},
	}

}

func createResourcePbK8sNodType() *resourcepb.Resource {
	return &resourcepb.Resource{
		Type: resourcekeys.HostType,
		Labels: map[string]string{
			resourcekeys.K8SKeyClusterName:   "cluster1",
			resourcekeys.K8SKeyPodName:       "pod1",
			resourcekeys.K8SKeyNamespaceName: "namespace1",
			resourcekeys.ContainerKeyName:    "container-name1",
			resourcekeys.CloudKeyZone:        "zone1",
			resourcekeys.HostKeyName:         "host1",
		},
	}
}

func createMonitoredResourcePbK8sNodType() *monitoredrespb.MonitoredResource {
	return &monitoredrespb.MonitoredResource{
		Type: "k8s_node",
		Labels: map[string]string{
			"location":     "zone1",
			"cluster_name": "cluster1",
			"node_name":    "host1",
		},
	}

}

func createResourcePbGCE() *resourcepb.Resource {
	return &resourcepb.Resource{
		Type: resourcekeys.CloudType,
		Labels: map[string]string{
			resourcekeys.CloudKeyProvider: resourcekeys.CloudProviderGCP,
			resourcekeys.HostKeyID:        "inst1",
			resourcekeys.CloudKeyZone:     "zone1",
		},
	}
}

func createMonitoredResourcePbGCE() *monitoredrespb.MonitoredResource {
	return &monitoredrespb.MonitoredResource{
		Type: "gce_instance",
		Labels: map[string]string{
			"instance_id": "inst1",
			"zone":        "zone1",
		},
	}
}

func createResourcePbAWS() *resourcepb.Resource {
	return &resourcepb.Resource{
		Type: resourcekeys.CloudType,
		Labels: map[string]string{
			resourcekeys.CloudKeyProvider:  resourcekeys.CloudProviderAWS,
			resourcekeys.HostKeyID:         "inst1",
			resourcekeys.CloudKeyRegion:    "region1",
			resourcekeys.CloudKeyAccountID: "account1",
		},
	}
}

func createMonitoredResourcePbAWS() *monitoredrespb.MonitoredResource {
	return &monitoredrespb.MonitoredResource{
		Type: "aws_ec2_instance",
		Labels: map[string]string{
			"instance_id": "inst1",
			"region":      "aws:region1",
			"aws_account": "account1",
		},
	}
}

//func createMetric(md *metricspb.MetricDescriptor, points []*metricspb.Point, labelValues ...*metricspb.LabelValue) *metricspb.Metric {
//	lvs := []*metricspb.LabelValue{}
//	lvs = append(lvs, labelValues...)
//	return &metricspb.Metric{
//		MetricDescriptor: md,
//		Timeseries: []*metricspb.TimeSeries{
//			{
//				StartTimestamp: startTimestamp,
//				LabelValues:    lvs,
//				Points:         points,
//			},
//		},
//	}
//}
//
func createGoogleMetric(md *googlemetricpb.MetricDescriptor, res *monitoredrespb.MonitoredResource, mk googlemetricpb.MetricDescriptor_MetricKind, mt googlemetricpb.MetricDescriptor_ValueType, points []*monitoringpb.Point) *monitoringpb.CreateTimeSeriesRequest {
	return &monitoringpb.CreateTimeSeriesRequest{
		Name: "projects/metrics_proto_test",
		TimeSeries: []*monitoringpb.TimeSeries{
			{
				Metric: &googlemetricpb.Metric{
					Type: md.Type,
					Labels: map[string]string{
						"empty_key":      "",
						"operation_type": "test_1",
					},
				},
				Resource:   res,
				MetricKind: mk,
				ValueType:  mt,
				Points:     points,
			},
		},
	}
}

//func writeToFile(tc *testCases) {
//	inFile, err := os.Create("/tmp/in_" + strings.Replace(tc.name, " ", "_", -1))
//	if err != nil {
//		panic("error opening in file " + tc.name)
//	}
//
//	for _, in := range tc.inMetric {
//		proto.MarshalText(inFile, in)
//		inFile.WriteString("---\n")
//	}
//	inFile.Close()
//
//	outMDFile, err := os.Create("/tmp/outMDR_" + strings.Replace(tc.name, " ", "_", -1))
//	if err != nil {
//		panic("error opening outMD file " + tc.name)
//	}
//
//	for _, outMDR := range tc.outMDR {
//		proto.MarshalText(outMDFile, outMDR)
//		outMDFile.WriteString("---\n")
//	}
//	outMDFile.Close()
//
//	outTSFile, err := os.Create("/tmp/outTSR_" + strings.Replace(tc.name, " ", "_", -1))
//	if err != nil {
//		panic("error opening outTS file " + tc.name)
//	}
//
//	for _, outTSR := range tc.outTSR {
//		proto.MarshalText(outTSFile, outTSR)
//		outTSFile.WriteString("---\n")
//	}
//	outTSFile.Close()
//}

func readTestCaseFromFiles(filename string) *testCases {
	tc := &testCases{
		name: filename,
	}

	// Read input Metrics proto.
	f, err := ioutil.ReadFile("testdata/" + "inMetrics_" + filename + ".txt")
	if err != nil {
		panic("error opening in file " + filename)
	}

	strMetrics := strings.Split(string(f), "---")
	for _, strMetric := range strMetrics {
		in := metricspb.Metric{}
		err = proto.UnmarshalText(strMetric, &in)
		if err != nil {
			panic("error unmarshalling Metric protos from file " + filename)
		}
		tc.inMetric = append(tc.inMetric, &in)
	}

	// Read expected output CreateMetricDescriptorRequest proto.
	f, err = ioutil.ReadFile("testdata/" + "outMDR_" + filename + ".txt")
	if err != nil {
		panic("error opening in file " + filename)
	}

	strOutMDRs := strings.Split(string(f), "---")
	for _, strOutMDR := range strOutMDRs {
		outMDR := monitoringpb.CreateMetricDescriptorRequest{}
		err = proto.UnmarshalText(strOutMDR, &outMDR)
		if err != nil {
			panic("error unmarshalling CreateMetricDescriptorRequest protos from file " + filename)
		}
		tc.outMDR = append(tc.outMDR, &outMDR)
	}

	// Read expected output CreateTimeSeriesRequest proto.
	f, err = ioutil.ReadFile("testdata/" + "outTSR_" + filename + ".txt")
	if err != nil {
		panic("error opening in file " + filename)
	}

	strOutTSRs := strings.Split(string(f), "---")
	for _, strOutTSR := range strOutTSRs {
		outTSR := monitoringpb.CreateTimeSeriesRequest{}
		err = proto.UnmarshalText(strOutTSR, &outTSR)
		if err != nil {
			panic("error unmarshalling CreateTimeSeriesRequest protos from file " + filename)
		}
		tc.outTSR = append(tc.outTSR, &outTSR)
	}
	return tc
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
	_, serverPortStr, _ := net.SplitHostPort(ln.Addr().String())
	return server, "localhost:" + serverPortStr, stop
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
