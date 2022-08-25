package main

import (
	"context"
	"flag"
	"net"
	"runtime"

	"github.com/forbearing/k8s-loadbalancer/pkg/args"
	"github.com/forbearing/k8s-loadbalancer/pkg/controller"
	"github.com/forbearing/k8s-loadbalancer/pkg/logger"
	"github.com/forbearing/k8s/service"
	"github.com/forbearing/k8s/util/signals"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

var (
	argPort       = pflag.Uint("port", 8080, "port to listen to for incoming HTTP requests")
	argBindAddr   = pflag.IP("bind-address", net.IPv4(0, 0, 0, 0), "IP address on which to serve the --port, set to 0.0.0.0 for all interfaces by default")
	argKubeconfig = pflag.String("kubeconfig", "", "path to kubeconfig file with authorization and master location information")
	argLogLevel   = pflag.String("log-level", "INFO", "level of API request logging, should be one of   'ERROR', 'WARNING|WARN', 'INFO', 'DEBUG' or 'TRACE'")
	argLogFormat  = pflag.String("log-format", "TEXT", "specify log format, should be on of 'TEXT' or 'JSON'")
	argLogFile    = pflag.String("log-output", "/dev/stdout", "specify log file, default output log to /dev/stdout")
	argUpstream   = pflag.StringSlice("upstream", []string{}, "multiple upstream hosts or IP to which the loadbalancer will proxy traffic, separated by common, eg: --upstream host1,host2,host3 or --upstream 1.1.1.1,2.2.2.2,3.3.3.3 ")
	argNumWorker  = pflag.Uint("worker", uint(runtime.NumCPU()), "the number of worker goroutines to handle k8s service resources and nginx daemon, default to the number of cpu")
)

func init() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	builder := args.NewBuilder()
	builder.SetPort(*argPort)
	builder.SetBindAddress(*argBindAddr)
	builder.SetKubeconfig(*argKubeconfig)
	builder.SetLogLevel(*argLogLevel)
	builder.SetLogFormat(*argLogFormat)
	builder.SetLogFile(*argLogFile)
	builder.SetUpstream(*argUpstream)
	builder.SetNumWorker(*argNumWorker)
}

func main() {
	logger.Init()

	//// just for debug
	//logrus.Info(args.GetPort())
	//logrus.Info(args.GetBindAddress())
	//logrus.Info(args.GetKubeconfig())
	//logrus.Info(args.GetLogLevel())
	//logrus.Info(args.GetLogFormat())
	//logrus.Info(args.GetLogFile())
	//logrus.Info(args.GetUpstream())
	//logrus.Info(args.GetNumWorker())

	handler := service.NewOrDie(context.Background(), args.GetKubeconfig(), "")
	stopCh := signals.SetupSignalChannel()
	ctrl := controller.NewController(handler.Clientset(), handler.ServiceInformer())

	handler.InformerFactory().Start(stopCh)
	if err := ctrl.Run(2, stopCh); err != nil {
		logrus.Fatal("Err running controller: %s", err.Error())
	}
}
