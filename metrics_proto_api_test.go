// Copyright 2019	, OpenCensus Authors
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

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/golang/protobuf/ptypes/wrappers"
	"google.golang.org/api/option"

	distributionpb "google.golang.org/genproto/googleapis/api/distribution"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	googlemetricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
	"google.golang.org/grpc"

	sd "contrib.go.opencensus.io/exporter/stackdriver"
	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
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
	outGaugeInterval = &monitoringpb.TimeInterval{
		EndTime: endTimestamp,
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

	inMetricDescLatency  = "Description of latency"
	outMetricDescLatency = "Description of latency"

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

	// points int64
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
	outPointsGaugeInt64_1 = []*monitoringpb.Point{
		{
			Interval: outGaugeInterval,
			Value:    outValueInt64_1,
		},
	}

	// points int64
	inPointsFloat64_1 = []*metricspb.Point{
		{
			Timestamp: endTimestamp,
			Value:     &metricspb.Point_DoubleValue{DoubleValue: float64(35.5)},
		},
	}
	outValueDouble64_1 = &monitoringpb.TypedValue{
		Value: &monitoringpb.TypedValue_DoubleValue{
			DoubleValue: float64(35.5),
		},
	}
	outPointsDouble64_1 = []*monitoringpb.Point{
		{
			Interval: outInterval,
			Value:    outValueDouble64_1,
		},
	}
	outPointsGaugeDouble64_1 = []*monitoringpb.Point{
		{
			Interval: outGaugeInterval,
			Value:    outValueDouble64_1,
		},
	}

	// Distribution bounds
	inBounds  = []float64{10, 20, 30, 40}
	outBounds = []float64{0, 10, 20, 30, 40}

	// Summary percentile
	inPercentile = []float64{10.0, 50.0, 90.0, 99.0}
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

func TestExportMetricOfDifferentType(t *testing.T) {
	server, addr, doneFn := createFakeServer(t)
	defer doneFn()

	// Now create a gRPC connection to the agent.
	conn := createConn(t, addr)
	defer conn.Close()

	// Finally create the OpenCensus stats exporter
	se := createExporter(t, conn, defaultOpts)

	inMDCummDouble := createMetricDescriptorByType("metricCummDouble", metricspb.MetricDescriptor_CUMULATIVE_DOUBLE)
	outMDCummDouble := createGoogleMetricPbDescriptorByType("metricCummDouble", googlemetricpb.MetricDescriptor_CUMULATIVE, googlemetricpb.MetricDescriptor_DOUBLE)
	inMDGaugeDouble := createMetricDescriptorByType("metricGaugeDouble", metricspb.MetricDescriptor_GAUGE_DOUBLE)
	outMDGaugeDouble := createGoogleMetricPbDescriptorByType("metricGaugeDouble", googlemetricpb.MetricDescriptor_GAUGE, googlemetricpb.MetricDescriptor_DOUBLE)
	inMDCummInt64 := createMetricDescriptorByType("metricCummInt64", metricspb.MetricDescriptor_CUMULATIVE_INT64)
	outMDCummInt64 := createGoogleMetricPbDescriptorByType("metricCummInt64", googlemetricpb.MetricDescriptor_CUMULATIVE, googlemetricpb.MetricDescriptor_INT64)
	inMDGaugeInt64 := createMetricDescriptorByType("metricGaugeInt64", metricspb.MetricDescriptor_GAUGE_INT64)
	outMDGaugeInt64 := createGoogleMetricPbDescriptorByType("metricGaugeInt64", googlemetricpb.MetricDescriptor_GAUGE, googlemetricpb.MetricDescriptor_INT64)

	inMDCummDist := createMetricDescriptorByType("metricCummDist", metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION)
	outMDCummDist := createGoogleMetricPbDescriptorByType("metricCummDist", googlemetricpb.MetricDescriptor_CUMULATIVE, googlemetricpb.MetricDescriptor_DISTRIBUTION)
	inPointsCummDist := createMetricPbPointDistribution(1, 11.9, inBounds, 1, 0, 0, 0)
	outPointsCummDist := createGoogleMetricPbPointDistribution(1, 11.9, startTimestamp, outBounds, 0, 1, 0, 0, 0)

	inMDGuageDist := createMetricDescriptorByType("metricGuageDist", metricspb.MetricDescriptor_GAUGE_DISTRIBUTION)
	outMDGuageDist := createGoogleMetricPbDescriptorByType("metricGuageDist", googlemetricpb.MetricDescriptor_GAUGE, googlemetricpb.MetricDescriptor_DISTRIBUTION)
	inPointsGaugeDist := createMetricPbPointDistribution(1, 11.9, inBounds, 1, 0, 0, 0)
	outPointsGuageDist := createGoogleMetricPbPointDistribution(1, 11.9, nil, outBounds, 0, 1, 0, 0, 0)

	inMDSummary := createMetricDescriptorByType("metricSummary", metricspb.MetricDescriptor_SUMMARY)
	outMDSummaryCount := createGoogleMetricPbDescriptorByType("metricSummary_summary_count", googlemetricpb.MetricDescriptor_CUMULATIVE, googlemetricpb.MetricDescriptor_INT64)
	outMDSummarySum := createGoogleMetricPbDescriptorByType("metricSummary_summary_sum", googlemetricpb.MetricDescriptor_CUMULATIVE, googlemetricpb.MetricDescriptor_DOUBLE)
	outMDSummaryPercentile := createGoogleMetricPbDescriptorByType("metricSummary_summary_percentile", googlemetricpb.MetricDescriptor_GAUGE, googlemetricpb.MetricDescriptor_DOUBLE)

	// Adjust description to original description of summary metris.
	outMDSummaryCount.Description = inMDSummary.Description
	outMDSummarySum.Description = inMDSummary.Description
	outMDSummaryPercentile.Description = inMDSummary.Description
	outMDSummaryCount.Unit = "1"
	lbl := &labelpb.LabelDescriptor{
		Key:         "percentile",
		ValueType:   labelpb.LabelDescriptor_STRING,
		Description: "the value at a given percentile of a distribution",
	}
	outMDSummaryPercentile.Labels = append(outMDSummaryPercentile.Labels, lbl)

	inPointsSummary := createMetricPbPointSummary(10, 119.0, inPercentile, 5.6, 9.6, 12.6, 17.6)
	outPointSummaryCount := createGoogleMetricPbPointInt64(10)
	outPointSummarySum := createGoogleMetricPbPointDouble(119.0, true)
	outPointSummaryPercentile1 := createGoogleMetricPbPointDouble(5.6, false)
	outPointSummaryPercentile2 := createGoogleMetricPbPointDouble(9.6, false)
	outPointSummaryPercentile3 := createGoogleMetricPbPointDouble(12.6, false)
	outPointSummaryPercentile4 := createGoogleMetricPbPointDouble(17.6, false)

	tcs := []*testCases{
		{
			name: "Metrics of different types",
			inMetric: []*metricspb.Metric{
				{
					MetricDescriptor: inMDCummDouble,
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inOperTypeValue1},
							Points:         inPointsFloat64_1,
						},
					},
				},
				{
					MetricDescriptor: inMDGaugeDouble,
					Timeseries: []*metricspb.TimeSeries{
						{
							LabelValues: []*metricspb.LabelValue{inEmptyValue, inOperTypeValue2},
							Points:      inPointsFloat64_1,
						},
					},
				},
				{
					MetricDescriptor: inMDCummInt64,
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inOperTypeValue1},
							Points:         inPointsInt64_1,
						},
					},
				},
				{
					MetricDescriptor: inMDGaugeInt64,
					Timeseries: []*metricspb.TimeSeries{
						{
							LabelValues: []*metricspb.LabelValue{inEmptyValue, inOperTypeValue2},
							Points:      inPointsInt64_1,
						},
					},
				},
				{
					MetricDescriptor: inMDCummDist,
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inOperTypeValue1},
							Points:         []*metricspb.Point{inPointsCummDist},
						},
					},
				},
				{
					MetricDescriptor: inMDGuageDist,
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inOperTypeValue1},
							Points:         []*metricspb.Point{inPointsGaugeDist},
						},
					},
				},
				{
					MetricDescriptor: inMDSummary,
					Timeseries: []*metricspb.TimeSeries{
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inOperTypeValue1},
							Points:         []*metricspb.Point{inPointsSummary},
						},
						//  Add another time series to test https://github.com/census-ecosystem/opencensus-go-exporter-stackdriver/pull/214
						{
							StartTimestamp: startTimestamp,
							LabelValues:    []*metricspb.LabelValue{inEmptyValue, inOperTypeValue2},
							Points:         []*metricspb.Point{inPointsSummary},
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
								Type: outMDCummDouble.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     outPointsDouble64_1,
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDGaugeDouble.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     outPointsGaugeDouble64_1,
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDCummInt64.Type,
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
								Type: outMDGaugeInt64.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_INT64,
							Points:     outPointsGaugeInt64_1,
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDCummDist.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_DISTRIBUTION,
							Points:     []*monitoringpb.Point{outPointsCummDist},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDGuageDist.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DISTRIBUTION,
							Points:     []*monitoringpb.Point{outPointsGuageDist},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummarySum.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummarySum},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryCount.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_INT64,
							Points:     []*monitoringpb.Point{outPointSummaryCount},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryPercentile.Type,
								Labels: map[string]string{
									"percentile":     "10.000000",
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummaryPercentile1},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryPercentile.Type,
								Labels: map[string]string{
									"percentile":     "50.000000",
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummaryPercentile2},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryPercentile.Type,
								Labels: map[string]string{
									"percentile":     "90.000000",
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummaryPercentile3},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryPercentile.Type,
								Labels: map[string]string{
									"percentile":     "99.000000",
									"empty_key":      "",
									"operation_type": "test_1",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummaryPercentile4},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummarySum.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummarySum},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryCount.Type,
								Labels: map[string]string{
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_CUMULATIVE,
							ValueType:  googlemetricpb.MetricDescriptor_INT64,
							Points:     []*monitoringpb.Point{outPointSummaryCount},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryPercentile.Type,
								Labels: map[string]string{
									"percentile":     "10.000000",
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummaryPercentile1},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryPercentile.Type,
								Labels: map[string]string{
									"percentile":     "50.000000",
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummaryPercentile2},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryPercentile.Type,
								Labels: map[string]string{
									"percentile":     "90.000000",
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummaryPercentile3},
						},
						{
							Metric: &googlemetricpb.Metric{
								Type: outMDSummaryPercentile.Type,
								Labels: map[string]string{
									"percentile":     "99.000000",
									"empty_key":      "",
									"operation_type": "test_2",
								},
							},
							Resource:   outGlobalResource,
							MetricKind: googlemetricpb.MetricDescriptor_GAUGE,
							ValueType:  googlemetricpb.MetricDescriptor_DOUBLE,
							Points:     []*monitoringpb.Point{outPointSummaryPercentile4},
						},
					},
				},
			},
			outMDR: []*monitoringpb.CreateMetricDescriptorRequest{
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDCummDouble,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDGaugeDouble,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDCummInt64,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDGaugeInt64,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDCummDist,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDGuageDist,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDSummarySum,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDSummaryCount,
				},
				{
					Name:             "projects/metrics_proto_test",
					MetricDescriptor: outMDSummaryPercentile,
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
		t.Errorf("Name[%s], TimeSeries[%d], Error: -got +want %s\n", tc.name, idx, diff)
	}

	var gotCreateMDRequest []*monitoringpb.CreateMetricDescriptorRequest
	server.forEachStackdriverMetricDescriptor(func(sdm *monitoringpb.CreateMetricDescriptorRequest) {
		gotCreateMDRequest = append(gotCreateMDRequest, sdm)
	})

	if diff, idx := requireMetricDescriptorRequestEqual(t, gotCreateMDRequest, tc.outMDR); diff != "" {
		t.Errorf("Name[%s], MetricDescriptor[%d], Error: -got +want %s\n", tc.name, idx, diff)
	}
	server.resetStackdriverMetricDescriptors()
	server.resetStackdriverTimeSeries()
}

func createMetricDescriptorByType(metricName string, mt metricspb.MetricDescriptor_Type) *metricspb.MetricDescriptor {
	inMetricName := "ocagent.io/" + metricName

	inMetricDesc := "Description of " + metricName

	metricUnit := "ms"

	inMD := createMetricPbDescriptor(inMetricName,
		inMetricDesc,
		metricUnit,
		mt,
		inEmptyKey,
		inOperTypeKey)

	return inMD
}

func createGoogleMetricPbDescriptorByType(metricName string, mk googlemetricpb.MetricDescriptor_MetricKind, mt googlemetricpb.MetricDescriptor_ValueType) *googlemetricpb.MetricDescriptor {
	inMetricName := "ocagent.io/" + metricName
	outCreateMDName := "projects/" + projectID + "/metricDescriptors/custom.googleapis.com/opencensus/" + inMetricName
	outMetricType := "custom.googleapis.com/opencensus/" + inMetricName
	outDisplayName := "OpenCensus/" + inMetricName

	outMetricDesc := "Description of " + metricName

	outMetricUnit := "ms"

	outMD := createGoogleMetricPbDescriptor(
		outCreateMDName,
		outMetricType,
		outMetricDesc,
		outDisplayName,
		outMetricUnit,
		mk,
		mt,
		"empty_key",
		"operation_type")
	return outMD
}

func createMetricPbPointDistribution(count int64, sum float64, bounds []float64, bktCounts ...int64) *metricspb.Point {
	buckets := []*metricspb.DistributionValue_Bucket{}
	for _, count := range bktCounts {
		bucket := &metricspb.DistributionValue_Bucket{
			Count: count,
		}
		buckets = append(buckets, bucket)
	}
	return &metricspb.Point{
		Timestamp: endTimestamp,
		Value: &metricspb.Point_DistributionValue{
			DistributionValue: &metricspb.DistributionValue{
				Count:                 count,
				Sum:                   sum,
				SumOfSquaredDeviation: 0,
				Buckets:               buckets,
				BucketOptions: &metricspb.DistributionValue_BucketOptions{
					Type: &metricspb.DistributionValue_BucketOptions_Explicit_{
						Explicit: &metricspb.DistributionValue_BucketOptions_Explicit{
							// Without zero bucket in
							Bounds: bounds,
						},
					},
				},
			},
		},
	}
}

func createGoogleMetricPbPointDistribution(count int64, mean float64, st *timestamp.Timestamp, bounds []float64, bktCounts ...int64) *monitoringpb.Point {
	return &monitoringpb.Point{
		Interval: &monitoringpb.TimeInterval{
			StartTime: st,
			EndTime:   endTimestamp,
		},
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DistributionValue{
				DistributionValue: &distributionpb.Distribution{
					Count:                 count,
					Mean:                  mean,
					SumOfSquaredDeviation: 0,
					BucketCounts:          bktCounts,
					BucketOptions: &distributionpb.Distribution_BucketOptions{
						Options: &distributionpb.Distribution_BucketOptions_ExplicitBuckets{
							ExplicitBuckets: &distributionpb.Distribution_BucketOptions_Explicit{
								Bounds: bounds,
							},
						},
					},
				},
			},
		},
	}
}

func createMetricPbPointSummary(count int64, sum float64, percentile []float64, values ...float64) *metricspb.Point {
	valAtPercentiles := []*metricspb.SummaryValue_Snapshot_ValueAtPercentile{}
	for i, value := range values {
		valAtPercentile := &metricspb.SummaryValue_Snapshot_ValueAtPercentile{
			Value:      value,
			Percentile: percentile[i],
		}
		valAtPercentiles = append(valAtPercentiles, valAtPercentile)
	}

	return &metricspb.Point{
		Timestamp: endTimestamp,
		Value: &metricspb.Point_SummaryValue{
			SummaryValue: &metricspb.SummaryValue{
				Count: &wrappers.Int64Value{Value: count},
				Sum:   &wrappers.DoubleValue{Value: sum},
				Snapshot: &metricspb.SummaryValue_Snapshot{
					PercentileValues: valAtPercentiles,
				},
			},
		},
	}
}

func createGoogleMetricPbPointInt64(value int64) *monitoringpb.Point {
	return &monitoringpb.Point{
		Interval: outInterval,
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_Int64Value{
				Int64Value: value,
			},
		},
	}
}

func createGoogleMetricPbPointDouble(value float64, includeStartTime bool) *monitoringpb.Point {
	interval := outInterval
	if includeStartTime == false {
		interval = outGaugeInterval
	}
	return &monitoringpb.Point{
		Interval: interval,
		Value: &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DoubleValue{
				DoubleValue: value,
			},
		},
	}
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
