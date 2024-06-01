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

package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type HTTPListener struct {
	IPV4Host string `json:"ipv4_host"`
	IPV6Host string `json:"ipv6_host"`
	Port     uint16 `json:"port"`
}

type Tracing struct {
	Enabled      bool   `json:"enabled"`
	OTLPEndpoint string `json:"otlp_endpoint"`
}

type PProf struct {
	HTTPListener
	Enabled bool `json:"enabled"`
}

type Metrics struct {
	HTTPListener
	Enabled bool `json:"enabled"`
}

// Config is the main configuration for the application
type Config struct {
	Tracing
	PProf   PProf   `json:"pprof"`
	Metrics Metrics `json:"metrics"`
}

//nolint:golint,gochecknoglobals
var (
	ConfigFileKey      = "config"
	TracingEnabledKey  = "tracing.enabled"
	TracingOTLPEndKey  = "tracing.otlp_endpoint"
	PProfEnabledKey    = "pprof.enabled"
	PProfIPV4HostKey   = "pprof.ipv4_host"
	PProfIPV6HostKey   = "pprof.ipv6_host"
	PProfPortKey       = "pprof.port"
	MetricsEnabledKey  = "metrics.enabled"
	MetricsIPV4HostKey = "metrics.ipv4_host"
	MetricsIPV6HostKey = "metrics.ipv6_host"
	MetricsPortKey     = "metrics.port"
)

const (
	DefaultConfigName      = "config.yaml"
	DefaultMetricsIPV4Host = "127.0.0.1"
	DefaultMetricsIPV6Host = "::1"
	DefaultMetricsPort     = 8081
	DefaultPprofIPV4Host   = "127.0.0.1"
	DefaultPprofIPV6Host   = "::1"
	DefaultPprofPort       = 6060
)

func RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(ConfigFileKey, "c", DefaultConfigName, "Config file path")
	cmd.Flags().Bool(TracingEnabledKey, false, "Enable Open Telemetry tracing")
	cmd.Flags().String(TracingOTLPEndKey, "", "Open Telemetry endpoint")
	cmd.Flags().Bool(PProfEnabledKey, false, "Enable PProf")
	cmd.Flags().String(PProfIPV4HostKey, DefaultMetricsIPV4Host, "PProf server IPv4 host")
	cmd.Flags().String(PProfIPV6HostKey, DefaultMetricsIPV6Host, "PProf server IPv6 host")
	cmd.Flags().Uint16(PProfPortKey, DefaultMetricsPort, "PProf server port")
	cmd.Flags().Bool(MetricsEnabledKey, false, "Enable metrics server")
	cmd.Flags().String(MetricsIPV4HostKey, DefaultMetricsIPV4Host, "Metrics server IPv4 host")
	cmd.Flags().String(MetricsIPV6HostKey, DefaultMetricsIPV6Host, "Metrics server IPv6 host")
	cmd.Flags().Uint16(MetricsPortKey, DefaultMetricsPort, "Metrics server port")
}

func (c *Config) Validate() error {
	return nil
}

//nolint:golint,gocyclo
func LoadConfig(cmd *cobra.Command) (*Config, error) {
	var config Config

	// Load flags from envs
	ctx, cancel := context.WithCancelCause(cmd.Context())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if ctx.Err() != nil {
			return
		}
		optName := strings.ReplaceAll(strings.ToUpper(f.Name), ".", "__")
		if val, ok := os.LookupEnv(optName); !f.Changed && ok {
			if err := f.Value.Set(val); err != nil {
				cancel(err)
			}
			f.Changed = true
		}
	})
	if ctx.Err() != nil {
		return &config, fmt.Errorf("failed to load env: %w", context.Cause(ctx))
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return &config, fmt.Errorf("failed to get config path: %w", err)
	}
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if errors.Is(err, os.ErrNotExist) && configPath == DefaultConfigName {
			// We can ignore this error if the default config file is not found
			return &config, nil
		} else if err != nil {
			return &config, fmt.Errorf("failed to read config: %w", err)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return &config, fmt.Errorf("failed to unmarshal config: %w", err)
		}
	}

	// Flag overrides here
	if cmd.Flags().Changed(PProfEnabledKey) {
		config.PProf.Enabled, err = cmd.Flags().GetBool(PProfEnabledKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get pprof enabled: %w", err)
		}
	}

	if cmd.Flags().Changed(PProfIPV4HostKey) {
		config.PProf.IPV4Host, err = cmd.Flags().GetString(PProfIPV4HostKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get pprof IPv4 host: %w", err)
		}
	}

	if cmd.Flags().Changed(PProfIPV6HostKey) {
		config.PProf.IPV6Host, err = cmd.Flags().GetString(PProfIPV6HostKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get pprof IPv6 host: %w", err)
		}
	}

	if cmd.Flags().Changed(PProfPortKey) {
		config.PProf.Port, err = cmd.Flags().GetUint16(PProfPortKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get pprof port: %w", err)
		}
	}

	if cmd.Flags().Changed(MetricsEnabledKey) {
		config.Metrics.Enabled, err = cmd.Flags().GetBool(MetricsEnabledKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get metrics enabled: %w", err)
		}
	}

	if cmd.Flags().Changed(MetricsIPV4HostKey) {
		config.Metrics.IPV4Host, err = cmd.Flags().GetString(MetricsIPV4HostKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get metrics IPv4 host: %w", err)
		}
	}

	if cmd.Flags().Changed(MetricsIPV6HostKey) {
		config.Metrics.IPV6Host, err = cmd.Flags().GetString(MetricsIPV6HostKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get metrics IPv6 host: %w", err)
		}
	}

	if cmd.Flags().Changed(MetricsPortKey) {
		config.Metrics.Port, err = cmd.Flags().GetUint16(MetricsPortKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get metrics port: %w", err)
		}
	}

	if cmd.Flags().Changed(TracingEnabledKey) {
		config.Tracing.Enabled, err = cmd.Flags().GetBool(TracingEnabledKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get tracing enabled: %w", err)
		}
	}

	if cmd.Flags().Changed(TracingOTLPEndKey) {
		config.Tracing.OTLPEndpoint, err = cmd.Flags().GetString(TracingOTLPEndKey)
		if err != nil {
			return &config, fmt.Errorf("failed to get tracing OTLP endpoint: %w", err)
		}
	}

	// Defaults
	if config.Metrics.IPV4Host == "" {
		config.Metrics.IPV4Host = DefaultMetricsIPV4Host
	}
	if config.Metrics.IPV6Host == "" {
		config.Metrics.IPV6Host = DefaultMetricsIPV6Host
	}
	if config.Metrics.Port == 0 {
		config.Metrics.Port = DefaultMetricsPort
	}
	if config.PProf.IPV4Host == "" {
		config.PProf.IPV4Host = DefaultPprofIPV4Host
	}
	if config.PProf.IPV6Host == "" {
		config.PProf.IPV6Host = DefaultPprofIPV6Host
	}
	if config.PProf.Port == 0 {
		config.PProf.Port = DefaultPprofPort
	}

	err = config.Validate()
	if err != nil {
		return &config, fmt.Errorf("failed to validate config: %w", err)
	}

	return &config, nil
}
