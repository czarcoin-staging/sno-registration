// Copyright (C) 2020 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/common/fpath"
	"storj.io/private/cfgstruct"
	"storj.io/private/process"
	"storj.io/snoregistration"
)

// Config contains configurable values for sno registration service.
type Config struct {
	SignUp snoregistration.Config
}

var (
	rootCmd = &cobra.Command{
		Use:   "snoregistration",
		Short: "snoregistration",
	}
	runCmd = &cobra.Command{
		Use:         "run",
		Short:       "runs the program",
		RunE:        cmdRun,
		Annotations: map[string]string{"type": "run"},
	}
	setupCmd = &cobra.Command{
		Use:         "setup",
		Short:       "setups the program",
		RunE:        cmdSetup,
		Annotations: map[string]string{"type": "setup"},
	}

	runConfig        Config
	setupConfig      Config
	defaultConfigDir = fpath.ApplicationDir("storj", "snoregistration")
)

func main() {
	process.Exec(rootCmd)
}

func init() {
	cfgstruct.SetupFlag(zap.L(), rootCmd, &defaultConfigDir, "config-dir", defaultConfigDir, "main directory for sno registration service configuration")
	defaults := cfgstruct.DefaultsFlag(rootCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(setupCmd)
	process.Bind(runCmd, &runConfig, cfgstruct.ConfDir(defaultConfigDir))
	process.Bind(setupCmd, &setupConfig, defaults, cfgstruct.SetupMode())
}

func cmdRun(cmd *cobra.Command, args []string) (err error) {
	ctx, _ := process.Ctx(cmd)
	log := zap.L()

	peer, err := snoregistration.New(log, runConfig.SignUp)
	if err != nil {
		return err
	}

	runError := peer.Run(ctx)
	closeError := peer.Close()
	return errs.Combine(runError, closeError)
}

func cmdSetup(cmd *cobra.Command, args []string) (err error) {
	setupDir, err := filepath.Abs(defaultConfigDir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(setupDir, os.ModePerm)
	if err != nil {
		return err
	}

	overrides := map[string]interface{}{
		"log.level": "info",
	}

	configFile := filepath.Join(setupDir, "config.yaml")
	err = process.SaveConfig(cmd, configFile, process.SaveConfigWithOverrides(overrides))
	if err != nil {
		return err
	}

	return err
}
