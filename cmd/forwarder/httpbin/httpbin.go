package httpbin

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/saucelabs/forwarder"
	"github.com/saucelabs/forwarder/bind"
	"github.com/saucelabs/forwarder/log"
	"github.com/saucelabs/forwarder/log/stdlog"
	"github.com/saucelabs/forwarder/runctx"
	"github.com/saucelabs/forwarder/utils/httpbin"
	"github.com/spf13/cobra"
)

type command struct {
	httpServerConfig *forwarder.HTTPServerConfig
	apiServerConfig  *forwarder.HTTPServerConfig
	logConfig        *log.Config
}

func (c *command) RunE(cmd *cobra.Command, args []string) error {
	config := bind.DescribeFlags(cmd.Flags())

	if f := c.logConfig.File; f != nil {
		defer f.Close()
	}
	logger := stdlog.New(c.logConfig)
	logger.Debugf("configuration\n%s", config)

	s, err := forwarder.NewHTTPServer(c.httpServerConfig, httpbin.Handler(), logger.Named("server"))
	if err != nil {
		return err
	}

	r := prometheus.NewRegistry()
	a, err := forwarder.NewHTTPServer(c.apiServerConfig, forwarder.NewAPIHandler(r, s.Ready, config, ""), logger.Named("api"))
	if err != nil {
		return err
	}

	return runctx.Funcs{s.Run, a.Run}.Run()
}

func Command() (cmd *cobra.Command) {
	c := command{
		httpServerConfig: forwarder.DefaultHTTPServerConfig(),
		apiServerConfig:  forwarder.DefaultHTTPServerConfig(),
		logConfig:        log.DefaultConfig(),
	}
	c.apiServerConfig.Addr = "localhost:10000"

	defer func() {
		fs := cmd.Flags()
		bind.HTTPServerConfig(fs, c.httpServerConfig, "")
		bind.HTTPServerConfig(fs, c.apiServerConfig, "api")
		bind.LogConfig(fs, c.logConfig)
		bind.MarkFlagFilename(cmd, "cert-file", "key-file", "log-file")
	}()
	return &cobra.Command{
		Use:   "httpbin [--protocol <http|https|h2>] [--address <host:port>] [flags]",
		Short: "Start HTTP(S) server that serves httpbin.org API",
		RunE:  c.RunE,
	}
}
