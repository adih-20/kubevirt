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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package v1beta1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
)

const DefaultFailureDeadline = 5 * time.Minute
const DefaultGracePeriod = 5 * time.Minute

// VirtualMachineSnapshot defines the operation of snapshotting a VM
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineSnapshotSpec `json:"spec"`

	// +optional
	Status *VirtualMachineSnapshotStatus `json:"status,omitempty"`
}

// DeletionPolicy defines that to do with VirtualMachineSnapshot
// when VirtualMachineSnapshot is deleted
type DeletionPolicy string

const (
	// VirtualMachineSnapshotContentDelete causes the
	// VirtualMachineSnapshotContent to be deleted
	VirtualMachineSnapshotContentDelete DeletionPolicy = "Delete"

	// VirtualMachineSnapshotContentRetain causes the
	// VirtualMachineSnapshotContent to stay around
	VirtualMachineSnapshotContentRetain DeletionPolicy = "Retain"
)

// VirtualMachineSnapshotSpec is the spec for a VirtualMachineSnapshot resource
type VirtualMachineSnapshotSpec struct {
	Source corev1.TypedLocalObjectReference `json:"source"`

	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// This time represents the number of seconds we permit the vm snapshot
	// to take. In case we pass this deadline we mark this snapshot
	// as failed.
	// Defaults to DefaultFailureDeadline - 5min
	// +optional
	FailureDeadline *metav1.Duration `json:"failureDeadline,omitempty"`
}

// Indication is a way to indicate the state of the vm when taking the snapshot
type Indication string

const (
	VMSnapshotOnlineSnapshotIndication Indication = "Online"
	VMSnapshotNoGuestAgentIndication   Indication = "NoGuestAgent"
	VMSnapshotGuestAgentIndication     Indication = "GuestAgent"
	VMSnapshotQuiesceFailedIndication  Indication = "QuiesceFailed"
	VMSnapshotPausedIndication         Indication = "Paused"
)

// SourceIndication provides an indication of the source VM with its description message
type SourceIndication struct {
	// Indication is the indication type
	Indication Indication `json:"indication"`
	// Message provides a description message of the indication
	Message string `json:"message"`
}

// VirtualMachineSnapshotPhase is the current phase of the VirtualMachineSnapshot
type VirtualMachineSnapshotPhase string

const (
	PhaseUnset VirtualMachineSnapshotPhase = ""
	InProgress VirtualMachineSnapshotPhase = "InProgress"
	Succeeded  VirtualMachineSnapshotPhase = "Succeeded"
	Failed     VirtualMachineSnapshotPhase = "Failed"
	Deleting   VirtualMachineSnapshotPhase = "Deleting"
	Unknown    VirtualMachineSnapshotPhase = "Unknown"
)

// VirtualMachineSnapshotStatus is the status for a VirtualMachineSnapshot resource
type VirtualMachineSnapshotStatus struct {
	// +optional
	SourceUID *types.UID `json:"sourceUID,omitempty"`

	// +optional
	VirtualMachineSnapshotContentName *string `json:"virtualMachineSnapshotContentName,omitempty"`

	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	Phase VirtualMachineSnapshotPhase `json:"phase,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *Error `json:"error,omitempty"`

	// +optional
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`

	// Deprecated: Use SourceIndications instead. This field will be removed in a future version.
	// +optional
	// +listType=set
	Indications []Indication `json:"indications,omitempty"`

	// +optional
	// +listType=atomic
	SourceIndications []SourceIndication `json:"sourceIndications,omitempty"`

	// +optional
	SnapshotVolumes *SnapshotVolumesLists `json:"snapshotVolumes,omitempty"`
}

// SnapshotVolumesLists includes the list of volumes which were included in the snapshot and volumes which were excluded from the snapshot
type SnapshotVolumesLists struct {
	// +optional
	// +listType=set
	IncludedVolumes []string `json:"includedVolumes,omitempty"`

	// +optional
	// +listType=set
	ExcludedVolumes []string `json:"excludedVolumes,omitempty"`
}

// Error is the last error encountered during the snapshot/restore
type Error struct {
	// +optional
	Time *metav1.Time `json:"time,omitempty"`

	// +optional
	Message *string `json:"message,omitempty"`
}

// ConditionType is the const type for Conditions
type ConditionType string

const (
	// ConditionReady is the "ready" condition type
	ConditionReady ConditionType = "Ready"

	// ConditionProgressing is the "progressing" condition type
	ConditionProgressing ConditionType = "Progressing"

	// ConditionFailure is the "failure" condition type
	ConditionFailure ConditionType = "Failure"
)

// Condition defines conditions
type Condition struct {
	Type ConditionType `json:"type"`

	Status corev1.ConditionStatus `json:"status"`

	// +optional
	// +nullable
	LastProbeTime metav1.Time `json:"lastProbeTime,omitempty"`

	// +optional
	// +nullable
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// +optional
	Reason string `json:"reason,omitempty"`

	// +optional
	Message string `json:"message,omitempty"`
}

// VirtualMachineSnapshotList is a list of VirtualMachineSnapshot resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualMachineSnapshot `json:"items"`
}

// VirtualMachineSnapshotContent contains the snapshot data
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshotContent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineSnapshotContentSpec `json:"spec"`

	// +optional
	Status *VirtualMachineSnapshotContentStatus `json:"status,omitempty"`
}

// VirtualMachineSnapshotContentSpec is the spec for a VirtualMachineSnapshotContent resource
type VirtualMachineSnapshotContentSpec struct {
	VirtualMachineSnapshotName *string `json:"virtualMachineSnapshotName,omitempty"`

	Source SourceSpec `json:"source"`

	// +optional
	// +listType=atomic
	VolumeBackups []VolumeBackup `json:"volumeBackups,omitempty"`
}

type VirtualMachine struct {
	// +kubebuilder:pruning:PreserveUnknownFields
	// +nullable
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// VirtualMachineSpec contains the VirtualMachine specification.
	Spec v1.VirtualMachineSpec `json:"spec,omitempty" valid:"required"`
	// Status holds the current state of the controller and brief information
	// about its associated VirtualMachineInstance
	Status v1.VirtualMachineStatus `json:"status,omitempty"`
}

// SourceSpec contains the appropriate spec for the resource being snapshotted
type SourceSpec struct {
	// +optional
	VirtualMachine *VirtualMachine `json:"virtualMachine,omitempty"`
}

type PersistentVolumeClaim struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired characteristics of a volume requested by a pod author.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	Spec corev1.PersistentVolumeClaimSpec `json:"spec,omitempty"`
}

// VolumeBackup contains the data neeed to restore a PVC
type VolumeBackup struct {
	VolumeName string `json:"volumeName"`

	PersistentVolumeClaim PersistentVolumeClaim `json:"persistentVolumeClaim"`

	// +optional
	VolumeSnapshotName *string `json:"volumeSnapshotName,omitempty"`
}

// VirtualMachineSnapshotContentStatus is the status for a VirtualMachineSnapshotStatus resource
type VirtualMachineSnapshotContentStatus struct {
	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *Error `json:"error,omitempty"`

	// +optional
	// +listType=atomic
	VolumeSnapshotStatus []VolumeSnapshotStatus `json:"volumeSnapshotStatus,omitempty"`
}

// VirtualMachineSnapshotContentList is a list of VirtualMachineSnapshot resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshotContentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualMachineSnapshotContent `json:"items"`
}

// VolumeSnapshotStatus is the status of a VolumeSnapshot
type VolumeSnapshotStatus struct {
	VolumeSnapshotName string `json:"volumeSnapshotName"`

	// +optional
	// +nullable
	CreationTime *metav1.Time `json:"creationTime,omitempty"`

	// +optional
	ReadyToUse *bool `json:"readyToUse,omitempty"`

	// +optional
	Error *Error `json:"error,omitempty"`
}

// VirtualMachineRestore defines the operation of restoring a VM
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineRestore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineRestoreSpec `json:"spec"`

	// +optional
	Status *VirtualMachineRestoreStatus `json:"status,omitempty"`
}

// TargetReadinessPolicy defines how to handle the restore in case
// the target is not ready
type TargetReadinessPolicy string

const (
	// VirtualMachineRestoreStopTarget defined TargetReadinessPolicy which stops the target so the
	// VirtualMachineRestore can continue immediatly
	VirtualMachineRestoreStopTarget TargetReadinessPolicy = "StopTarget"

	// VirtualMachineRestoreWaitGracePeriodAndFail defines TargetReadinessPolicy which lets the
	// user `DefaultGracePeriod` time to get the target ready.
	// If not ready in that time the restore will fail
	VirtualMachineRestoreWaitGracePeriodAndFail TargetReadinessPolicy = "WaitGracePeriod"

	//VirtualMachineRestoreFailImmediate defines TargetReadinessPolicy which if VirtualMachineRestore
	// was initiated when target is not ready it fails the restore immediately
	VirtualMachineRestoreFailImmediate TargetReadinessPolicy = "FailImmediate"

	// VirtualMachineRestoreWaitEventually defines TargetReadinessPolicy which keeps the
	// VirtualMachineRestore around and once the target is ready the restore will
	// occur. No timeout for the operation
	VirtualMachineRestoreWaitEventually TargetReadinessPolicy = "WaitEventually"
)

// VolumeRestorePolicy defines how to handle the restore of snapshotted volumes
type VolumeRestorePolicy string

const (
	// VolumeRestorePolicyRandomizeNames defines a VolumeRestorePolicy which creates
	// new PVCs with randomized names for each snapshotted volume. This is the default policy.
	VolumeRestorePolicyRandomizeNames VolumeRestorePolicy = "RandomizeNames"

	// VolumeRestorePolicyInPlace defines a VolumeRestorePolicy which overwrites
	// existing PVCs for each snapshotted volumes. That means deleting the original PVC if it still
	// exists, and restoring the volume with the same name as the original PVC.
	VolumeRestorePolicyInPlace VolumeRestorePolicy = "InPlace"
)

// VolumeOwnershipPolicy defines what owns volumes once they're restored
type VolumeOwnershipPolicy string

const (
	// VolumeOwnershipPolicyVm defines a VolumeOwnershipPolicyVm where restored volumes are owned by the restored VM
	VolumeOwnershipPolicyVm VolumeOwnershipPolicy = "Vm"

	// VolumeOwnershipPolicyNone defines a VolumeOwnershipPolicyVm where restored volumes are not owned by any entity
	VolumeOwnershipPolicyNone VolumeOwnershipPolicy = "None"
)

// VirtualMachineRestoreSpec is the spec for a VirtualMachineRestore resource
type VirtualMachineRestoreSpec struct {
	// initially only VirtualMachine type supported
	Target corev1.TypedLocalObjectReference `json:"target"`

	VirtualMachineSnapshotName string `json:"virtualMachineSnapshotName"`

	// +optional
	TargetReadinessPolicy *TargetReadinessPolicy `json:"targetReadinessPolicy,omitempty"`

	// +optional
	VolumeRestorePolicy *VolumeRestorePolicy `json:"volumeRestorePolicy,omitempty"`

	// +optional
	VolumeOwnershipPolicy *VolumeOwnershipPolicy `json:"volumeOwnershipPolicy,omitempty"`

	// VolumeRestoreOverrides gives the option to change properties of each restored volume
	// For example, specifying the name of the restored volume, or adding labels/annotations to it
	// +optional
	// +listType=atomic
	VolumeRestoreOverrides []VolumeRestoreOverride `json:"volumeRestoreOverrides,omitempty"`

	// If the target for the restore does not exist, it will be created. Patches holds JSON patches that would be
	// applied to the target manifest before it's created. Patches should fit the target's Kind.
	//
	// Example for a patch: {"op": "replace", "path": "/metadata/name", "value": "new-vm-name"}
	//
	// +optional
	// +listType=atomic
	Patches []string `json:"patches,omitempty"`
}

// VirtualMachineRestoreStatus is the status for a VirtualMachineRestore resource
type VirtualMachineRestoreStatus struct {
	// +optional
	// +listType=atomic
	Restores []VolumeRestore `json:"restores,omitempty"`

	// +optional
	RestoreTime *metav1.Time `json:"restoreTime,omitempty"`

	// +optional
	// +listType=set
	DeletedDataVolumes []string `json:"deletedDataVolumes,omitempty"`

	// +optional
	Complete *bool `json:"complete,omitempty"`

	// +optional
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`
}

// VolumeRestore contains the data needed to restore a PVC
type VolumeRestore struct {
	VolumeName string `json:"volumeName"`

	PersistentVolumeClaimName string `json:"persistentVolumeClaim"`

	VolumeSnapshotName string `json:"volumeSnapshotName"`

	// +optional
	DataVolumeName *string `json:"dataVolumeName,omitempty"`
}

// VolumeRestoreOverride specifies how a volume should be restored from a VirtualMachineSnapshot
type VolumeRestoreOverride struct {
	VolumeName string `json:"volumeName,omitempty"`
	// +optional
	RestoreName string `json:"restoreName,omitempty"`
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// VirtualMachineRestoreList is a list of VirtualMachineRestore resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineRestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualMachineRestore `json:"items"`
}

// VirtualMachineSnapshotSchedule defines a schedule for taking snapshots of VirtualMachines
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshotSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec VirtualMachineSnapshotScheduleSpec `json:"spec"`

	// +optional
	Status *VirtualMachineSnapshotScheduleStatus `json:"status,omitempty"`
}

// VirtualMachineSnapshotScheduleSpec is the spec for a VirtualMachineSnapshotSchedule resource
type VirtualMachineSnapshotScheduleSpec struct {
	// Source is the TypedLocalObjectReference of the VirtualMachine to snapshot.
	// If specified, takes precedence over VMSelector.
	// +optional
	Source *corev1.TypedLocalObjectReference `json:"source,omitempty"`

	// VMSelector is a LabelSelector to select VirtualMachines to snapshot.
	// Multiple VMs can be selected using this field.
	// +optional
	VMSelector *metav1.LabelSelector `json:"vmSelector,omitempty"`

	// Schedule defines the cron expression for when snapshots should be taken.
	// The schedule is interpreted with respect to the UTC timezone.
	// Supports standard cron expressions and pre-defined shortcuts:
	// @hourly, @daily, @weekly, @monthly, @yearly
	Schedule string `json:"schedule"`

	// Disabled when set to true makes the schedule inactive
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// Retention defines the policy for retaining snapshots
	// +optional
	Retention *VirtualMachineSnapshotScheduleRetention `json:"retention,omitempty"`

	// SnapshotTemplate contains settings for the VirtualMachineSnapshots that are created
	// +optional
	SnapshotTemplate *VirtualMachineSnapshotTemplateSpec `json:"snapshotTemplate,omitempty"`

	// FailurePolicy defines how to handle snapshot failures
	// +optional
	FailurePolicy *ScheduleFailurePolicy `json:"failurePolicy,omitempty"`
}

// VirtualMachineSnapshotScheduleRetention defines the retention policy for scheduled snapshots
type VirtualMachineSnapshotScheduleRetention struct {
	// Expires is the length of time a snapshot should be retained.
	// Format is a duration string (e.g., "168h" for 1 week, "720h" for 30 days).
	// When both Expires and MaxCount are set, snapshots are deleted when either
	// condition is met.
	// +optional
	Expires *metav1.Duration `json:"expires,omitempty"`

	// MaxCount is the maximum number of snapshots to retain per VirtualMachine.
	// When this limit is reached, the oldest snapshot will be deleted.
	// When both Expires and MaxCount are set, snapshots are deleted when either
	// condition is met.
	// +optional
	MaxCount *int32 `json:"maxCount,omitempty"`
}

// VirtualMachineSnapshotTemplateSpec contains settings for snapshots created by the schedule
type VirtualMachineSnapshotTemplateSpec struct {
	// Labels to add to each VirtualMachineSnapshot
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to add to each VirtualMachineSnapshot
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// DeletionPolicy defines what to do with the VirtualMachineSnapshotContent
	// when the VirtualMachineSnapshot is deleted
	// +optional
	DeletionPolicy *DeletionPolicy `json:"deletionPolicy,omitempty"`

	// FailureDeadline is the time limit for a snapshot to complete.
	// If not specified, defaults to 5 minutes.
	// +optional
	FailureDeadline *metav1.Duration `json:"failureDeadline,omitempty"`
}

// ScheduleFailurePolicy defines how to handle snapshot failures
type ScheduleFailurePolicy string

const (
	// ScheduleFailurePolicyContinue continues with the next scheduled snapshot
	// even if the previous one failed
	ScheduleFailurePolicyContinue ScheduleFailurePolicy = "Continue"

	// ScheduleFailurePolicyPause pauses the schedule when a snapshot fails
	// until the user manually resumes it
	ScheduleFailurePolicyPause ScheduleFailurePolicy = "Pause"
)

// VirtualMachineSnapshotSchedulePhase is the current phase of the schedule
type VirtualMachineSnapshotSchedulePhase string

const (
	// SchedulePhaseActive indicates the schedule is active and taking snapshots
	SchedulePhaseActive VirtualMachineSnapshotSchedulePhase = "Active"

	// SchedulePhasePaused indicates the schedule is paused (disabled or due to failure)
	SchedulePhasePaused VirtualMachineSnapshotSchedulePhase = "Paused"

	// SchedulePhaseFailed indicates the schedule has encountered an error
	SchedulePhaseFailed VirtualMachineSnapshotSchedulePhase = "Failed"
)

// VirtualMachineSnapshotScheduleStatus is the status for a VirtualMachineSnapshotSchedule
type VirtualMachineSnapshotScheduleStatus struct {
	// Phase is the current phase of the schedule
	// +optional
	Phase VirtualMachineSnapshotSchedulePhase `json:"phase,omitempty"`

	// LastSnapshotTime is the time when the last snapshot was taken
	// +optional
	// +nullable
	LastSnapshotTime *metav1.Time `json:"lastSnapshotTime,omitempty"`

	// NextSnapshotTime is the time when the next snapshot is scheduled
	// +optional
	// +nullable
	NextSnapshotTime *metav1.Time `json:"nextSnapshotTime,omitempty"`

	// LastSuccessfulSnapshotName is the name of the last successfully completed snapshot
	// +optional
	LastSuccessfulSnapshotName string `json:"lastSuccessfulSnapshotName,omitempty"`

	// CurrentSnapshotCount is the current number of snapshots managed by this schedule
	// per each VM when using VMSelector, or total when using Source
	// +optional
	CurrentSnapshotCount int32 `json:"currentSnapshotCount,omitempty"`

	// Error contains the last error encountered by the controller
	// +optional
	Error *Error `json:"error,omitempty"`

	// Conditions represent the latest available observations of the schedule's state
	// +optional
	// +listType=atomic
	Conditions []Condition `json:"conditions,omitempty"`

	// VMSnapshotStatuses contains the status of snapshots per VM when using VMSelector
	// +optional
	// +listType=atomic
	VMSnapshotStatuses []VMSnapshotStatus `json:"vmSnapshotStatuses,omitempty"`
}

// VMSnapshotStatus contains snapshot status for a specific VM
type VMSnapshotStatus struct {
	// VMName is the name of the VirtualMachine
	VMName string `json:"vmName"`

	// LastSnapshotName is the name of the last snapshot taken for this VM
	// +optional
	LastSnapshotName string `json:"lastSnapshotName,omitempty"`

	// LastSnapshotTime is when the last snapshot was taken for this VM
	// +optional
	// +nullable
	LastSnapshotTime *metav1.Time `json:"lastSnapshotTime,omitempty"`

	// CurrentSnapshotCount is the current number of snapshots for this VM
	// +optional
	CurrentSnapshotCount int32 `json:"currentSnapshotCount,omitempty"`

	// Error contains any error for this VM's snapshots
	// +optional
	Error *Error `json:"error,omitempty"`
}

// VirtualMachineSnapshotScheduleList is a list of VirtualMachineSnapshotSchedule resources
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type VirtualMachineSnapshotScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []VirtualMachineSnapshotSchedule `json:"items"`
}
