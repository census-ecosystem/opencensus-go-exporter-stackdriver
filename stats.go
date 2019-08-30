// Copyright 2017, OpenCensus Authors
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
	"os"
	"path"
	"strconv"
	"strings"
	"sync"

	opencensus "go.opencensus.io"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	"go.opencensus.io/metric/metricdata"
	"go.opencensus.io/metric/metricexport"
	"google.golang.org/api/option"
	"google.golang.org/api/support/bundler"
	"google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

const (
	maxTimeSeriesPerUpload    = 200
	opencensusTaskKey         = "opencensus_task"
	opencensusTaskDescription = "Opencensus task identifier"
	defaultDisplayNamePrefix  = "OpenCensus"
	version                   = "0.10.0"
)

var userAgent = fmt.Sprintf("opencensus-go %s; stackdriver-exporter %s", opencensus.Version(), version)

// statsExporter exports stats to the Stackdriver Monitoring.
type statsExporter struct {
	o Options

	metricsBundler *bundler.Bundler

	protoMu                sync.Mutex
	protoMetricDescriptors map[string]bool // Saves the metric descriptors that were already created remotely

	metricMu          sync.Mutex
	metricDescriptors map[string]bool // Saves the metric descriptors that were already created remotely

	c             *monitoring.MetricClient
	defaultLabels map[string]labelValue
	ir            *metricexport.IntervalReader

	initReaderOnce sync.Once
}

var (
	errBlankProjectID = errors.New("expecting a non-blank ProjectID")
)

// newStatsExporter returns an exporter that uploads stats data to Stackdriver Monitoring.
// Only one Stackdriver exporter should be created per ProjectID per process, any subsequent
// invocations of NewExporter with the same ProjectID will return an error.
func newStatsExporter(o Options) (*statsExporter, error) {
	if strings.TrimSpace(o.ProjectID) == "" {
		return nil, errBlankProjectID
	}

	opts := append(o.MonitoringClientOptions, option.WithUserAgent(userAgent))
	ctx := o.Context
	if ctx == nil {
		ctx = context.Background()
	}
	client, err := monitoring.NewMetricClient(ctx, opts...)
	if err != nil {
		return nil, err
	}
	e := &statsExporter{
		c:                      client,
		o:                      o,
		protoMetricDescriptors: make(map[string]bool),
		metricDescriptors:      make(map[string]bool),
	}

	var defaultLablesNotSanitized map[string]labelValue
	if o.DefaultMonitoringLabels != nil {
		defaultLablesNotSanitized = o.DefaultMonitoringLabels.m
	} else {
		defaultLablesNotSanitized = map[string]labelValue{
			opencensusTaskKey: {val: getTaskValue(), desc: opencensusTaskDescription},
		}
	}

	e.defaultLabels = make(map[string]labelValue)
	// Fill in the defaults firstly, irrespective of if the labelKeys and labelValues are mismatched.
	for key, label := range defaultLablesNotSanitized {
		e.defaultLabels[sanitize(key)] = label
	}

	e.metricsBundler = bundler.NewBundler((*metricdata.Metric)(nil), func(bundle interface{}) {
		metrics := bundle.([]*metricdata.Metric)
		e.handleMetricsUpload(metrics)
	})
	if delayThreshold := e.o.BundleDelayThreshold; delayThreshold > 0 {
		e.metricsBundler.DelayThreshold = delayThreshold
	}
	if countThreshold := e.o.BundleCountThreshold; countThreshold > 0 {
		e.metricsBundler.BundleCountThreshold = countThreshold
	}
	return e, nil
}

func (e *statsExporter) startMetricsReader() error {
	e.initReaderOnce.Do(func() {
		e.ir, _ = metricexport.NewIntervalReader(&metricexport.Reader{}, e)
	})
	e.ir.ReportingInterval = e.o.ReportingInterval
	return e.ir.Start()
}

func (e *statsExporter) stopMetricsReader() {
	if e.ir != nil {
		e.ir.Stop()
	}
}

// getTaskValue returns a task label value in the format of
// "go-<pid>@<hostname>".
func getTaskValue() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	return "go-" + strconv.Itoa(os.Getpid()) + "@" + hostname
}

// Flush waits for exported view data and metrics to be uploaded.
//
// This is useful if your program is ending and you do not
// want to lose data that hasn't yet been exported.
func (e *statsExporter) Flush() {
	e.metricsBundler.Flush()
}

func (e *statsExporter) displayName(suffix string) string {
	displayNamePrefix := defaultDisplayNamePrefix
	if e.o.MetricPrefix != "" {
		displayNamePrefix = e.o.MetricPrefix
	}
	return path.Join(displayNamePrefix, suffix)
}

func shouldInsertZeroBound(bounds ...float64) bool {
	if len(bounds) > 0 && bounds[0] != 0.0 {
		return true
	}
	return false
}

func addZeroBucketCountOnCondition(insert bool, counts ...int64) []int64 {
	if insert {
		return append([]int64{0}, counts...)
	}
	return counts
}

func addZeroBoundOnCondition(insert bool, bounds ...float64) []float64 {
	if insert {
		return append([]float64{0.0}, bounds...)
	}
	return bounds
}

var createMetricDescriptor = func(ctx context.Context, c *monitoring.MetricClient, mdr *monitoringpb.CreateMetricDescriptorRequest) (*metric.MetricDescriptor, error) {
	return c.CreateMetricDescriptor(ctx, mdr)
}

var getMetricDescriptor = func(ctx context.Context, c *monitoring.MetricClient, mdr *monitoringpb.GetMetricDescriptorRequest) (*metric.MetricDescriptor, error) {
	return c.GetMetricDescriptor(ctx, mdr)
}

var createTimeSeries = func(ctx context.Context, c *monitoring.MetricClient, ts *monitoringpb.CreateTimeSeriesRequest) error {
	return c.CreateTimeSeries(ctx, ts)
}

var knownExternalMetricPrefixes = []string{
	"custom.googleapis.com/",
	"external.googleapis.com/",
}

// builtinMetric returns true if a MetricType is a heuristically known
// built-in Stackdriver metric
func builtinMetric(metricType string) bool {
	for _, knownExternalMetric := range knownExternalMetricPrefixes {
		if strings.HasPrefix(metricType, knownExternalMetric) {
			return false
		}
	}
	return true
}
