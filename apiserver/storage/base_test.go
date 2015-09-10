// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package storage_test

import (
	"github.com/juju/errors"
	"github.com/juju/names"
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"

	"github.com/juju/juju/apiserver/common"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/apiserver/storage"
	"github.com/juju/juju/apiserver/testing"
	"github.com/juju/juju/state"
	jujustorage "github.com/juju/juju/storage"
	coretesting "github.com/juju/juju/testing"
)

type baseStorageSuite struct {
	coretesting.BaseSuite

	resources  *common.Resources
	authorizer testing.FakeAuthorizer

	api   *storage.API
	state *mockState

	storageTag      names.StorageTag
	storageInstance *mockStorageInstance
	unitTag         names.UnitTag
	machineTag      names.MachineTag

	volumeTag            names.VolumeTag
	volume               *mockVolume
	volumeAttachment     *mockVolumeAttachment
	filesystemTag        names.FilesystemTag
	filesystem           *mockFilesystem
	filesystemAttachment *mockFilesystemAttachment
	calls                []string

	poolManager *mockPoolManager
	pools       map[string]*jujustorage.Config

	blocks map[state.BlockType]state.Block
}

func (s *baseStorageSuite) SetUpTest(c *gc.C) {
	s.BaseSuite.SetUpTest(c)
	s.resources = common.NewResources()
	s.authorizer = testing.FakeAuthorizer{names.NewUserTag("testuser"), true}
	s.calls = []string{}
	s.state = s.constructState(c)

	s.pools = make(map[string]*jujustorage.Config)
	s.poolManager = s.constructPoolManager(c)

	var err error
	s.api, err = storage.CreateAPI(s.state, s.poolManager, s.resources, s.authorizer)
	c.Assert(err, jc.ErrorIsNil)
}

func (s *baseStorageSuite) assertCalls(c *gc.C, expectedCalls []string) {
	c.Assert(s.calls, jc.SameContents, expectedCalls)
}

const (
	allStorageInstancesCall                 = "allStorageInstances"
	storageInstanceAttachmentsCall          = "storageInstanceAttachments"
	unitAssignedMachineCall                 = "UnitAssignedMachine"
	storageInstanceCall                     = "StorageInstance"
	storageInstanceFilesystemCall           = "StorageInstanceFilesystem"
	storageInstanceFilesystemAttachmentCall = "storageInstanceFilesystemAttachment"
	storageInstanceVolumeCall               = "storageInstanceVolume"
	volumeCall                              = "volumeCall"
	machineVolumeAttachmentsCall            = "machineVolumeAttachments"
	volumeAttachmentsCall                   = "volumeAttachments"
	allVolumesCall                          = "allVolumes"
	filesystemCall                          = "filesystemCall"
	machineFilesystemAttachmentsCall        = "machineFilesystemAttachments"
	filesystemAttachmentsCall               = "filesystemAttachments"
	allFilesystemsCall                      = "allFilesystems"
	addStorageForUnitCall                   = "addStorageForUnit"
	getBlockForTypeCall                     = "getBlockForType"
)

func (s *baseStorageSuite) constructState(c *gc.C) *mockState {
	s.unitTag = names.NewUnitTag("mysql/0")
	s.storageTag = names.NewStorageTag("data/0")

	s.storageInstance = &mockStorageInstance{
		kind:       state.StorageKindFilesystem,
		owner:      s.unitTag,
		storageTag: s.storageTag,
	}

	storageInstanceAttachment := &mockStorageAttachment{storage: s.storageInstance}

	s.machineTag = names.NewMachineTag("66")
	s.filesystemTag = names.NewFilesystemTag("104")
	s.volumeTag = names.NewVolumeTag("22")
	s.filesystem = &mockFilesystem{
		tag:     s.filesystemTag,
		storage: &s.storageTag,
	}
	s.filesystemAttachment = &mockFilesystemAttachment{
		filesystem: s.filesystemTag,
		machine:    s.machineTag,
	}
	s.volume = &mockVolume{tag: s.volumeTag, storage: &s.storageTag}
	s.volumeAttachment = &mockVolumeAttachment{
		VolumeTag:  s.volumeTag,
		MachineTag: s.machineTag,
	}

	s.blocks = make(map[state.BlockType]state.Block)
	return &mockState{
		allStorageInstances: func() ([]state.StorageInstance, error) {
			s.calls = append(s.calls, allStorageInstancesCall)
			return []state.StorageInstance{s.storageInstance}, nil
		},
		storageInstance: func(sTag names.StorageTag) (state.StorageInstance, error) {
			s.calls = append(s.calls, storageInstanceCall)
			c.Assert(sTag, gc.DeepEquals, s.storageTag)
			return s.storageInstance, nil
		},
		storageInstanceAttachments: func(tag names.StorageTag) ([]state.StorageAttachment, error) {
			s.calls = append(s.calls, storageInstanceAttachmentsCall)
			c.Assert(tag, gc.DeepEquals, s.storageTag)
			return []state.StorageAttachment{storageInstanceAttachment}, nil
		},
		storageInstanceFilesystem: func(sTag names.StorageTag) (state.Filesystem, error) {
			s.calls = append(s.calls, storageInstanceFilesystemCall)
			c.Assert(sTag, gc.DeepEquals, s.storageTag)
			return s.filesystem, nil
		},
		storageInstanceFilesystemAttachment: func(m names.MachineTag, f names.FilesystemTag) (state.FilesystemAttachment, error) {
			s.calls = append(s.calls, storageInstanceFilesystemAttachmentCall)
			c.Assert(m, gc.DeepEquals, s.machineTag)
			c.Assert(f, gc.DeepEquals, s.filesystemTag)
			return s.filesystemAttachment, nil
		},
		storageInstanceVolume: func(t names.StorageTag) (state.Volume, error) {
			s.calls = append(s.calls, storageInstanceVolumeCall)
			c.Assert(t, gc.DeepEquals, s.storageTag)
			return s.volume, nil
		},
		unitAssignedMachine: func(u names.UnitTag) (names.MachineTag, error) {
			s.calls = append(s.calls, unitAssignedMachineCall)
			c.Assert(u, gc.DeepEquals, s.unitTag)
			return s.machineTag, nil
		},
		volume: func(tag names.VolumeTag) (state.Volume, error) {
			s.calls = append(s.calls, volumeCall)
			c.Assert(tag, gc.DeepEquals, s.volumeTag)
			return s.volume, nil
		},
		machineVolumeAttachments: func(machine names.MachineTag) ([]state.VolumeAttachment, error) {
			s.calls = append(s.calls, machineVolumeAttachmentsCall)
			c.Assert(machine, gc.DeepEquals, s.machineTag)
			return []state.VolumeAttachment{s.volumeAttachment}, nil
		},
		volumeAttachments: func(volume names.VolumeTag) ([]state.VolumeAttachment, error) {
			s.calls = append(s.calls, volumeAttachmentsCall)
			c.Assert(volume, gc.DeepEquals, s.volumeTag)
			return []state.VolumeAttachment{s.volumeAttachment}, nil
		},
		allVolumes: func() ([]state.Volume, error) {
			s.calls = append(s.calls, allVolumesCall)
			return []state.Volume{s.volume}, nil
		},
		filesystem: func(tag names.FilesystemTag) (state.Filesystem, error) {
			s.calls = append(s.calls, filesystemCall)
			c.Assert(tag, gc.DeepEquals, s.filesystemTag)
			return s.filesystem, nil
		},
		machineFilesystemAttachments: func(machine names.MachineTag) ([]state.FilesystemAttachment, error) {
			s.calls = append(s.calls, machineFilesystemAttachmentsCall)
			c.Assert(machine, gc.DeepEquals, s.machineTag)
			return []state.FilesystemAttachment{s.filesystemAttachment}, nil
		},
		filesystemAttachments: func(filesystem names.FilesystemTag) ([]state.FilesystemAttachment, error) {
			s.calls = append(s.calls, filesystemAttachmentsCall)
			c.Assert(filesystem, gc.DeepEquals, s.filesystemTag)
			return []state.FilesystemAttachment{s.filesystemAttachment}, nil
		},
		allFilesystems: func() ([]state.Filesystem, error) {
			s.calls = append(s.calls, allFilesystemsCall)
			return []state.Filesystem{s.filesystem}, nil
		},
		envName: "storagetest",
		addStorageForUnit: func(u names.UnitTag, name string, cons state.StorageConstraints) error {
			s.calls = append(s.calls, addStorageForUnitCall)
			return nil
		},
		getBlockForType: func(t state.BlockType) (state.Block, bool, error) {
			s.calls = append(s.calls, getBlockForTypeCall)
			val, found := s.blocks[t]
			return val, found, nil
		},
	}
}

func (s *baseStorageSuite) addBlock(c *gc.C, t state.BlockType, msg string) {
	s.blocks[t] = mockBlock{
		t:   t,
		msg: msg,
	}
}

func (s *baseStorageSuite) blockAllChanges(c *gc.C, msg string) {
	s.addBlock(c, state.ChangeBlock, msg)
}

func (s *baseStorageSuite) blockDestroyEnvironment(c *gc.C, msg string) {
	s.addBlock(c, state.DestroyBlock, msg)
}

func (s *baseStorageSuite) blockRemoveObject(c *gc.C, msg string) {
	s.addBlock(c, state.RemoveBlock, msg)
}

func (s *baseStorageSuite) assertBlocked(c *gc.C, err error, msg string) {
	c.Assert(params.IsCodeOperationBlocked(err), jc.IsTrue)
	c.Assert(err, gc.ErrorMatches, msg)
}

func (s *baseStorageSuite) constructPoolManager(c *gc.C) *mockPoolManager {
	return &mockPoolManager{
		getPool: func(name string) (*jujustorage.Config, error) {
			if one, ok := s.pools[name]; ok {
				return one, nil
			}
			return nil, errors.NotFoundf("mock pool manager: get pool %v", name)
		},
		createPool: func(name string, providerType jujustorage.ProviderType, attrs map[string]interface{}) (*jujustorage.Config, error) {
			pool, err := jujustorage.NewConfig(name, providerType, attrs)
			s.pools[name] = pool
			return pool, err
		},
		deletePool: func(name string) error {
			delete(s.pools, name)
			return nil
		},
		listPools: func() ([]*jujustorage.Config, error) {
			result := make([]*jujustorage.Config, len(s.pools))
			i := 0
			for _, v := range s.pools {
				result[i] = v
				i++
			}
			return result, nil
		},
	}
}
