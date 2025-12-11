/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 *
 */

package snapshot

import (
	"fmt"
	"time"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	watchutil "kubevirt.io/kubevirt/pkg/virt-controller/watch/util"
)

// VMSnapshotScheduleController is responsible for scheduling VM snapshots
type VMSnapshotScheduleController struct {
	Client kubecli.KubevirtClient

	VMSnapshotScheduleInformer cache.SharedIndexInformer
	VMSnapshotInformer         cache.SharedIndexInformer
	VMInformer                 cache.SharedIndexInformer

	Recorder record.EventRecorder

	ResyncPeriod time.Duration

	scheduleQueue workqueue.TypedRateLimitingInterface[string]
}

// Init initializes the schedule controller
func (ctrl *VMSnapshotScheduleController) Init() error {
	ctrl.scheduleQueue = workqueue.NewTypedRateLimitingQueueWithConfig[string](
		workqueue.DefaultTypedControllerRateLimiter[string](),
		workqueue.TypedRateLimitingQueueConfig[string]{Name: "virt-controller-snapshot-schedule"},
	)

	_, err := ctrl.VMSnapshotScheduleInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMSnapshotSchedule,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMSnapshotSchedule(newObj) },
			DeleteFunc: ctrl.handleVMSnapshotSchedule,
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	// Watch VirtualMachineSnapshots to update schedule status
	_, err = ctrl.VMSnapshotInformer.AddEventHandlerWithResyncPeriod(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.handleVMSnapshotForSchedule,
			UpdateFunc: func(oldObj, newObj interface{}) { ctrl.handleVMSnapshotForSchedule(newObj) },
			DeleteFunc: ctrl.handleVMSnapshotForSchedule,
		},
		ctrl.ResyncPeriod,
	)
	if err != nil {
		return err
	}

	return nil
}

// Run starts the schedule controller
func (ctrl *VMSnapshotScheduleController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer ctrl.scheduleQueue.ShutDown()

	log.Log.Info("Starting snapshot schedule controller.")
	defer log.Log.Info("Shutting down snapshot schedule controller.")

	if !cache.WaitForCacheSync(
		stopCh,
		ctrl.VMSnapshotScheduleInformer.HasSynced,
		ctrl.VMSnapshotInformer.HasSynced,
		ctrl.VMInformer.HasSynced,
	) {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	for i := 0; i < threadiness; i++ {
		go wait.Until(ctrl.scheduleWorker, time.Second, stopCh)
	}

	<-stopCh
	return nil
}

func (ctrl *VMSnapshotScheduleController) scheduleWorker() {
	for ctrl.processScheduleWorkItem() {
	}
}

func (ctrl *VMSnapshotScheduleController) processScheduleWorkItem() bool {
	return watchutil.ProcessWorkItem(ctrl.scheduleQueue, func(key string) (time.Duration, error) {
		log.Log.V(3).Infof("Schedule worker processing key [%s]", key)

		storeObj, exists, err := ctrl.VMSnapshotScheduleInformer.GetStore().GetByKey(key)
		if err != nil {
			return 0, err
		}

		if !exists {
			log.Log.V(3).Infof("VirtualMachineSnapshotSchedule %s no longer exists", key)
			return 0, nil
		}

		schedule, ok := storeObj.(*snapshotv1.VirtualMachineSnapshotSchedule)
		if !ok {
			return 0, fmt.Errorf("unexpected resource %+v", storeObj)
		}

		return ctrl.updateVMSnapshotSchedule(schedule.DeepCopy())
	})
}

func (ctrl *VMSnapshotScheduleController) handleVMSnapshotSchedule(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	schedule, ok := obj.(*snapshotv1.VirtualMachineSnapshotSchedule)
	if !ok {
		log.Log.Errorf("unexpected resource: %+v", obj)
		return
	}

	key, err := cache.MetaNamespaceKeyFunc(schedule)
	if err != nil {
		log.Log.Errorf("failed to get key from object: %v, %v", schedule, err)
		return
	}

	log.Log.V(3).Infof("enqueued %q for sync", key)
	ctrl.scheduleQueue.Add(key)
}

func (ctrl *VMSnapshotScheduleController) handleVMSnapshotForSchedule(obj interface{}) {
	if unknown, ok := obj.(cache.DeletedFinalStateUnknown); ok && unknown.Obj != nil {
		obj = unknown.Obj
	}

	snapshot, ok := obj.(*snapshotv1.VirtualMachineSnapshot)
	if !ok {
		return
	}

	// Check if this snapshot belongs to a schedule
	if snapshot.Labels == nil {
		return
	}
	scheduleName, ok := snapshot.Labels[scheduleNameLabel]
	if !ok {
		return
	}

	// Enqueue the schedule for reconciliation
	key := fmt.Sprintf("%s/%s", snapshot.Namespace, scheduleName)
	log.Log.V(3).Infof("Snapshot %s changed, enqueueing schedule %s", snapshot.Name, key)
	ctrl.scheduleQueue.Add(key)
}

// EnqueueAll enqueues all schedules for reconciliation
func (ctrl *VMSnapshotScheduleController) EnqueueAll() {
	objs := ctrl.VMSnapshotScheduleInformer.GetStore().List()
	for _, obj := range objs {
		schedule, ok := obj.(*snapshotv1.VirtualMachineSnapshotSchedule)
		if !ok {
			continue
		}
		key, err := cache.MetaNamespaceKeyFunc(schedule)
		if err != nil {
			continue
		}
		ctrl.scheduleQueue.Add(key)
	}
}
