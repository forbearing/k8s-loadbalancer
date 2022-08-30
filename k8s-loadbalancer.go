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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	argPort       = pflag.Int("port", 8080, "port to listen to for incoming HTTP requests")
	argBindAddr   = pflag.IP("bind-address", net.IPv4(0, 0, 0, 0), "IP address on which to serve the --port, set to 0.0.0.0 for all interfaces by default")
	argKubeconfig = pflag.String("kubeconfig", "", "path to kubeconfig file with authorization and master location information")
	argLogLevel   = pflag.String("log-level", "INFO", "level of API request logging, should be one of   'ERROR', 'WARNING|WARN', 'INFO', 'DEBUG' or 'TRACE'")
	argLogFormat  = pflag.String("log-format", "TEXT", "specify log format, should be on of 'TEXT' or 'JSON'")
	argLogFile    = pflag.String("log-output", "/dev/stdout", "specify log file, default output log to /dev/stdout")
	argUpstream   = pflag.StringSlice("upstream", []string{}, "multiple upstream hosts or IP to which the loadbalancer will proxy traffic, separated by common, eg: --upstream host1,host2,host3 or --upstream 1.1.1.1,2.2.2.2,3.3.3.3 ")
	argNumWorker  = pflag.Int("worker", runtime.NumCPU(), "the number of worker goroutines to handle k8s service resources and nginx daemon, default to the number of cpu")
	//argEnableFirewall = pflag.Bool("enable-firewall", false, "whether enable ufw for debian/ubuntu and firewalld for rocky/centos, default to false")
	//argConfPath = pflag.String("conf", "", "the configuration file path")
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
	// init the log-level, log-format, log-output according to the arguments.
	// you can also call logger.New() to get a new *logrus.Logger that is not
	// affected by logger.Init().
	logger.Init()

	// service.NewOrDie will creates a handler which with various methods to
	// make it  easily to operators k8s service resource in golang coding.
	// panic if create service handler failed.
	handler := service.NewOrDie(context.Background(), args.GetKubeconfig(), metav1.NamespaceAll)

	// SetInformerFactoryResyncPeriod  set the period of the infomer relist
	// k8s service resource object, default to 0(no resync).
	handler.SetInformerFactoryResyncPeriod(0)

	// SetupSignalChannel will creates a chan struct{} and it will receive a element
	// when this controller captured Ctrl-C or SIGTERM signal.
	// If stopCh receive a element, the service informer and the workers created by
	// this controller will stop work.
	stopCh := signals.SetupSignalChannel()
	ctrl := controller.NewController(handler)

	// start the shared informer.
	handler.InformerFactory().Start(stopCh)
	// block here until this controller capture SIGINT or SIGTERM signal.
	if err := ctrl.Run(args.GetNumWorker(), stopCh); err != nil {
		logrus.Fatal("Error running controller: %s", err.Error())
	}
}
