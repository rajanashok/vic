// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package session caches vSphere objects to avoid having to repeatedly
// make govmomi client calls.
//
// To obtain a Session, call Create with a Config. The config
// contains the SDK URL (Service) and the desired vSphere resources.
// Create then connects to Service and stores govmomi objects for
// each corresponding value in Config. The Session is returned and
// the user can use the cached govmomi objects in the exported fields of
// Session instead of directly using a govmomi Client.
//
package session

import (
	"crypto/tls"
	"net/url"
	"time"

	"golang.org/x/net/context"

	"github.com/juju/errors"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/vim25/soap"
)

// Config contains the configuration used to create a Session.
type Config struct {
	Service   string // SDK URL or proxy
	Insecure  bool   // Allow insecure connection to Service
	Keepalive time.Duration

	Cluster    string
	Datacenter string
	Datastore  string
	Host       string
	Network    string
	Pool       string

	CertFile string
	KeyFile  string
}

// HasCertificate checks for presence of a certificate and keyfile
func (c *Config) HasCertificate() bool {
	return c.CertFile != "" && c.KeyFile != ""
}

// Session caches vSphere objects obtained by querying the SDK.
type Session struct {
	*govmomi.Client

	Cluster    *object.ComputeResource
	Datacenter *object.Datacenter
	Datastore  *object.Datastore
	Host       *object.HostSystem
	Network    object.NetworkReference
	Pool       *object.ResourcePool
}

// Create accepts a Config and returns a Session with the cached vSphere resources.
func Create(config Config) (*Session, error) {
	s := &Session{}
	var err error
	var user *url.Userinfo
	var ctx = context.Background()

	soapURL, err := soap.ParseURL(config.Service)
	if err != nil {
		return nil, errors.Errorf("SDK URL (%s) could not be parsed: %s", config.Service, err)
	}

	// we can't set a keep alive if we log in directly with client creation
	user = soapURL.User
	soapURL.User = nil

	// 1st connect without any userinfo to get the API type
	s.Client, err = govmomi.NewClient(ctx, soapURL, config.Insecure)
	if err != nil {
		return nil, errors.Errorf("Failed to connect to %s: %s", soapURL.String(), err)
	}

	if config.HasCertificate() {
		if !s.IsVC() {
			return nil, errors.Errorf("Certificate based authentication not yet supported with ESXi")
		}

		// load the certificates
		cert, err2 := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err2 != nil {
			return nil, errors.Errorf("Unable to load X509 key pair(%s,%s): %s", config.CertFile, config.KeyFile, err2)
		}

		// create the new client
		s.Client, err = govmomi.NewClientWithCertificate(ctx, soapURL, config.Insecure, cert)
		if err != nil {
			return nil, errors.Errorf("Failed to connect to %s: %s", soapURL.String(), err)
		}
	}

	if config.Keepalive != 0 {
		// now that we've verified everything, enable keepalive
		s.RoundTripper = session.KeepAlive(s.RoundTripper, config.Keepalive)
	}

	// and now that the keepalive is registered we can log in to trigger it
	if !config.HasCertificate() {
		err = s.Login(ctx, user)
	} else {
		err = s.LoginExtensionByCertificate(ctx, user.Username(), "")
	}
	if err != nil {
		return nil, errors.Errorf("Failed to log in to %s: %s", soapURL.String(), err)
	}

	// Populate s
	finder := find.NewFinder(s.Client.Client, true)

	s.Datacenter, err = finder.DatacenterOrDefault(ctx, config.Datacenter)
	if err != nil {
		return nil, err
	}
	finder.SetDatacenter(s.Datacenter)

	s.Cluster, err = finder.ComputeResourceOrDefault(ctx, config.Cluster)
	if err != nil {
		return nil, err
	}

	s.Datastore, err = finder.DatastoreOrDefault(ctx, config.Datastore)
	if err != nil {
		return nil, err
	}

	s.Host, err = finder.HostSystemOrDefault(ctx, config.Host)
	if err != nil {
		if _, ok := err.(*find.DefaultMultipleFoundError); !ok || !s.IsVC() {
			return nil, err
		}
	}

	s.Network, err = finder.NetworkOrDefault(ctx, config.Network)
	if err != nil {
		return nil, err
	}

	s.Pool, err = finder.ResourcePoolOrDefault(ctx, config.Pool)
	if err != nil {
		return nil, err
	}

	return s, nil
}
