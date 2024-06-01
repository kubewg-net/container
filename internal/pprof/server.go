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

package pprof

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/kubewg-net/container/internal/config"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	ipv4Server *http.Server
	ipv6Server *http.Server
	stopped    bool
	config     *config.PProf
}

func NewServer(config *config.PProf) *Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP)
	mux.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
	mux.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	mux.HandleFunc("/debug/pprof/heap", pprof.Handler("heap").ServeHTTP)
	mux.HandleFunc("/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
	mux.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)

	return &Server{
		ipv4Server: &http.Server{
			Addr:              fmt.Sprintf("%s:%d", config.IPV4Host, config.Port),
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           mux,
		},
		ipv6Server: &http.Server{
			Addr:              fmt.Sprintf("[%s]:%d", config.IPV6Host, config.Port),
			ReadHeaderTimeout: 5 * time.Second,
			Handler:           mux,
		},
		config: config,
	}
}

func (s *Server) Start() {
	waitGrp := sync.WaitGroup{}
	waitGrp.Add(1)
	go func() {
		defer waitGrp.Done()
		if err := s.ipv4Server.ListenAndServe(); err != nil && !s.stopped {
			slog.Error("PProf server error", "error", err.Error())
		}
	}()

	waitGrp.Add(1)
	go func() {
		defer waitGrp.Done()
		if err := s.ipv6Server.ListenAndServe(); err != nil && !s.stopped {
			slog.Error("PProf server error", "error", err.Error())
		}
	}()

	slog.Info("PProf server started", "ipv4", s.config.IPV4Host, "ipv6", s.config.IPV6Host, "port", s.config.Port)

	waitGrp.Wait()
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.stopped = true

	errGrp := errgroup.Group{}
	if s.ipv4Server != nil {
		errGrp.Go(func() error {
			return s.ipv4Server.Shutdown(ctx)
		})
	}
	if s.ipv6Server != nil {
		errGrp.Go(func() error {
			return s.ipv6Server.Shutdown(ctx)
		})
	}

	return errGrp.Wait()
}
