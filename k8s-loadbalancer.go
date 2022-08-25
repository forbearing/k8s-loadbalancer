package main

import (
	"context"
	"flag"
	"log"
	"net"

	"github.com/forbearing/k8s"
	"github.com/forbearing/k8s/service"
	"github.com/forbearing/k8s/util/signals"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type updateObj struct {
	oldObj interface{}
	newObj interface{}
}

var (
	argPort           = pflag.Uint("port", 8080, "port to listen to for incoming HTTP requests")
	argBindAddr       = pflag.IP("bind-address", net.IPv4(0, 0, 0, 0), "IP address on which to serve the --port, set to 0.0.0.0 for all interfaces by default")
	argKubeConfigFile = pflag.String("kubeconfig", "", "path to kubeconfig file with authorization and master location information")
	// argLogLevel
	// argLogFormat
	// argLogFile
)

func main() {
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	clientset := k8s.ClientsetOrDie(*argKubeConfigFile)
	stopCh := signals.SetupSignalChannel()
	_ = clientset
	_ = stopCh

	handler := service.NewOrDie(context.Background(), *argKubeConfigFile, "")

	addQueue := make(chan interface{}, 100)
	updateQueue := make(chan updateObj, 100)
	deleteQueue := make(chan interface{}, 100)

	addFunc := func(obj interface{}) { addQueue <- obj }
	updateFunc := func(oldObj interface{}, newObj interface{}) {
		uo := updateObj{}
		uo.oldObj = oldObj
		uo.newObj = newObj
		updateQueue <- uo
	}
	deleteFunc := func(obj interface{}) { deleteQueue <- obj }

	go func() {
		handler.RunInformer(stopCh, addFunc, updateFunc, deleteFunc)
	}()

	for {
		select {
		case obj := <-addQueue:
			myObj := obj.(metav1.Object)
			log.Println("Add: ", myObj.GetName())
		case ou := <-updateQueue:
			oldObj := ou.oldObj.(*corev1.Service)
			curObj := ou.newObj.(*corev1.Service)
			if oldObj.ResourceVersion == curObj.ResourceVersion {
				return
			}
			log.Println("Update:", curObj.Name)
		case obj := <-deleteQueue:
			myObj := obj.(metav1.Object)
			log.Println("Delete", myObj.GetName())
		case <-stopCh:
			log.Println("Informer stopped.")
			return
		}

	}
}
