// Copyright 2013 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package localstorage_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	gc "launchpad.net/gocheck"

	"launchpad.net/juju-core/environs"
	"launchpad.net/juju-core/environs/localstorage"
	"launchpad.net/juju-core/errors"
	jc "launchpad.net/juju-core/testing/checkers"
)

type storageSuite struct{}

var _ = gc.Suite(&storageSuite{})

// TestPersistence tests the adding, reading, listing and removing
// of files from the local storage.
func (s *storageSuite) TestPersistence(c *gc.C) {
	listener, _, _ := startServer(c)
	defer listener.Close()

	storage := localstorage.Client(listener.Addr().String())
	names := []string{
		"aa",
		"zzz/aa",
		"zzz/bb",
	}
	for _, name := range names {
		checkFileDoesNotExist(c, storage, name)
		checkPutFile(c, storage, name, []byte(name))
	}
	checkList(c, storage, "", names)
	checkList(c, storage, "a", []string{"aa"})
	checkList(c, storage, "zzz/", []string{"zzz/aa", "zzz/bb"})

	storage2 := localstorage.Client(listener.Addr().String())
	for _, name := range names {
		checkFileHasContents(c, storage2, name, []byte(name))
	}

	// remove the first file and check that the others remain.
	err := storage2.Remove(names[0])
	c.Check(err, gc.IsNil)

	// check that it's ok to remove a file twice.
	err = storage2.Remove(names[0])
	c.Check(err, gc.IsNil)

	// ... and check it's been removed in the other environment
	checkFileDoesNotExist(c, storage, names[0])

	// ... and that the rest of the files are still around
	checkList(c, storage2, "", names[1:])

	for _, name := range names[1:] {
		err := storage2.Remove(name)
		c.Assert(err, gc.IsNil)
	}

	// check they've all gone
	checkList(c, storage2, "", nil)

	// Check that RemoveAll works.
	checkRemoveAll(c, storage2)
}

func checkList(c *gc.C, storage environs.StorageReader, prefix string, names []string) {
	lnames, err := storage.List(prefix)
	c.Assert(err, gc.IsNil)
	c.Assert(lnames, gc.DeepEquals, names)
}

type readerWithClose struct {
	*bytes.Buffer
	closeCalled bool
}

var _ io.Reader = (*readerWithClose)(nil)
var _ io.Closer = (*readerWithClose)(nil)

func (r *readerWithClose) Close() error {
	r.closeCalled = true
	return nil
}

func checkPutFile(c *gc.C, storage environs.StorageWriter, name string, contents []byte) {
	c.Logf("check putting file %s ...", name)
	reader := &readerWithClose{bytes.NewBuffer(contents), false}
	err := storage.Put(name, reader, int64(len(contents)))
	c.Assert(err, gc.IsNil)
	c.Assert(reader.closeCalled, jc.IsFalse)
}

func checkFileDoesNotExist(c *gc.C, storage environs.StorageReader, name string) {
	r, err := storage.Get(name)
	c.Assert(r, gc.IsNil)
	c.Assert(err, jc.Satisfies, errors.IsNotFoundError)
}

func checkFileHasContents(c *gc.C, storage environs.StorageReader, name string, contents []byte) {
	r, err := storage.Get(name)
	c.Assert(err, gc.IsNil)
	c.Check(r, gc.NotNil)
	defer r.Close()

	data, err := ioutil.ReadAll(r)
	c.Check(err, gc.IsNil)
	c.Check(data, gc.DeepEquals, contents)

	url, err := storage.URL(name)
	c.Assert(err, gc.IsNil)

	resp, err := http.Get(url)
	c.Assert(err, gc.IsNil)
	data, err = ioutil.ReadAll(resp.Body)
	c.Assert(err, gc.IsNil)
	defer resp.Body.Close()
	c.Assert(resp.StatusCode, gc.Equals, http.StatusOK, gc.Commentf("error response: %s", data))
	c.Check(data, gc.DeepEquals, contents)
}

func checkRemoveAll(c *gc.C, storage environs.Storage) {
	contents := []byte("File contents.")
	aFile := "a-file.txt"
	err := storage.Put(aFile, bytes.NewBuffer(contents), int64(len(contents)))
	c.Assert(err, gc.IsNil)
	err = storage.Put("empty-file", bytes.NewBuffer(nil), 0)
	c.Assert(err, gc.IsNil)

	err = storage.RemoveAll()
	c.Assert(err, gc.IsNil)

	files, err := storage.List("")
	c.Assert(err, gc.IsNil)
	c.Check(files, gc.HasLen, 0)

	_, err = storage.Get(aFile)
	c.Assert(err, gc.NotNil)
	c.Check(err, gc.ErrorMatches, fmt.Sprintf("file %q not found", aFile))
}
