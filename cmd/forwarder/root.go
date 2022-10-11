// Copyright 2021 The forwarder Authors. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package main

import (
	"github.com/saucelabs/forwarder/cmd/forwarder/proxy"
	"github.com/saucelabs/forwarder/cmd/forwarder/version"
	"github.com/spf13/cobra"
)

const envPrefix = "FORWARDER"

func rootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "proxy",
		Short: "A simple flexible forward proxy",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return bindFlagsToEnv(cmd, envPrefix)
		},
	}

	rootCmd.AddCommand(
		proxy.Command(),
		version.Command(),
	)
	for _, cmd := range rootCmd.Commands() {
		appendEnvToUsage(cmd, envPrefix)
	}

	return rootCmd
}
