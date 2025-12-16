#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

PROJ_ROOT=$(cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)
GO_MODULE=$(go list -m)

if [ $# -lt 3 ]; then
  echo "Usage: $0 <GROUP> <VERSION> <KIND>"
  echo "Example: $0 sentinel v1 Sentinel"
  exit 1
fi

GROUP=$1
VERSION=$2
KIND=$3

# Helpers
# lower case kind: Sentinel -> sentinel
LOWER_KIND=$(echo "${KIND}" | tr '[:upper:]' '[:lower:]')
# plural kind (naive): sentinel -> sentinels
PLURAL_KIND="${LOWER_KIND}s"

# Target directory
CONTROLLER_DIR="${PROJ_ROOT}/internal/controller/${GROUP}"
mkdir -p "${CONTROLLER_DIR}"
TARGET_FILE="${CONTROLLER_DIR}/controller.go"

if [ -f "${TARGET_FILE}" ]; then
  if [ "${FORCE:-false}" != "true" ]; then
    echo "Error: ${TARGET_FILE} already exists."
    echo "Set FORCE=true to overwrite."
    exit 1
  else
    echo "Warning: Overwriting ${TARGET_FILE}..."
  fi
fi

echo "Scaffolding controller for ${GROUP}/${VERSION} ${KIND} in ${TARGET_FILE}..."

cat <<EOF > "${TARGET_FILE}"
package ${GROUP}

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	clientset "${GO_MODULE}/pkg/generated/clientset/versioned"
	samplescheme "${GO_MODULE}/pkg/generated/clientset/versioned/scheme"
	informers "${GO_MODULE}/pkg/generated/informers/externalversions/${GROUP}/${VERSION}"
	listers "${GO_MODULE}/pkg/generated/listers/${GROUP}/${VERSION}"
)

const controllerAgentName = "${LOWER_KIND}-controller"

const (
	// SuccessSynced is used as part of the Event 'reason' when a ${KIND} is synced
	SuccessSynced = "Synced"
	// ErrResourceExists is used as part of the Event 'reason' when a ${KIND} fails
	// to sync due to a Deployment of the same name already existing.
	ErrResourceExists = "ErrResourceExists"

	// MessageResourceSynced is the message used for an Event fired when a ${KIND}
	// is synced successfully
	MessageResourceSynced = "${KIND} synced successfully"
)

// Controller is the controller implementation for ${KIND} resources
type Controller struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// ${LOWER_KIND}clientset is a clientset for our own API group
	${LOWER_KIND}clientset clientset.Interface

	${PLURAL_KIND}Lister listers.${KIND}Lister
	${PLURAL_KIND}Synced cache.InformerSynced

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder
}

// NewController returns a new ${LOWER_KIND} controller
func NewController(
	kubeclientset kubernetes.Interface,
	${LOWER_KIND}clientset clientset.Interface,
	${LOWER_KIND}Informer informers.${KIND}Informer) *Controller {

	// Create event broadcaster
	// Add ${LOWER_KIND}-controller types to the default Kubernetes Scheme so Events can be
	// logged for ${LOWER_KIND}-controller types.
	utilruntime.Must(samplescheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	controller := &Controller{
		kubeclientset:     kubeclientset,
		${LOWER_KIND}clientset: ${LOWER_KIND}clientset,
		${PLURAL_KIND}Lister:   ${LOWER_KIND}Informer.Lister(),
		${PLURAL_KIND}Synced:   ${LOWER_KIND}Informer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "${KIND}s"),
		recorder:          recorder,
	}

	klog.Info("Setting up event handlers")
	// Set up an event handler for when ${KIND} resources change
	${LOWER_KIND}Informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueue${KIND},
		UpdateFunc: func(old, new interface{}) {
			controller.enqueue${KIND}(new)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(workers int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting ${KIND} controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.${PLURAL_KIND}Synced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process ${KIND} resources
	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		// We call Done here so the workqueue knows we have finished
		// processing this item. We also must remember to call Forget if we
		// do not want this work item being re-queued. For example, we do
		// not call Forget if a transient error occurs, instead the item is
		// put back on the workqueue and attempted again after a back-off
		// period.
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		// We expect strings to come off the workqueue. These are of the
		// form namespace/name. We do this as the delayed nature of the
		// workqueue means the items in the informer cache may actually be
		// more up to date that when the item was initially put onto the
		// workqueue.
		if key, ok = obj.(string); !ok {
			// As the item in the workqueue is actually invalid, we call
			// Forget here else we'd go into a loop of attempting to
			// process a work item that is invalid.
			c.workqueue.Forget(obj)
			utilruntime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		// Run the syncHandler, passing it the namespace/name string of the
		// ${KIND} resource to be synced.
		if err := c.syncHandler(key); err != nil {
			// Put the item back on the workqueue to handle any transient errors.
			c.workqueue.AddRateLimited(key)
			return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
		}
		// Finally, if no error occurs we Forget this item so it does not
		// get queued again until another change happens.
		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced '%s'", key)
		return nil
	}(obj)

	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the ${KIND} resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the ${KIND} resource with this namespace/name
	${LOWER_KIND}, err := c.${PLURAL_KIND}Lister.${KIND}s(namespace).Get(name)
	if err != nil {
		// The ${KIND} resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("${LOWER_KIND} '%s' in work queue no longer exists", key))
			return nil
		}

		return err
	}

	// TODO: HERE IS WHERE YOUR CONTROLLER LOGIC GOES
	// Example:
	// 1. Check if the Deployment exists (if you manage deployments)
	// 2. Create the Deployment if it doesn't exist
	// 3. Update the Deployment if it doesn't match the ${KIND} definition
	// 4. Update the ${KIND} status

	klog.Infof("Processing ${KIND}: %s/%s", ${LOWER_KIND}.Namespace, ${LOWER_KIND}.Name)

	c.recorder.Event(${LOWER_KIND}, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
	return nil
}

// enqueue${KIND} takes a ${KIND} resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than ${KIND}.
func (c *Controller) enqueue${KIND}(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}
EOF
echo "Done."
