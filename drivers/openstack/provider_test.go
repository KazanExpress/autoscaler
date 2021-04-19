// Copyright 2018 Drone.IO Inc
// Use of this source code is governed by the Polyform License
// that can be found in the LICENSE file.

package openstack

import (
	"testing"

	"github.com/gophercloud/gophercloud"
)

func TestDefaults(t *testing.T) {
	v, err := New(
		WithComputeClient(&gophercloud.ServiceClient{}),
		WithNetworkClient(&gophercloud.ServiceClient{}),
	)
	if err != nil {
		t.Error(err)
		return
	}
	p := v.(*provider)
	// Add tests if we set some actual defaults in the future.
	_ = p
}
