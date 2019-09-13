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
	"strings"

	monitoring "cloud.google.com/go/monitoring/apiv3"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

type metricsBatcher struct {
	projectName string
	allTss      []*monitoringpb.TimeSeries
	allErrs     []error

	// Counts all dropped TimeSeries by this metricsBatcher.
	droppedTimeSeries int

	workers   []*worker
	curWorker int

	mc *monitoring.MetricClient
}

func newMetricsBatcher(ctx context.Context, projectID string, numWorkers int, mc *monitoring.MetricClient) *metricsBatcher {
	workers := make([]*worker, 0, numWorkers)
	for i := 0; i < numWorkers; i++ {
		w := newWorker(ctx, mc)
		workers = append(workers, w)
		go w.start()
	}
	return &metricsBatcher{
		projectName:       fmt.Sprintf("projects/%s", projectID),
		allTss:            make([]*monitoringpb.TimeSeries, 0, maxTimeSeriesPerUpload),
		droppedTimeSeries: 0,
		workers:           workers,
		curWorker:         0,
		mc:                mc,
	}
}

func (mb *metricsBatcher) recordDroppedTimeseries(numTimeSeries int, errs ...error) {
	mb.droppedTimeSeries += numTimeSeries
	for _, err := range errs {
		if err != nil {
			mb.allErrs = append(mb.allErrs, err)
		}
	}
}

func (mb *metricsBatcher) addTimeSeries(ts *monitoringpb.TimeSeries) {
	mb.allTss = append(mb.allTss, ts)
	if len(mb.allTss) == maxTimeSeriesPerUpload && len(mb.workers) != 0 {
		req := &monitoringpb.CreateTimeSeriesRequest{
			Name:       mb.projectName,
			TimeSeries: mb.allTss,
		}
		mb.workers[mb.curWorker].reqChan <- req
		mb.nextWorker()
		mb.allTss = make([]*monitoringpb.TimeSeries, 0, maxTimeSeriesPerUpload)
	}
}

func (mb *metricsBatcher) nextWorker() {
	mb.curWorker++
	if mb.curWorker >= len(mb.workers) {
		mb.curWorker = 0
	}
}

func (mb *metricsBatcher) close(ctx context.Context) error {
	for _, w := range mb.workers {
		resp := w.stop()
		mb.recordDroppedTimeseries(resp.droppedTimeSeries, resp.errs...)
	}

	// Send any remaining time series
	if len(mb.allTss) > 0 && mb.mc != nil {
		var reqs []*monitoringpb.CreateTimeSeriesRequest
		for start := 0; start < len(mb.allTss); {
			end := start + maxTimeSeriesPerUpload
			if end > len(mb.allTss) {
				end = len(mb.allTss)
			}
			reqs = append(reqs, &monitoringpb.CreateTimeSeriesRequest{
				Name:       mb.projectName,
				TimeSeries: mb.allTss[start:end],
			})
			start = end
		}

		for _, req := range reqs {
			mb.recordDroppedTimeseries(sendReq(ctx, mb.mc, req))
		}
	}

	numErrors := len(mb.allErrs)
	if numErrors == 0 {
		return nil
	}

	if numErrors == 1 {
		return mb.allErrs[0]
	}

	errMsgs := make([]string, 0, numErrors)
	for _, err := range mb.allErrs {
		errMsgs = append(errMsgs, err.Error())
	}
	return fmt.Errorf("[%s]", strings.Join(errMsgs, "; "))
}

// sendReq sends create time series requests to Stackdriver,
// and returns the count of dropped time series and error.
func sendReq(ctx context.Context, c *monitoring.MetricClient, req *monitoringpb.CreateTimeSeriesRequest) (int, error) {
	err := createTimeSeries(ctx, c, req)
	if err != nil {
		return len(req.TimeSeries), err
	}
	return 0, nil
}

type worker struct {
	ctx context.Context
	mc  *monitoring.MetricClient

	resp *response

	quit     chan bool
	respChan chan *response
	reqChan  chan *monitoringpb.CreateTimeSeriesRequest
}

func newWorker(ctx context.Context, mc *monitoring.MetricClient) *worker {
	return &worker{
		ctx:      ctx,
		mc:       mc,
		resp:     &response{},
		reqChan:  make(chan *monitoringpb.CreateTimeSeriesRequest),
		respChan: make(chan *response),
		quit:     make(chan bool),
	}
}

func (w *worker) start() {
	for {
		select {
		case req := <-w.reqChan:
			w.recordDroppedTimeseries(sendReq(w.ctx, w.mc, req))
		case <-w.quit:
			close(w.reqChan)
			w.respChan <- w.resp
			return
		}
	}
}

func (w *worker) stop() *response {
	w.quit <- true
	return <-w.respChan
}

func (w *worker) recordDroppedTimeseries(numTimeSeries int, err error) {
	w.resp.droppedTimeSeries += numTimeSeries
	if err != nil {
		w.resp.errs = append(w.resp.errs, err)
	}
}

type response struct {
	droppedTimeSeries int
	errs              []error
}
