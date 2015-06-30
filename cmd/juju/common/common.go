// Copyright 2014 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package common

import (
	"fmt"
	"net/http"
	"path"

	"github.com/juju/cmd"
	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/persistent-cookiejar"
	"github.com/juju/utils"
	"golang.org/x/net/publicsuffix"
	"gopkg.in/juju/charm.v5"
	"gopkg.in/juju/charm.v5/charmrepo"
	"gopkg.in/juju/charmstore.v4/csclient"
	"gopkg.in/macaroon-bakery.v0/httpbakery"
	"gopkg.in/macaroon.v1"

	"github.com/juju/juju/api"
	"github.com/juju/juju/apiserver/params"
	"github.com/juju/juju/environs/config"
)

var logger = loggo.GetLogger("juju.cmd.juju")

// ResolveCharmURL resolves the given charm URL string
// by looking it up in the appropriate charm repository.
// If it is a charm store charm URL, the given csParams will
// be used to access the charm store repository.
// If it is a local charm URL, the local charm repository at
// the given repoPath will be used. The given configuration
// will be used to add any necessary attributes to the repo
// and to resolve the default series if possible.
//
// ResolveCharmURL also returns the charm repository holding
// the charm.
func ResolveCharmURL(curlStr string, csParams charmrepo.NewCharmStoreParams, repoPath string, conf *config.Config) (*charm.URL, charmrepo.Interface, error) {
	ref, err := charm.ParseReference(curlStr)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	repo, err := charmrepo.InferRepository(ref, csParams, repoPath)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	repo = config.SpecializeCharmRepo(repo, conf)
	if ref.Series == "" {
		if defaultSeries, ok := conf.DefaultSeries(); ok {
			ref.Series = defaultSeries
		}
	}
	if ref.Schema == "local" && ref.Series == "" {
		possibleURL := *ref
		possibleURL.Series = "trusty"
		logger.Errorf("The series is not specified in the environment (default-series) or with the charm. Did you mean:\n\t%s", &possibleURL)
		return nil, nil, errors.Errorf("cannot resolve series for charm: %q", ref)
	}
	if ref.Series != "" && ref.Revision != -1 {
		// The URL is already fully resolved; do not
		// bother with an unnecessary round-trip to the
		// charm store.
		curl, err := ref.URL("")
		if err != nil {
			panic(err)
		}
		return curl, repo, nil
	}
	curl, err := repo.Resolve(ref)
	if err != nil {
		return nil, nil, errors.Trace(err)
	}
	return curl, repo, nil
}

// AddCharmViaAPI calls the appropriate client API calls to add the
// given charm URL to state. For non-public charm URLs, this function also
// handles the macaroon authorization process using the given CsClient.
// The resulting charm URL of the added charm is displayed on stdout.
func AddCharmViaAPI(client *api.Client, ctx *cmd.Context, curl *charm.URL, repo charmrepo.Interface, csclient *CsClient) (*charm.URL, error) {
	switch curl.Schema {
	case "local":
		ch, err := repo.Get(curl)
		if err != nil {
			return nil, err
		}
		stateCurl, err := client.AddLocalCharm(curl, ch)
		if err != nil {
			return nil, err
		}
		curl = stateCurl
	case "cs":
		if err := client.AddCharm(curl); err != nil {
			if !params.IsCodeUnauthorized(err) {
				return nil, errors.Mask(err)
			}
			m, err := csclient.authorize(curl)
			if err != nil {
				return nil, errors.Mask(err)
			}
			if err := client.AddCharmWithAuthorization(curl, m); err != nil {
				return nil, errors.Mask(err)
			}
		}
	default:
		return nil, fmt.Errorf("unsupported charm URL schema: %q", curl.Schema)
	}
	ctx.Infof("Added charm %q to the environment.", curl)
	return curl, nil
}

// CsClient gives access to the charm store server and provides parameters
// for connecting to the charm store.
type CsClient struct {
	jar    *cookiejar.Jar
	params charmrepo.NewCharmStoreParams
}

// NewCharmStoreClient is called to obtain a charm store client
// including the parameters for connecting to the charm store, and
// helpers to save the local authorization cookies and to authorize
// non-public charm deployments. It is defined as a variable so it can
// be changed for testing purposes.
var NewCharmStoreClient = func() (*CsClient, error) {
	jar, client, err := newHTTPClient()
	if err != nil {
		return nil, errors.Mask(err)
	}
	return &CsClient{
		jar: jar,
		params: charmrepo.NewCharmStoreParams{
			HTTPClient:   client,
			VisitWebPage: httpbakery.OpenWebBrowser,
		},
	}, nil
}

func newHTTPClient() (*cookiejar.Jar, *http.Client, error) {
	cookieFile := path.Join(utils.Home(), ".go-cookies")
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		panic(err)
	}
	if err := jar.Load(cookieFile); err != nil {
		return nil, nil, err
	}
	client := httpbakery.NewHTTPClient()
	client.Jar = jar
	return jar, client, nil
}

// authorize acquires and return the charm store delegatable macaroon to be
// used to add the charm corresponding to the given URL.
// The macaroon is properly attenuated so that it can only be used to deploy
// the given charm URL.
func (c *CsClient) authorize(curl *charm.URL) (*macaroon.Macaroon, error) {
	client := csclient.New(csclient.Params{
		URL:          c.params.URL,
		HTTPClient:   c.params.HTTPClient,
		VisitWebPage: c.params.VisitWebPage,
	})
	var m *macaroon.Macaroon
	if err := client.Get("/delegatable-macaroon", &m); err != nil {
		return nil, errors.Trace(err)
	}
	if err := m.AddFirstPartyCaveat("is-entity " + curl.String()); err != nil {
		return nil, errors.Trace(err)
	}
	return m, nil
}

// SaveJar saves the cookie jar.
func (c *CsClient) SaveJar() error {
	return c.jar.Save()
}

// Jar returns the cookie jar.
func (c *CsClient) Jar() *cookiejar.Jar {
	return c.jar
}

// Params returns the params.
func (c *CsClient) Params() charmrepo.NewCharmStoreParams {
	return c.params
}

func (c *CsClient) SetUrl(url string) {
	c.params.URL = url
}
