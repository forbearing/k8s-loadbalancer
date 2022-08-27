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
	logger.Init()

	////// just for debug
	//logrus.Debug(args.GetPort())
	//logrus.Debug(args.GetBindAddress())
	//logrus.Debug(args.GetKubeconfig())
	//logrus.Debug(args.GetLogLevel())
	//logrus.Debug(args.GetLogFormat())
	//logrus.Debug(args.GetLogFile())
	//logrus.Debug(args.GetUpstream())
	//logrus.Debug(args.GetNumWorker())

	handler := service.NewOrDie(context.Background(), args.GetKubeconfig(), metav1.NamespaceAll)
	stopCh := signals.SetupSignalChannel()
	ctrl := controller.NewController(handler)

	handler.InformerFactory().Start(stopCh)
	if err := ctrl.Run(args.GetNumWorker(), stopCh); err != nil {
		logrus.Fatal("Error running controller: %s", err.Error())
	}
}
