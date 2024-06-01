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

package metrics_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/kubewg-net/container/internal/config"
	"github.com/kubewg-net/container/internal/metrics"
)

func TestServer(t *testing.T) {
	t.Parallel()
	config := &config.Metrics{
		Enabled: true,
		HTTPListener: config.HTTPListener{
			IPV4Host: config.DefaultMetricsIPV4Host,
			IPV6Host: config.DefaultMetricsIPV6Host,
			Port:     config.DefaultMetricsPort,
		},
	}
	metricsServer := metrics.NewServer(config)
	if metricsServer == nil {
		t.Fatal("expected metrics server to be created")
	}

	go metricsServer.Start()
	time.Sleep(1 * time.Second)

	httpClient := http.Client{
		Timeout: 60 * time.Second,
	}

	metricsResponse, err := httpClient.Get(fmt.Sprintf("http://%s:%d/metrics", config.IPV4Host, config.Port))
	if err != nil {
		t.Fatalf("expected metrics server to be reachable via IPv4: %v", err)
	}

	if metricsResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected metrics response to be successful: %d", metricsResponse.StatusCode)
	}

	metricsResponse, err = httpClient.Get(fmt.Sprintf("http://[%s]:%d/metrics", config.IPV6Host, config.Port))
	if err != nil {
		t.Fatalf("expected metrics server to be reachable via IPv6: %v", err)
	}

	if metricsResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected metrics response to be successful: %d", metricsResponse.StatusCode)
	}

	err = metricsServer.Stop()
	if err != nil {
		t.Fatalf("expected metrics server to stop: %v", err)
	}

	metricsResponse, err = httpClient.Get(fmt.Sprintf("http://%s:%d/metrics", config.IPV4Host, config.Port))
	if err == nil {
		t.Fatalf("expected metrics server to be unreachable")
	}

	if metricsResponse != nil {
		t.Fatalf("expected metrics response to be nil: %v", metricsResponse)
	}
}
