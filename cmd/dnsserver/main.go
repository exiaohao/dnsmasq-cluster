package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/exiaohao/home-dns/pkg/dns"
	"github.com/spf13/cobra"
	"k8s.io/apiserver/pkg/util/logs"
)

var (
	serverOpts dns.InitOptions

	rootCmd = &cobra.Command{
		Use:          "dnsserver",
		Short:        "A mini dns server",
		Long:         "A mini DNS server with cache & spec config",
		SilenceUsage: true,
	}
	serverCmd = &cobra.Command{
		Use:   "run",
		Short: "A mini dns server",
		Long:  "A mini DNS server with cache & spec config",
		RunE: func(*cobra.Command, []string) error {
			logs.InitLogs()
			defer logs.FlushLogs()

			stopCh := setupSignalHandler()

			server := new(dns.Server)
			server.Initialize(serverOpts)
			server.Run(stopCh)

			return nil
		},
	}
)

func init() {
	serverCmd.PersistentFlags().StringVar(&serverOpts.ConfigFile, "config", "", "Path to config file")
	rootCmd.AddCommand(serverCmd)
}

func main() {
	flag.Parse()
	if err := rootCmd.Execute(); err != nil {
		os.Exit(-1)
	}
}

func setupSignalHandler() (stopCh <-chan struct{}) {
	stop := make(chan struct{})
	sigs := make(chan os.Signal, 2)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		close(stop)
		<-sigs
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}
