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

package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/kubewg-net/container/internal/config"
	"github.com/kubewg-net/container/internal/metrics"
	"github.com/kubewg-net/container/internal/pprof"
	"github.com/spf13/cobra"
	"github.com/ztrue/shutdown"
	"golang.org/x/sync/errgroup"
)

func NewCommand(version, commit string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "container",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Annotations: map[string]string{
			"version": version,
			"commit":  commit,
		},
		RunE:          run,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	config.RegisterFlags(cmd)
	return cmd
}

func run(cmd *cobra.Command, _ []string) error {
	slog.Info("kubewg container", "version", cmd.Annotations["version"], "commit", cmd.Annotations["commit"])

	config, err := config.LoadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var metricsServer *metrics.Server
	var pprofServer *pprof.Server

	// Start the metrics server
	if config.Metrics.Enabled {
		slog.Info("Starting metrics server")
		metricsServer = metrics.NewServer(&config.Metrics)
		go metricsServer.Start()
	}

	// Start the pprof server
	if config.PProf.Enabled {
		slog.Info("Starting pprof server")
		pprofServer = pprof.NewServer(&config.PProf)
		go pprofServer.Start()
	}

	stop := func(sig os.Signal) {
		slog.Info("Shutting down", "signal", sig.String())
		errGrp := errgroup.Group{}

		if metricsServer != nil {
			errGrp.Go(func() error {
				return metricsServer.Stop()
			})
		}

		if pprofServer != nil {
			errGrp.Go(func() error {
				return pprofServer.Stop()
			})
		}

		if err := errGrp.Wait(); err != nil {
			slog.Error("Error shutting down", "error", err.Error())
			os.Exit(1)
		}

		slog.Info("Shutdown complete")
	}

	shutdown.AddWithParam(stop)
	shutdown.Listen(syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)

	return nil
}
