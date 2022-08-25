package controller

import (
	"fmt"
	"time"

	"github.com/forbearing/k8s-loadbalancer/pkg/nginx"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
)

type Controller struct {
	clientset     kubernetes.Interface
	serviceLister corelisters.ServiceLister
	serviceSynced cache.InformerSynced
	workqueue     workqueue.RateLimitingInterface
	recorder      record.EventRecorder
}

func NewController(clientset kubernetes.Interface, serviceInformer coreinformers.ServiceInformer) *Controller {
	controller := &Controller{
		clientset:     clientset,
		serviceLister: serviceInformer.Lister(),
		serviceSynced: serviceInformer.Informer().HasSynced,
		workqueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "loadbalancer"),
	}

	logrus.Info("Setting up event handlers")
	serviceInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueService,
		UpdateFunc: func(old, new interface{}) {
			newSvc := new.(*corev1.Service)
			oldSvc := old.(*corev1.Service)
			if newSvc.ResourceVersion == oldSvc.ResourceVersion {
				return
			}
			controller.enqueueService(new)
		},
		DeleteFunc: controller.enqueueService,
	})

	return controller
}

// Run
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	logrus.Info("Starting loadbalancer controller")

	logrus.Info("Waiting for informe cache to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.serviceSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	logrus.Info("Starting workers")
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorkers, time.Second, stopCh)
	}

	logrus.Info("Started workers")
	<-stopCh
	logrus.Info("Shutting down workers")

	return nil
}

// runWorkers
func (c *Controller) runWorkers() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}

	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
		}
		if err := c.handleNginx(key); err != nil {
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		c.workqueue.Forget(obj)
		logrus.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// handleNginx
func (c *Controller) handleNginx(key string) error {
	//logrus.Infof("%s", key)
	//return nginx.Install()
	return nginx.Uninstall()
}

// enqueueService
func (c *Controller) enqueueService(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}
