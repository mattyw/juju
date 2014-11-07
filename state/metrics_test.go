// Copyright 2013, 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package state_test

import (
	"strconv"
	"time"

	"github.com/juju/errors"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/state"
	"github.com/juju/juju/testing/factory"
)

type MetricSuite struct {
	ConnSuite
	unit *state.Unit
}

var _ = gc.Suite(&MetricSuite{})

func (s *MetricSuite) SetUpTest(c *gc.C) {
	s.ConnSuite.SetUpTest(c)
	s.unit = s.assertAddUnit(c)
}

func (s *MetricSuite) TestAddNoMetrics(c *gc.C) {
	now := state.NowToTheSecond()
	_, err := s.unit.AddMetrics(now, []state.Metric{})
	c.Assert(err, gc.ErrorMatches, "cannot add a batch of 0 metrics")
}

func (s *MetricSuite) TestAddMetric(c *gc.C) {
	now := state.NowToTheSecond()
	envUUID := s.State.EnvironUUID()
	m := state.Metric{"item", "5", now, []byte("creds")}
	metricBatch, err := s.unit.AddMetrics(now, []state.Metric{m})
	c.Assert(err, gc.IsNil)
	c.Assert(metricBatch.Unit(), gc.Equals, "wordpress/0")
	c.Assert(metricBatch.EnvUUID(), gc.Equals, envUUID)
	c.Assert(metricBatch.CharmURL(), gc.Equals, "local:quantal/quantal-wordpress-3")
	c.Assert(metricBatch.Sent(), gc.Equals, false)
	c.Assert(metricBatch.Created(), gc.Equals, now)
	c.Assert(metricBatch.Metrics(), gc.HasLen, 1)

	metric := metricBatch.Metrics()[0]
	c.Assert(metric.Key, gc.Equals, "item")
	c.Assert(metric.Value, gc.Equals, "5")
	c.Assert(metric.Time.Equal(now), jc.IsTrue)
	c.Assert(metric.Credentials, gc.DeepEquals, []byte("creds"))

	saved, err := s.State.MetricBatch(metricBatch.UUID())
	c.Assert(err, gc.IsNil)
	c.Assert(saved.Unit(), gc.Equals, "wordpress/0")
	c.Assert(metricBatch.CharmURL(), gc.Equals, "local:quantal/quantal-wordpress-3")
	c.Assert(saved.Sent(), gc.Equals, false)
	c.Assert(saved.Metrics(), gc.HasLen, 1)
	metric = saved.Metrics()[0]
	c.Assert(metric.Key, gc.Equals, "item")
	c.Assert(metric.Value, gc.Equals, "5")
	c.Assert(metric.Time.Equal(now), jc.IsTrue)
	c.Assert(metric.Credentials, gc.DeepEquals, []byte("creds"))
}

func assertUnitRemoved(c *gc.C, unit *state.Unit) {
	assertUnitDead(c, unit)
	err := unit.Remove()
	c.Assert(err, gc.IsNil)
}

func assertUnitDead(c *gc.C, unit *state.Unit) {
	err := unit.EnsureDead()
	c.Assert(err, gc.IsNil)
}

func (s *MetricSuite) assertAddUnit(c *gc.C) *state.Unit {
	charm := s.AddTestingCharm(c, "wordpress")
	svc := s.AddTestingService(c, "wordpress", charm)
	unit, err := svc.AddUnit()
	c.Assert(err, gc.IsNil)
	err = unit.SetCharmURL(charm.URL())
	c.Assert(err, gc.IsNil)
	return unit
}

func (s *MetricSuite) TestAddMetricNonExitentUnit(c *gc.C) {
	assertUnitRemoved(c, s.unit)
	now := state.NowToTheSecond()
	m := state.Metric{"item", "5", now, []byte{}}
	_, err := s.unit.AddMetrics(now, []state.Metric{m})
	c.Assert(err, gc.ErrorMatches, `wordpress/0 not found`)
}

func (s *MetricSuite) TestAddMetricDeadUnit(c *gc.C) {
	assertUnitDead(c, s.unit)
	now := state.NowToTheSecond()
	m := state.Metric{"item", "5", now, []byte{}}
	_, err := s.unit.AddMetrics(now, []state.Metric{m})
	c.Assert(err, gc.ErrorMatches, `wordpress/0 not found`)
}

func (s *MetricSuite) TestSetMetricSent(c *gc.C) {
	now := state.NowToTheSecond()
	m := state.Metric{"item", "5", now, []byte{}}
	added, err := s.unit.AddMetrics(now, []state.Metric{m})
	c.Assert(err, gc.IsNil)
	saved, err := s.State.MetricBatch(added.UUID())
	c.Assert(err, gc.IsNil)
	err = saved.SetSent()
	c.Assert(err, gc.IsNil)
	c.Assert(saved.Sent(), jc.IsTrue)
	saved, err = s.State.MetricBatch(added.UUID())
	c.Assert(err, gc.IsNil)
	c.Assert(saved.Sent(), jc.IsTrue)
}

func (s *MetricSuite) TestCleanupMetrics(c *gc.C) {
	oldTime := time.Now().Add(-(time.Hour * 25))
	m := state.Metric{"item", "5", oldTime, []byte("creds")}
	oldMetric, err := s.unit.AddMetrics(oldTime, []state.Metric{m})
	c.Assert(err, gc.IsNil)
	oldMetric.SetSent()

	now := time.Now()
	m = state.Metric{"item", "5", now, []byte("creds")}
	newMetric, err := s.unit.AddMetrics(now, []state.Metric{m})
	c.Assert(err, gc.IsNil)
	newMetric.SetSent()
	err = s.State.CleanupOldMetrics()
	c.Assert(err, gc.IsNil)

	_, err = s.State.MetricBatch(newMetric.UUID())
	c.Assert(err, gc.IsNil)

	_, err = s.State.MetricBatch(oldMetric.UUID())
	c.Assert(err, jc.Satisfies, errors.IsNotFound)
}

func (s *MetricSuite) TestCleanupNoMetrics(c *gc.C) {
	err := s.State.CleanupOldMetrics()
	c.Assert(err, gc.IsNil)
}

func (s *MetricSuite) TestMetricBatches(c *gc.C) {
	now := state.NowToTheSecond()
	m := state.Metric{"item", "5", now, []byte("creds")}
	_, err := s.unit.AddMetrics(now, []state.Metric{m})
	c.Assert(err, gc.IsNil)
	metricBatches, err := s.State.MetricBatches()
	c.Assert(err, gc.IsNil)
	c.Assert(metricBatches, gc.HasLen, 1)
	c.Assert(metricBatches[0].Unit(), gc.Equals, "wordpress/0")
	c.Assert(metricBatches[0].CharmURL(), gc.Equals, "local:quantal/quantal-wordpress-3")
	c.Assert(metricBatches[0].Sent(), gc.Equals, false)
	c.Assert(metricBatches[0].Metrics(), gc.HasLen, 1)
}

// TestCountMetrics asserts the correct values are returned
// by CountofUnsentMetrics and CountofSentMetrics.
func (s *MetricSuite) TestCountMetrics(c *gc.C) {
	unit := s.factory.MakeUnit(c, &factory.UnitParams{SetCharmURL: true})
	now := time.Now()
	s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: false, Time: &now})
	s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: false, Time: &now})
	s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: true, Time: &now})
	sent, err := s.State.CountofSentMetrics()
	c.Assert(err, gc.IsNil)
	c.Assert(sent, gc.Equals, 1)
	unsent, err := s.State.CountofUnsentMetrics()
	c.Assert(err, gc.IsNil)
	c.Assert(unsent, gc.Equals, 2)
	c.Assert(unsent+sent, gc.Equals, 3)
}

func (s *MetricSuite) TestSetMetricBatchesSent(c *gc.C) {
	unit := s.factory.MakeUnit(c, &factory.UnitParams{SetCharmURL: true})
	now := time.Now()
	metrics := make([]*state.MetricBatch, 3)
	for i := range metrics {
		metrics[i] = s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: false, Time: &now})
	}
	err := s.State.SetMetricBatchesSent(metrics)
	c.Assert(err, gc.IsNil)
	sent, err := s.State.CountofSentMetrics()
	c.Assert(err, gc.IsNil)
	c.Assert(sent, gc.Equals, 3)

}

func (s *MetricSuite) TestMetricsToSend(c *gc.C) {
	unit := s.factory.MakeUnit(c, &factory.UnitParams{SetCharmURL: true})
	now := state.NowToTheSecond()
	s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: false, Time: &now})
	s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: false, Time: &now})
	s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: true, Time: &now})
	result, err := s.State.MetricsToSend(5)
	c.Assert(err, gc.IsNil)
	c.Assert(result, gc.HasLen, 2)
}

// TestMetricsToSendBatches checks that metrics are properly batched.
func (s *MetricSuite) TestMetricsToSendBatches(c *gc.C) {
	unit := s.factory.MakeUnit(c, &factory.UnitParams{SetCharmURL: true})
	now := state.NowToTheSecond()
	for i := 0; i < 6; i++ {
		s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: false, Time: &now})
	}
	for i := 0; i < 4; i++ {
		s.factory.MakeMetric(c, &factory.MetricParams{Unit: unit, Sent: true, Time: &now})
	}
	for i := 0; i < 3; i++ {
		result, err := s.State.MetricsToSend(2)
		c.Assert(err, gc.IsNil)
		c.Assert(result, gc.HasLen, 2)
		s.State.SetMetricBatchesSent(result)
	}
	result, err := s.State.MetricsToSend(2)
	c.Assert(err, gc.IsNil)
	c.Assert(result, gc.HasLen, 0)
}

// TestUnitsRequiringMetric checks that the correct number of units are returned by the call
func (s *MetricSuite) TestUnitsRequiringMetric(c *gc.C) {
	meteredCharm := s.factory.MakeCharm(c, &factory.CharmParams{Name: "metered", URL: "cs:quantal/metered"})
	//TODO (mattyw) Would be useful to be able to create a unit directly from a charm in the factory
	meteredService := s.factory.MakeService(c, &factory.ServiceParams{Charm: meteredCharm})
	s.factory.MakeUnit(c, &factory.UnitParams{SetCharmURL: true})
	s.factory.MakeUnit(c, &factory.UnitParams{Service: meteredService, SetCharmURL: true})
	charms, err := s.State.CharmsRequiringMetric("pings")
	c.Assert(err, gc.IsNil)
	c.Assert(charms, gc.HasLen, 1)
	c.Assert(charms[0].String(), gc.Equals, "cs:quantal/metered")
}

func (s *MetricSuite) TestCharmSetUrlDuration(c *gc.C) {
	t0 := time.Now().Add(3 * time.Second)
	meteredCharm := s.factory.MakeCharm(c, &factory.CharmParams{Name: "metered", URL: "cs:quantal/metered"})
	meteredService := s.factory.MakeService(c, &factory.ServiceParams{Charm: meteredCharm})
	s.factory.MakeUnit(c, &factory.UnitParams{SetCharmURL: true})
	meteredUnit := s.factory.MakeUnit(c, &factory.UnitParams{Service: meteredService, SetCharmURL: true})
	charms, err := s.State.CharmsRequiringMetric("pings")
	c.Assert(err, gc.IsNil)
	times, err := s.State.CharmSetUrlDuration(t0, charms)
	c.Assert(err, gc.IsNil)
	c.Assert(times, gc.HasLen, 1)
	c.Assert(times[*meteredCharm.URL()][meteredUnit.Name()], gc.HasLen, 1)
	metric := times[*meteredCharm.URL()][meteredUnit.Name()][0]
	c.Assert(metric.Key, gc.Equals, "juju-unit-time")
	unitTime, err := strconv.Atoi(metric.Value)
	c.Assert(err, gc.IsNil)
	c.Assert(unitTime > 1, jc.IsTrue)
	c.Assert(unitTime < 3, jc.IsTrue)
}

func (s *MetricSuite) TestAddBulkMetrics(c *gc.C) {
	meteredUnit0 := s.factory.MakeUnit(c, &factory.UnitParams{SetCharmURL: true})
	meteredCharm := s.factory.MakeCharm(c, &factory.CharmParams{Name: "metered", URL: "cs:quantal/metered"})
	meteredService := s.factory.MakeService(c, &factory.ServiceParams{Charm: meteredCharm})
	meteredUnit1 := s.factory.MakeUnit(c, &factory.UnitParams{Service: meteredService, SetCharmURL: true})
	url0, _ := meteredUnit0.CharmURL()
	url1, _ := meteredUnit1.CharmURL()
	ct0 := map[string][]state.Metric{
		meteredUnit0.Name(): []state.Metric{{Key: "foobar", Value: "123"}},
	}
	ct1 := map[string][]state.Metric{
		meteredUnit1.Name(): []state.Metric{{Key: "barbar", Value: "456"}},
	}
	bulkMetrics := state.BulkMetrics{
		*url0: ct0, *url1: ct1,
	}
	mb, err := s.State.AddBulkMetrics(bulkMetrics)
	c.Assert(err, gc.IsNil)
	c.Assert(mb, gc.HasLen, 2)
	count, err := s.State.CountofUnsentMetrics()
	c.Assert(err, gc.IsNil)
	c.Assert(count, gc.Equals, 2)
	m0 := mb[0].Metrics()
	c.Assert(m0, gc.HasLen, 1)
	c.Assert(m0[0].Key, gc.Equals, "foobar")
	c.Assert(m0[0].Value, gc.Equals, "123")
	m1 := mb[1].Metrics()
	c.Assert(m1, gc.HasLen, 1)
	c.Assert(m1[0].Key, gc.Equals, "barbar")
	c.Assert(m1[0].Value, gc.Equals, "456")
}
