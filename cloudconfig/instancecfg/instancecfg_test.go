// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package instancecfg_test

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/cloudconfig/instancecfg"
	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/state/multiwatcher"
	"github.com/juju/juju/testing"
)

type instancecfgSuite struct {
	testing.BaseSuite
}

var _ = gc.Suite(&instancecfgSuite{})

func (*instancecfgSuite) TestInstanceTagsStateServer(c *gc.C) {
	cfg := testing.CustomModelConfig(c, testing.Attrs{})
	stateServerJobs := []multiwatcher.MachineJob{multiwatcher.JobManageModel}
	nonStateServerJobs := []multiwatcher.MachineJob{multiwatcher.JobHostUnits}
	testInstanceTags(c, cfg, stateServerJobs, map[string]string{
		"juju-model-uuid": testing.ModelTag.Id(),
		"juju-is-state":   "true",
	})
	testInstanceTags(c, cfg, nonStateServerJobs, map[string]string{
		"juju-model-uuid": testing.ModelTag.Id(),
	})
}

func (*instancecfgSuite) TestInstanceTagsNoUUID(c *gc.C) {
	attrsWithoutUUID := testing.FakeConfig()
	delete(attrsWithoutUUID, "uuid")
	cfgWithoutUUID, err := config.New(config.NoDefaults, attrsWithoutUUID)
	c.Assert(err, jc.ErrorIsNil)
	testInstanceTags(c,
		cfgWithoutUUID,
		[]multiwatcher.MachineJob(nil),
		map[string]string{"juju-model-uuid": ""},
	)
}

func (*instancecfgSuite) TestInstanceTagsUserSpecified(c *gc.C) {
	cfg := testing.CustomModelConfig(c, testing.Attrs{
		"resource-tags": "a=b c=",
	})
	testInstanceTags(c, cfg, nil, map[string]string{
		"juju-model-uuid": testing.ModelTag.Id(),
		"a":               "b",
		"c":               "",
	})
}

func testInstanceTags(c *gc.C, cfg *config.Config, jobs []multiwatcher.MachineJob, expectTags map[string]string) {
	tags := instancecfg.InstanceTags(cfg, jobs)
	c.Assert(tags, jc.DeepEquals, expectTags)
}
