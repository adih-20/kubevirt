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
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	kubevirtv1 "kubevirt.io/api/core/v1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
)

const (
	scheduleNameLabel      = "snapshot.kubevirt.io/schedule-name"
	scheduleNamespaceLabel = "snapshot.kubevirt.io/schedule-namespace"
	scheduledSnapshotLabel = "snapshot.kubevirt.io/scheduled"

	scheduleCreateSnapshotEvent   = "ScheduledSnapshotCreated"
	scheduleDeleteSnapshotEvent   = "ScheduledSnapshotDeleted"
	scheduleRetentionCleanupEvent = "RetentionCleanup"
	scheduleFailedEvent           = "ScheduledSnapshotFailed"
	scheduleInvalidCronEvent      = "InvalidCronExpression"
	scheduleNoVMsMatchedEvent     = "NoVMsMatchedSelector"
)

// updateVMSnapshotSchedule handles reconciliation of VirtualMachineSnapshotSchedule
func (ctrl *VMSnapshotScheduleController) updateVMSnapshotSchedule(schedule *snapshotv1.VirtualMachineSnapshotSchedule) (time.Duration, error) {
	log.Log.V(3).Infof("Processing VirtualMachineSnapshotSchedule %s/%s", schedule.Namespace, schedule.Name)

	// Initialize status if nil
	if schedule.Status == nil {
		schedule.Status = &snapshotv1.VirtualMachineSnapshotScheduleStatus{}
	}

	// Validate the cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	cronSchedule, err := parser.Parse(schedule.Spec.Schedule)
	if err != nil {
		return ctrl.updateScheduleStatusError(schedule, fmt.Errorf("invalid cron expression: %v", err))
	}

	// Check if schedule is disabled
	if schedule.Spec.Disabled {
		return ctrl.updateScheduleStatusPaused(schedule)
	}

	// Get VMs to snapshot
	vms, err := ctrl.getVMsToSnapshot(schedule)
	if err != nil {
		return ctrl.updateScheduleStatusError(schedule, err)
	}

	if len(vms) == 0 {
		ctrl.Recorder.Event(schedule, corev1.EventTypeWarning, scheduleNoVMsMatchedEvent, "No VirtualMachines matched the selector")
		return ctrl.updateScheduleStatusActive(schedule, cronSchedule)
	}

	// Check if it's time to create a snapshot
	now := time.Now().UTC()
	var nextRun time.Time

	if schedule.Status.LastSnapshotTime != nil {
		nextRun = cronSchedule.Next(schedule.Status.LastSnapshotTime.Time)
	} else {
		// First run - schedule immediately or at next cron time
		nextRun = cronSchedule.Next(now.Add(-time.Second))
	}

	if now.After(nextRun) || now.Equal(nextRun) {
		// Time to create snapshots
		if err := ctrl.createScheduledSnapshots(schedule, vms); err != nil {
			// Check failure policy
			if schedule.Spec.FailurePolicy != nil && *schedule.Spec.FailurePolicy == snapshotv1.ScheduleFailurePolicyPause {
				return ctrl.updateScheduleStatusError(schedule, err)
			}
			// Log error but continue with Continue policy
			log.Log.Warningf("Failed to create scheduled snapshot for %s/%s: %v", schedule.Namespace, schedule.Name, err)
			ctrl.Recorder.Eventf(schedule, corev1.EventTypeWarning, scheduleFailedEvent, "Failed to create snapshot: %v", err)
		}

		// Update last snapshot time
		schedule.Status.LastSnapshotTime = &metav1.Time{Time: now}
	}

	// Handle retention policy
	if err := ctrl.applyRetentionPolicy(schedule, vms); err != nil {
		log.Log.Warningf("Failed to apply retention policy for %s/%s: %v", schedule.Namespace, schedule.Name, err)
	}

	return ctrl.updateScheduleStatusActive(schedule, cronSchedule)
}

// getVMsToSnapshot returns the VMs that should be snapshotted based on the schedule spec
func (ctrl *VMSnapshotScheduleController) getVMsToSnapshot(schedule *snapshotv1.VirtualMachineSnapshotSchedule) ([]*kubevirtv1.VirtualMachine, error) {
	var vms []*kubevirtv1.VirtualMachine

	// If Source is specified, use it directly
	if schedule.Spec.Source != nil {
		if schedule.Spec.Source.Kind != "VirtualMachine" {
			return nil, fmt.Errorf("source kind must be VirtualMachine, got %s", schedule.Spec.Source.Kind)
		}

		key := fmt.Sprintf("%s/%s", schedule.Namespace, schedule.Spec.Source.Name)
		obj, exists, err := ctrl.VMInformer.GetStore().GetByKey(key)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, fmt.Errorf("VirtualMachine %s not found", key)
		}
		vm, ok := obj.(*kubevirtv1.VirtualMachine)
		if !ok {
			return nil, fmt.Errorf("unexpected object type: %T", obj)
		}
		vms = append(vms, vm)
		return vms, nil
	}

	// Use VMSelector to find matching VMs
	if schedule.Spec.VMSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(schedule.Spec.VMSelector)
		if err != nil {
			return nil, fmt.Errorf("invalid vmSelector: %v", err)
		}

		objs := ctrl.VMInformer.GetStore().List()
		for _, obj := range objs {
			vm, ok := obj.(*kubevirtv1.VirtualMachine)
			if !ok {
				continue
			}
			if vm.Namespace != schedule.Namespace {
				continue
			}
			if selector.Matches(labels.Set(vm.Labels)) {
				vms = append(vms, vm)
			}
		}
		return vms, nil
	}

	return nil, fmt.Errorf("either source or vmSelector must be specified")
}

// createScheduledSnapshots creates VirtualMachineSnapshots for the given VMs
func (ctrl *VMSnapshotScheduleController) createScheduledSnapshots(schedule *snapshotv1.VirtualMachineSnapshotSchedule, vms []*kubevirtv1.VirtualMachine) error {
	var errs []string

	for _, vm := range vms {
		if err := ctrl.createSnapshotForVM(schedule, vm); err != nil {
			errs = append(errs, fmt.Sprintf("VM %s: %v", vm.Name, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to create snapshots: %s", strings.Join(errs, "; "))
	}

	return nil
}

// createSnapshotForVM creates a VirtualMachineSnapshot for a single VM
func (ctrl *VMSnapshotScheduleController) createSnapshotForVM(schedule *snapshotv1.VirtualMachineSnapshotSchedule, vm *kubevirtv1.VirtualMachine) error {
	timestamp := time.Now().UTC().Format("20060102-150405")
	snapshotName := fmt.Sprintf("%s-%s-%s", schedule.Name, vm.Name, timestamp)

	// Build labels for the snapshot
	snapshotLabels := map[string]string{
		scheduleNameLabel:       schedule.Name,
		scheduleNamespaceLabel:  schedule.Namespace,
		scheduledSnapshotLabel:  "true",
		snapshotSourceNameLabel: vm.Name,
	}

	// Add template labels if specified
	if schedule.Spec.SnapshotTemplate != nil && schedule.Spec.SnapshotTemplate.Labels != nil {
		for k, v := range schedule.Spec.SnapshotTemplate.Labels {
			snapshotLabels[k] = v
		}
	}

	// Build annotations
	var snapshotAnnotations map[string]string
	if schedule.Spec.SnapshotTemplate != nil && schedule.Spec.SnapshotTemplate.Annotations != nil {
		snapshotAnnotations = make(map[string]string)
		for k, v := range schedule.Spec.SnapshotTemplate.Annotations {
			snapshotAnnotations[k] = v
		}
	}

	apiGroup := kubevirtv1.SchemeGroupVersion.Group
	snapshot := &snapshotv1.VirtualMachineSnapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:        snapshotName,
			Namespace:   schedule.Namespace,
			Labels:      snapshotLabels,
			Annotations: snapshotAnnotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: snapshotv1.SchemeGroupVersion.String(),
					Kind:       "VirtualMachineSnapshotSchedule",
					Name:       schedule.Name,
					UID:        schedule.UID,
					Controller: pointer.P(true),
				},
			},
		},
		Spec: snapshotv1.VirtualMachineSnapshotSpec{
			Source: corev1.TypedLocalObjectReference{
				APIGroup: &apiGroup,
				Kind:     "VirtualMachine",
				Name:     vm.Name,
			},
		},
	}

	// Apply template settings
	if schedule.Spec.SnapshotTemplate != nil {
		if schedule.Spec.SnapshotTemplate.DeletionPolicy != nil {
			snapshot.Spec.DeletionPolicy = schedule.Spec.SnapshotTemplate.DeletionPolicy
		}
		if schedule.Spec.SnapshotTemplate.FailureDeadline != nil {
			snapshot.Spec.FailureDeadline = schedule.Spec.SnapshotTemplate.FailureDeadline
		}
	}

	_, err := ctrl.Client.VirtualMachineSnapshot(schedule.Namespace).Create(context.Background(), snapshot, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			log.Log.V(3).Infof("Snapshot %s already exists", snapshotName)
			return nil
		}
		return err
	}

	ctrl.Recorder.Eventf(schedule, corev1.EventTypeNormal, scheduleCreateSnapshotEvent, "Created snapshot %s for VM %s", snapshotName, vm.Name)
	log.Log.Infof("Created scheduled snapshot %s for VM %s", snapshotName, vm.Name)

	return nil
}

// applyRetentionPolicy handles cleanup of old snapshots based on retention settings
func (ctrl *VMSnapshotScheduleController) applyRetentionPolicy(schedule *snapshotv1.VirtualMachineSnapshotSchedule, vms []*kubevirtv1.VirtualMachine) error {
	if schedule.Spec.Retention == nil {
		return nil
	}

	for _, vm := range vms {
		if err := ctrl.applyRetentionForVM(schedule, vm); err != nil {
			log.Log.Warningf("Failed to apply retention for VM %s: %v", vm.Name, err)
		}
	}

	return nil
}

// applyRetentionForVM applies retention policy for a single VM
func (ctrl *VMSnapshotScheduleController) applyRetentionForVM(schedule *snapshotv1.VirtualMachineSnapshotSchedule, vm *kubevirtv1.VirtualMachine) error {
	// Get all snapshots created by this schedule for this VM
	snapshots, err := ctrl.getScheduledSnapshotsForVM(schedule, vm)
	if err != nil {
		return err
	}

	if len(snapshots) == 0 {
		return nil
	}

	// Sort by creation time (oldest first)
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreationTimestamp.Before(&snapshots[j].CreationTimestamp)
	})

	var snapshotsToDelete []*snapshotv1.VirtualMachineSnapshot
	now := time.Now().UTC()

	// Check expiration
	if schedule.Spec.Retention.Expires != nil {
		expireDuration := schedule.Spec.Retention.Expires.Duration
		for _, snapshot := range snapshots {
			age := now.Sub(snapshot.CreationTimestamp.Time)
			if age > expireDuration {
				snapshotsToDelete = append(snapshotsToDelete, snapshot)
			}
		}
	}

	// Check max count
	if schedule.Spec.Retention.MaxCount != nil {
		maxCount := int(*schedule.Spec.Retention.MaxCount)
		// Filter out already marked for deletion
		remaining := make([]*snapshotv1.VirtualMachineSnapshot, 0)
		for _, s := range snapshots {
			found := false
			for _, d := range snapshotsToDelete {
				if s.Name == d.Name {
					found = true
					break
				}
			}
			if !found {
				remaining = append(remaining, s)
			}
		}

		// If we still have more than maxCount, delete oldest
		if len(remaining) > maxCount {
			toDelete := remaining[:len(remaining)-maxCount]
			for _, s := range toDelete {
				found := false
				for _, d := range snapshotsToDelete {
					if s.Name == d.Name {
						found = true
						break
					}
				}
				if !found {
					snapshotsToDelete = append(snapshotsToDelete, s)
				}
			}
		}
	}

	// Delete the snapshots
	for _, snapshot := range snapshotsToDelete {
		err := ctrl.Client.VirtualMachineSnapshot(snapshot.Namespace).Delete(context.Background(), snapshot.Name, metav1.DeleteOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			log.Log.Warningf("Failed to delete snapshot %s: %v", snapshot.Name, err)
			continue
		}
		ctrl.Recorder.Eventf(schedule, corev1.EventTypeNormal, scheduleDeleteSnapshotEvent, "Deleted snapshot %s due to retention policy", snapshot.Name)
		log.Log.Infof("Deleted snapshot %s due to retention policy", snapshot.Name)
	}

	return nil
}

// getScheduledSnapshotsForVM returns all snapshots created by this schedule for a specific VM
func (ctrl *VMSnapshotScheduleController) getScheduledSnapshotsForVM(schedule *snapshotv1.VirtualMachineSnapshotSchedule, vm *kubevirtv1.VirtualMachine) ([]*snapshotv1.VirtualMachineSnapshot, error) {
	var result []*snapshotv1.VirtualMachineSnapshot

	objs := ctrl.VMSnapshotInformer.GetStore().List()
	for _, obj := range objs {
		snapshot, ok := obj.(*snapshotv1.VirtualMachineSnapshot)
		if !ok {
			continue
		}

		// Check if snapshot belongs to this schedule
		if snapshot.Namespace != schedule.Namespace {
			continue
		}
		if snapshot.Labels == nil {
			continue
		}
		if snapshot.Labels[scheduleNameLabel] != schedule.Name {
			continue
		}
		if snapshot.Labels[snapshotSourceNameLabel] != vm.Name {
			continue
		}

		result = append(result, snapshot)
	}

	return result, nil
}

// updateScheduleStatusError updates the schedule status to indicate an error
func (ctrl *VMSnapshotScheduleController) updateScheduleStatusError(schedule *snapshotv1.VirtualMachineSnapshotSchedule, err error) (time.Duration, error) {
	schedule.Status.Phase = snapshotv1.SchedulePhaseFailed
	now := metav1.Now()
	errMsg := err.Error()
	schedule.Status.Error = &snapshotv1.Error{
		Time:    &now,
		Message: &errMsg,
	}

	if updateErr := ctrl.updateScheduleStatus(schedule); updateErr != nil {
		return 0, updateErr
	}

	ctrl.Recorder.Eventf(schedule, corev1.EventTypeWarning, scheduleFailedEvent, "Schedule failed: %v", err)
	return 0, err
}

// updateScheduleStatusPaused updates the schedule status to indicate it is paused
func (ctrl *VMSnapshotScheduleController) updateScheduleStatusPaused(schedule *snapshotv1.VirtualMachineSnapshotSchedule) (time.Duration, error) {
	schedule.Status.Phase = snapshotv1.SchedulePhasePaused
	schedule.Status.Error = nil

	if err := ctrl.updateScheduleStatus(schedule); err != nil {
		return 0, err
	}

	return 0, nil
}

// updateScheduleStatusActive updates the schedule status to indicate it is active
func (ctrl *VMSnapshotScheduleController) updateScheduleStatusActive(schedule *snapshotv1.VirtualMachineSnapshotSchedule, cronSchedule cron.Schedule) (time.Duration, error) {
	schedule.Status.Phase = snapshotv1.SchedulePhaseActive
	schedule.Status.Error = nil

	// Calculate next snapshot time
	var fromTime time.Time
	if schedule.Status.LastSnapshotTime != nil {
		fromTime = schedule.Status.LastSnapshotTime.Time
	} else {
		fromTime = time.Now().UTC()
	}
	nextRun := cronSchedule.Next(fromTime)
	schedule.Status.NextSnapshotTime = &metav1.Time{Time: nextRun}

	// Update snapshot count
	count, err := ctrl.countSnapshotsForSchedule(schedule)
	if err != nil {
		log.Log.Warningf("Failed to count snapshots for schedule %s/%s: %v", schedule.Namespace, schedule.Name, err)
	}
	schedule.Status.CurrentSnapshotCount = count

	if err := ctrl.updateScheduleStatus(schedule); err != nil {
		return 0, err
	}

	// Calculate requeue duration - requeue slightly after next run time
	requeueAfter := time.Until(nextRun) + time.Second
	if requeueAfter < time.Second {
		requeueAfter = time.Second
	}

	return requeueAfter, nil
}

// countSnapshotsForSchedule counts all snapshots created by this schedule
func (ctrl *VMSnapshotScheduleController) countSnapshotsForSchedule(schedule *snapshotv1.VirtualMachineSnapshotSchedule) (int32, error) {
	var count int32

	objs := ctrl.VMSnapshotInformer.GetStore().List()
	for _, obj := range objs {
		snapshot, ok := obj.(*snapshotv1.VirtualMachineSnapshot)
		if !ok {
			continue
		}

		if snapshot.Namespace != schedule.Namespace {
			continue
		}
		if snapshot.Labels == nil {
			continue
		}
		if snapshot.Labels[scheduleNameLabel] == schedule.Name {
			count++
		}
	}

	return count, nil
}

// updateScheduleStatus updates the status subresource of the schedule
func (ctrl *VMSnapshotScheduleController) updateScheduleStatus(schedule *snapshotv1.VirtualMachineSnapshotSchedule) error {
	// Get the current version from the store
	key := fmt.Sprintf("%s/%s", schedule.Namespace, schedule.Name)
	storeObj, exists, err := ctrl.VMSnapshotScheduleInformer.GetStore().GetByKey(key)
	if err != nil || !exists {
		return err
	}

	current, ok := storeObj.(*snapshotv1.VirtualMachineSnapshotSchedule)
	if !ok {
		return fmt.Errorf("unexpected object type")
	}

	// Only update if status has changed
	if statusEqual(current.Status, schedule.Status) {
		return nil
	}

	patchBytes, err := patch.GeneratePatchPayload(
		patch.PatchOperation{
			Op:    patch.PatchReplaceOp,
			Path:  "/status",
			Value: schedule.Status,
		},
	)
	if err != nil {
		return err
	}

	_, err = ctrl.Client.VirtualMachineSnapshotSchedule(schedule.Namespace).Patch(
		context.Background(),
		schedule.Name,
		"application/json-patch+json",
		patchBytes,
		metav1.PatchOptions{},
	)

	return err
}

// statusEqual compares two schedule statuses for equality
func statusEqual(a, b *snapshotv1.VirtualMachineSnapshotScheduleStatus) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	if a.Phase != b.Phase {
		return false
	}
	if a.CurrentSnapshotCount != b.CurrentSnapshotCount {
		return false
	}
	if a.LastSuccessfulSnapshotName != b.LastSuccessfulSnapshotName {
		return false
	}

	// Compare times
	if (a.LastSnapshotTime == nil) != (b.LastSnapshotTime == nil) {
		return false
	}
	if a.LastSnapshotTime != nil && !a.LastSnapshotTime.Equal(b.LastSnapshotTime) {
		return false
	}

	if (a.NextSnapshotTime == nil) != (b.NextSnapshotTime == nil) {
		return false
	}
	if a.NextSnapshotTime != nil && !a.NextSnapshotTime.Equal(b.NextSnapshotTime) {
		return false
	}

	return true
}
