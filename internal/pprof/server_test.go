// SPDX-License-Identifier: AGPL-3.0-or-later
// KubeWG - Wireguard in your Kubernetes cluster
// Copyright (C) 2024 Jacob McSwain
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.
//
// The source code is available at <https://github.com/kubewg-net/container>.

package pprof_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/kubewg-net/container/internal/config"
	"github.com/kubewg-net/container/internal/pprof"
)

func TestServer(t *testing.T) {
	t.Parallel()
	config := &config.PProf{
		Enabled: true,
		HTTPListener: config.HTTPListener{
			IPV4Host: config.DefaultPProfIPV4Host,
			IPV6Host: config.DefaultPProfIPV6Host,
			Port:     config.DefaultPProfPort,
		},
	}
	pprofServer := pprof.NewServer(config)
	if pprofServer == nil {
		t.Fatal("expected PProf server to be created")
	}

	go pprofServer.Start()
	time.Sleep(1 * time.Second)

	httpClient := http.Client{
		Timeout: 60 * time.Second,
	}

	pprofResponse, err := httpClient.Get(fmt.Sprintf("http://%s:%d/debug/pprof/", config.IPV4Host, config.Port))
	if err != nil {
		t.Fatalf("expected PProf server to be reachable via IPv4: %v", err)
	}

	if pprofResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected PProf response to be successful: %d", pprofResponse.StatusCode)
	}

	pprofResponse, err = httpClient.Get(fmt.Sprintf("http://[%s]:%d/debug/pprof/", config.IPV6Host, config.Port))
	if err != nil {
		t.Fatalf("expected PProf server to be reachable via IPv6: %v", err)
	}

	if pprofResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected PProf response to be successful: %d", pprofResponse.StatusCode)
	}

	err = pprofServer.Stop()
	if err != nil {
		t.Fatalf("expected PProf server to stop: %v", err)
	}

	pprofResponse, err = httpClient.Get(fmt.Sprintf("http://%s:%d/debug/pprof/", config.IPV4Host, config.Port))
	if err == nil {
		t.Fatalf("expected PProf server to be unreachable")
	}

	if pprofResponse != nil {
		t.Fatalf("expected PProf response to be nil: %v", pprofResponse)
	}
}
