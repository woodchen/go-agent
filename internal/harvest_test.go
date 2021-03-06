package internal

import (
	"testing"
	"time"
)

func TestCreateFinalMetrics(t *testing.T) {
	now := time.Now()

	h := newHarvest(now)
	h.createFinalMetrics()
	expectMetrics(t, h.metrics, []WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{customEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{customEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{txnEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{txnEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{errorEventsSeen, "", true, []float64{0, 0, 0, 0, 0, 0}},
		{errorEventsSent, "", true, []float64{0, 0, 0, 0, 0, 0}},
	})

	h = newHarvest(now)
	h.metrics = newMetricTable(0, now)
	h.customEvents = newCustomEvents(1)
	h.txnEvents = newTxnEvents(1)
	h.errorEvents = newErrorEvents(1)

	h.metrics.addSingleCount("drop me!", unforced)

	customE, err := createCustomEvent("my event type", map[string]interface{}{"zip": 1}, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	h.customEvents.Add(customE)
	h.customEvents.Add(customE)

	txnE := &txnEvent{}
	h.txnEvents.AddTxnEvent(txnE)
	h.txnEvents.AddTxnEvent(txnE)

	h.errorEvents.Add(&errorEvent{})
	h.errorEvents.Add(&errorEvent{})

	h.createFinalMetrics()
	expectMetrics(t, h.metrics, []WantMetric{
		{instanceReporting, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{customEventsSeen, "", true, []float64{2, 0, 0, 0, 0, 0}},
		{customEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{txnEventsSeen, "", true, []float64{2, 0, 0, 0, 0, 0}},
		{txnEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{errorEventsSeen, "", true, []float64{2, 0, 0, 0, 0, 0}},
		{errorEventsSent, "", true, []float64{1, 0, 0, 0, 0, 0}},
		{supportabilityDropped, "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
}

func TestEmptyPayloads(t *testing.T) {
	h := newHarvest(time.Now())
	payloads := h.payloads()
	for _, p := range payloads {
		d, err := p.Data("agentRunID", time.Now())
		if d != nil || err != nil {
			t.Error(d, err)
		}
	}
}

func TestMergeFailedHarvest(t *testing.T) {
	start1 := time.Now()
	start2 := start1.Add(1 * time.Minute)
	h := newHarvest(start1)
	h.metrics.addCount("zip", 1, forced)
	h.txnEvents.AddTxnEvent(&txnEvent{
		Name:      "finalName",
		Timestamp: time.Now(),
		Duration:  1 * time.Second,
	})
	customEventParams := map[string]interface{}{"zip": 1}
	ce, err := createCustomEvent("myEvent", customEventParams, time.Now())
	if nil != err {
		t.Fatal(err)
	}
	h.customEvents.Add(ce)
	h.errorEvents.Add(&errorEvent{
		klass:    "klass",
		msg:      "msg",
		when:     time.Now(),
		txnName:  "finalName",
		duration: 1 * time.Second,
	})
	e := &txnError{
		when:  time.Now(),
		msg:   "msg",
		klass: "klass",
		stack: getStackTrace(0),
	}
	addTxnError(h.errorTraces, e, "finalName", "requestURI", nil)

	if start1 != h.metrics.metricPeriodStart {
		t.Error(h.metrics.metricPeriodStart)
	}
	if 0 != h.metrics.failedHarvests {
		t.Error(h.metrics.failedHarvests)
	}
	if 0 != h.customEvents.events.failedHarvests {
		t.Error(h.customEvents.events.failedHarvests)
	}
	if 0 != h.txnEvents.events.failedHarvests {
		t.Error(h.txnEvents.events.failedHarvests)
	}
	if 0 != h.errorEvents.events.failedHarvests {
		t.Error(h.errorEvents.events.failedHarvests)
	}
	expectMetrics(t, h.metrics, []WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	expectCustomEvents(t, h.customEvents, []WantCustomEvent{
		{Type: "myEvent", Params: customEventParams},
	})
	expectErrorEvents(t, h.errorEvents, []WantErrorEvent{
		{TxnName: "finalName", Msg: "msg", Klass: "klass"},
	})
	expectTxnEvents(t, h.txnEvents, []WantTxnEvent{
		{Name: "finalName"},
	})
	expectErrors(t, h.errorTraces, []WantError{{
		TxnName: "finalName",
		Msg:     "msg",
		Klass:   "klass",
		Caller:  "internal.TestMergeFailedHarvest",
		URL:     "requestURI",
	}})

	nextHarvest := newHarvest(start2)
	if start2 != nextHarvest.metrics.metricPeriodStart {
		t.Error(nextHarvest.metrics.metricPeriodStart)
	}
	payloads := h.payloads()
	for _, p := range payloads {
		p.mergeIntoHarvest(nextHarvest)
	}

	if start1 != nextHarvest.metrics.metricPeriodStart {
		t.Error(nextHarvest.metrics.metricPeriodStart)
	}
	if 1 != nextHarvest.metrics.failedHarvests {
		t.Error(nextHarvest.metrics.failedHarvests)
	}
	if 1 != nextHarvest.customEvents.events.failedHarvests {
		t.Error(nextHarvest.customEvents.events.failedHarvests)
	}
	if 1 != nextHarvest.txnEvents.events.failedHarvests {
		t.Error(nextHarvest.txnEvents.events.failedHarvests)
	}
	if 1 != nextHarvest.errorEvents.events.failedHarvests {
		t.Error(nextHarvest.errorEvents.events.failedHarvests)
	}
	expectMetrics(t, nextHarvest.metrics, []WantMetric{
		{"zip", "", true, []float64{1, 0, 0, 0, 0, 0}},
	})
	expectCustomEvents(t, nextHarvest.customEvents, []WantCustomEvent{
		{Type: "myEvent", Params: customEventParams},
	})
	expectErrorEvents(t, nextHarvest.errorEvents, []WantErrorEvent{
		{TxnName: "finalName", Msg: "msg", Klass: "klass"},
	})
	expectTxnEvents(t, nextHarvest.txnEvents, []WantTxnEvent{
		{Name: "finalName"},
	})
	expectErrors(t, nextHarvest.errorTraces, []WantError{})
}
