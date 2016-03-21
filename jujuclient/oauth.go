// Copyright 2016 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package jujuclient

import (
	"net/http"
	"net/url"

	"github.com/juju/idmclient/ussologin"
	"gopkg.in/juju/environschema.v1/form"

	"github.com/juju/juju/juju/osenv"
)

// NewTokenStore returns a FileTokenStore for storing the USSO oauth token
func NewTokenStore() *ussologin.FileTokenStore {
	return ussologin.NewFileTokenStore(osenv.JujuXDGDataHomePath("store-usso-token"))
}

func VisitWebPage(filler form.Filler, client *http.Client, store ussologin.TokenStore) func(*url.URL) error {
	return ussologin.VisitWebPage(filler, client, store)
}
