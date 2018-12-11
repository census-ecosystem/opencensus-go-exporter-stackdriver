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

package stackdriver

/*
The code in this file is responsible for converting OpenCensus Proto metrics
directly to Stackdriver Metrics.
*/

import (
	"errors"
	"fmt"
	"path"

	"github.com/golang/protobuf/ptypes/timestamp"

	distributionpb "google.golang.org/genproto/googleapis/api/distribution"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	googlemetricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoredrespb "google.golang.org/genproto/googleapis/api/monitoredres"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"

	metricspb "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	resourcepb "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
)

var errNilMetric = errors.New("expecting a non-nil metric")

func (se *statsExporter) protoToMonitoringMetricDescriptor(metric *metricspb.Metric) (*googlemetricpb.MetricDescriptor, error) {
	if metric == nil {
		return nil, errNilMetric
	}

	metricName, description, unit, _ := metricProseFromProto(metric)
	metricType, _ := se.metricTypeFromProto(metricName)
	displayName := se.displayName(metricName)
	metricKind, valueType := protoMetricDescriptorTypeToMetricKind(metric)

	sdm := &googlemetricpb.MetricDescriptor{
		Name:        fmt.Sprintf("projects/%s/metricDescriptors/%s", se.o.ProjectID, metricType),
		DisplayName: displayName,
		Description: description,
		Unit:        unit,
		Type:        metricType,
		MetricKind:  metricKind,
		ValueType:   valueType,
		Labels:      labelDescriptorsFromProto(se.defaultLabels, metric.GetMetricDescriptor().GetLabelKeys()),
	}

	return sdm, nil
}

func labelDescriptorsFromProto(defaults map[string]labelValue, protoLabelKeys []*metricspb.LabelKey) []*labelpb.LabelDescriptor {
	labelDescriptors := make([]*labelpb.LabelDescriptor, 0, len(defaults)+len(protoLabelKeys))

	// Fill in the defaults first.
	for key, lbl := range defaults {
		labelDescriptors = append(labelDescriptors, &labelpb.LabelDescriptor{
			Key:         sanitize(key),
			Description: lbl.desc,
			ValueType:   labelpb.LabelDescriptor_STRING,
		})
	}

	// Now fill in those from the metric.
	for _, protoKey := range protoLabelKeys {
		labelDescriptors = append(labelDescriptors, &labelpb.LabelDescriptor{
			Key:         sanitize(protoKey.GetKey()),
			Description: protoKey.GetDescription(),
			ValueType:   labelpb.LabelDescriptor_STRING, // We only use string tags
		})
	}
	return labelDescriptors
}

func metricProseFromProto(metric *metricspb.Metric) (name, description, unit string, ok bool) {
	mname := metric.GetName()
	if mname != "" {
		name = mname
		return
	}

	md := metric.GetMetricDescriptor()

	name = md.GetName()
	unit = md.GetUnit()
	description = md.GetDescription()

	return
}

func (se *statsExporter) metricTypeFromProto(name string) (string, bool) {
	// TODO: (@odeke-em) support non-"custom.googleapis.com" metrics names.
	name = path.Join("custom.googleapis.com", "opencensus", name)
	return name, true
}

func fromProtoPoint(startTime *timestamp.Timestamp, pt *metricspb.Point) (*monitoringpb.Point, error) {
	if pt == nil {
		return nil, nil
	}

	mptv, err := protoToMetricPoint(pt.Value)
	if err != nil {
		return nil, err
	}

	mpt := &monitoringpb.Point{
		Value: mptv,
		Interval: &monitoringpb.TimeInterval{
			StartTime: startTime,
			EndTime:   pt.Timestamp,
		},
	}
	return mpt, nil
}

func protoToMetricPoint(value interface{}) (*monitoringpb.TypedValue, error) {
	if value == nil {
		return nil, nil
	}

	var err error
	var tval *monitoringpb.TypedValue
	switch v := value.(type) {
	default:
		// All the other types are not yet handled.
		// TODO: (@odeke-em, @songy23) talk to the Stackdriver team to determine
		// the use cases for:
		//
		//      *TypedValue_BoolValue
		//      *TypedValue_StringValue
		//
		// and then file feature requests on OpenCensus-Specs and then OpenCensus-Proto,
		// lest we shall error here.
		//
		// TODO: Add conversion from SummaryValue when
		//      https://github.com/census-ecosystem/opencensus-go-exporter-stackdriver/issues/66
		// has been figured out.
		err = fmt.Errorf("protoToMetricPoint: unknown Data type: %T", value)

	case *metricspb.Point_Int64Value:
		tval = &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_Int64Value{
				Int64Value: v.Int64Value,
			},
		}

	case *metricspb.Point_DoubleValue:
		tval = &monitoringpb.TypedValue{
			Value: &monitoringpb.TypedValue_DoubleValue{
				DoubleValue: v.DoubleValue,
			},
		}

	case *metricspb.Point_DistributionValue:
		dv := v.DistributionValue
		var mv *monitoringpb.TypedValue_DistributionValue
		if dv != nil {
			var mean float64
			if dv.Count > 0 {
				mean = float64(dv.Sum) / float64(dv.Count)
			}
			mv = &monitoringpb.TypedValue_DistributionValue{
				DistributionValue: &distributionpb.Distribution{
					Count:                 dv.Count,
					Mean:                  mean,
					SumOfSquaredDeviation: dv.SumOfSquaredDeviation,
					BucketCounts:          bucketCounts(dv.Buckets),
				},
			}

			if bopts := dv.BucketOptions; bopts != nil && bopts.Type != nil {
				bexp, ok := bopts.Type.(*metricspb.DistributionValue_BucketOptions_Explicit_)
				if ok && bexp != nil && bexp.Explicit != nil {
					mv.DistributionValue.BucketOptions = &distributionpb.Distribution_BucketOptions{
						Options: &distributionpb.Distribution_BucketOptions_ExplicitBuckets{
							ExplicitBuckets: &distributionpb.Distribution_BucketOptions_Explicit{
								Bounds: bexp.Explicit.Bounds[:],
							},
						},
					}
				}
			}
		}
		tval = &monitoringpb.TypedValue{Value: mv}
	}

	return tval, err
}

func bucketCounts(buckets []*metricspb.DistributionValue_Bucket) []int64 {
	bucketCounts := make([]int64, len(buckets))
	for i, bucket := range buckets {
		if bucket != nil {
			bucketCounts[i] = bucket.Count
		}
	}
	return bucketCounts
}

func protoMetricDescriptorTypeToMetricKind(m *metricspb.Metric) (googlemetricpb.MetricDescriptor_MetricKind, googlemetricpb.MetricDescriptor_ValueType) {
	dt := m.GetMetricDescriptor()
	if dt == nil {
		return googlemetricpb.MetricDescriptor_METRIC_KIND_UNSPECIFIED, googlemetricpb.MetricDescriptor_VALUE_TYPE_UNSPECIFIED
	}

	switch dt.Type {
	case metricspb.MetricDescriptor_CUMULATIVE_INT64:
		return googlemetricpb.MetricDescriptor_CUMULATIVE, googlemetricpb.MetricDescriptor_INT64

	case metricspb.MetricDescriptor_CUMULATIVE_DOUBLE:
		return googlemetricpb.MetricDescriptor_CUMULATIVE, googlemetricpb.MetricDescriptor_DOUBLE

	case metricspb.MetricDescriptor_CUMULATIVE_DISTRIBUTION:
		return googlemetricpb.MetricDescriptor_CUMULATIVE, googlemetricpb.MetricDescriptor_DISTRIBUTION

	case metricspb.MetricDescriptor_GAUGE_DOUBLE:
		return googlemetricpb.MetricDescriptor_GAUGE, googlemetricpb.MetricDescriptor_DOUBLE

	case metricspb.MetricDescriptor_GAUGE_INT64:
		return googlemetricpb.MetricDescriptor_GAUGE, googlemetricpb.MetricDescriptor_INT64

	case metricspb.MetricDescriptor_GAUGE_DISTRIBUTION:
		return googlemetricpb.MetricDescriptor_GAUGE, googlemetricpb.MetricDescriptor_DISTRIBUTION

	default:
		return googlemetricpb.MetricDescriptor_METRIC_KIND_UNSPECIFIED, googlemetricpb.MetricDescriptor_VALUE_TYPE_UNSPECIFIED
	}
}

func protoResourceToMonitoredResource(rsp *resourcepb.Resource) *monitoredrespb.MonitoredResource {
	if rsp == nil {
		return nil
	}
	mrsp := &monitoredrespb.MonitoredResource{
		Type: rsp.Type,
	}
	if rsp.Labels != nil {
		mrsp.Labels = make(map[string]string, len(rsp.Labels))
		for k, v := range rsp.Labels {
			mrsp.Labels[k] = v
		}
	}
	return mrsp
}
