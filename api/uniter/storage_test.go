// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package uniter_test

import (
	"github.com/juju/errors"
	"github.com/juju/names"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/api/base/testing"
	"github.com/juju/juju/api/uniter"
	"github.com/juju/juju/apiserver/params"
	coretesting "github.com/juju/juju/testing"
)

var _ = gc.Suite(&storageSuite{})

type storageSuite struct {
	coretesting.BaseSuite
}

func (s *storageSuite) TestUnitStorageAttachments(c *gc.C) {
	storageAttachments := []params.StorageAttachment{{
		StorageTag: "storage-whatever-0",
		OwnerTag:   "service-mysql",
		UnitTag:    "unit-mysql-0",
		Kind:       params.StorageKindBlock,
		Location:   "/dev/sda",
	}}

	var called bool
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "Uniter")
		c.Check(version, gc.Equals, 2)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "UnitStorageAttachments")
		c.Check(arg, gc.DeepEquals, params.Entities{
			Entities: []params.Entity{{Tag: "unit-mysql-0"}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.StorageAttachmentsResults{})
		*(result.(*params.StorageAttachmentsResults)) = params.StorageAttachmentsResults{
			Results: []params.StorageAttachmentsResult{{
				Result: storageAttachments,
			}},
		}
		called = true
		return nil
	})

	st := uniter.NewState(apiCaller, names.NewUnitTag("mysql/0"))
	attachments, err := st.UnitStorageAttachments(names.NewUnitTag("mysql/0"))
	c.Check(err, jc.ErrorIsNil)
	c.Check(called, jc.IsTrue)
	c.Assert(attachments, gc.DeepEquals, storageAttachments)
}

func (s *storageSuite) TestStorageAttachmentResultCountMismatch(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		*(result.(*params.StorageAttachmentsResults)) = params.StorageAttachmentsResults{
			[]params.StorageAttachmentsResult{{}, {}},
		}
		return nil
	})
	st := uniter.NewState(apiCaller, names.NewUnitTag("mysql/0"))
	c.Assert(func() {
		st.UnitStorageAttachments(names.NewUnitTag("mysql/0"))
	}, gc.PanicMatches, "expected 1 result, got 2")
}

func (s *storageSuite) TestAPIErrors(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		return errors.New("bad")
	})
	st := uniter.NewState(apiCaller, names.NewUnitTag("mysql/0"))
	_, err := st.UnitStorageAttachments(names.NewUnitTag("mysql/0"))
	c.Check(err, gc.ErrorMatches, "bad")
}

func (s *storageSuite) TestWatchUnitStorageAttachments(c *gc.C) {
	var called bool
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "Uniter")
		c.Check(version, gc.Equals, 2)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "WatchUnitStorageAttachments")
		c.Check(arg, gc.DeepEquals, params.Entities{
			Entities: []params.Entity{{Tag: "unit-mysql-0"}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.StringsWatchResults{})
		*(result.(*params.StringsWatchResults)) = params.StringsWatchResults{
			Results: []params.StringsWatchResult{{
				Error: &params.Error{Message: "FAIL"},
			}},
		}
		called = true
		return nil
	})

	st := uniter.NewState(apiCaller, names.NewUnitTag("mysql/0"))
	_, err := st.WatchUnitStorageAttachments(names.NewUnitTag("mysql/0"))
	c.Check(err, gc.ErrorMatches, "FAIL")
	c.Check(called, jc.IsTrue)
}

func (s *storageSuite) TestWatchStorageAttachments(c *gc.C) {
	var called bool
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "Uniter")
		c.Check(version, gc.Equals, 2)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "WatchStorageAttachmentInfos")
		c.Check(arg, gc.DeepEquals, params.StorageAttachmentIds{
			Ids: []params.StorageAttachmentId{{
				StorageTag: "storage-data-0",
				UnitTag:    "unit-mysql-0",
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.NotifyWatchResults{})
		*(result.(*params.NotifyWatchResults)) = params.NotifyWatchResults{
			Results: []params.NotifyWatchResult{{
				Error: &params.Error{Message: "FAIL"},
			}},
		}
		called = true
		return nil
	})

	st := uniter.NewState(apiCaller, names.NewUnitTag("mysql/0"))
	_, err := st.WatchStorageAttachment(names.NewStorageTag("data/0"), names.NewUnitTag("mysql/0"))
	c.Check(err, gc.ErrorMatches, "FAIL")
	c.Check(called, jc.IsTrue)
}

func (s *storageSuite) TestStorageAttachments(c *gc.C) {
	storageAttachment := params.StorageAttachment{
		StorageTag: "storage-whatever-0",
		OwnerTag:   "service-mysql",
		UnitTag:    "unit-mysql-0",
		Kind:       params.StorageKindBlock,
		Location:   "/dev/sda",
	}

	var called bool
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "Uniter")
		c.Check(version, gc.Equals, 2)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "StorageAttachments")
		c.Check(arg, gc.DeepEquals, params.StorageAttachmentIds{
			Ids: []params.StorageAttachmentId{{
				StorageTag: "storage-data-0",
				UnitTag:    "unit-mysql-0",
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.StorageAttachmentResults{})
		*(result.(*params.StorageAttachmentResults)) = params.StorageAttachmentResults{
			Results: []params.StorageAttachmentResult{{
				Result: storageAttachment,
			}},
		}
		called = true
		return nil
	})

	st := uniter.NewState(apiCaller, names.NewUnitTag("mysql/0"))
	attachment, err := st.StorageAttachment(names.NewStorageTag("data/0"), names.NewUnitTag("mysql/0"))
	c.Check(err, jc.ErrorIsNil)
	c.Check(called, jc.IsTrue)
	c.Assert(attachment, gc.DeepEquals, storageAttachment)
}

func (s *storageSuite) TestEnsureStorageAttachmentDead(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "Uniter")
		c.Check(version, gc.Equals, 2)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "EnsureStorageAttachmentsDead")
		c.Check(arg, gc.DeepEquals, params.StorageAttachmentIds{
			Ids: []params.StorageAttachmentId{{
				StorageTag: "storage-data-0",
				UnitTag:    "unit-mysql-0",
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.ErrorResults{})
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			Results: []params.ErrorResult{{
				Error: &params.Error{Message: "yessirilikeit"},
			}},
		}
		return nil
	})

	st := uniter.NewState(apiCaller, names.NewUnitTag("mysql/0"))
	err := st.EnsureStorageAttachmentDead(names.NewStorageTag("data/0"), names.NewUnitTag("mysql/0"))
	c.Check(err, gc.ErrorMatches, "yessirilikeit")
}

func (s *storageSuite) TestRemoveStorageAttachment(c *gc.C) {
	apiCaller := testing.APICallerFunc(func(objType string, version int, id, request string, arg, result interface{}) error {
		c.Check(objType, gc.Equals, "Uniter")
		c.Check(version, gc.Equals, 2)
		c.Check(id, gc.Equals, "")
		c.Check(request, gc.Equals, "RemoveStorageAttachments")
		c.Check(arg, gc.DeepEquals, params.StorageAttachmentIds{
			Ids: []params.StorageAttachmentId{{
				StorageTag: "storage-data-0",
				UnitTag:    "unit-mysql-0",
			}},
		})
		c.Assert(result, gc.FitsTypeOf, &params.ErrorResults{})
		*(result.(*params.ErrorResults)) = params.ErrorResults{
			Results: []params.ErrorResult{{
				Error: &params.Error{Message: "yoink"},
			}},
		}
		return nil
	})

	st := uniter.NewState(apiCaller, names.NewUnitTag("mysql/0"))
	err := st.RemoveStorageAttachment(names.NewStorageTag("data/0"), names.NewUnitTag("mysql/0"))
	c.Check(err, gc.ErrorMatches, "yoink")
}
