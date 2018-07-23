package v1alpha1

import (
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd/api"
)

// StorkRuleActionType is a type for actions that are supported in a stork rule
type StorkRuleActionType string

const (
	// StorkRuleActionCommand is a command action
	StorkRuleActionCommand StorkRuleActionType = "command"
	// StorkClusterPairResourcePlural is plural for "clusterpair" resource
	StorkClusterPairResourcePlural = "clusterpairs"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorkRule denotes an object to declare a rule that performs actions on pods
type StorkRule struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata,omitempty"`
	Spec            []StorkRuleItem `json:"spec"`
}

// StorkRuleItem represents one items in a stork rule spec
type StorkRuleItem struct {
	// PodSelector is a map of key value pairs that are used to select the pods using their labels
	PodSelector map[string]string `json:"podSelector"`
	// Actions are actions to be performed on the pods selected using the selector
	Actions []StorkRuleAction `json:"actions"`
}

// StorkRuleAction represents an action in a stork rule item
type StorkRuleAction struct {
	// Type is a type of the stork rule action
	Type StorkRuleActionType `json:"type"`
	// Background indicates that the action needs to be performed in the background
	// +optional
	Background bool `json:"background,omitempty"`
	// RunInSinglePod indicates that the action needs to be performed in a single pod
	//                from the list of pods that match the selector
	// +optional
	RunInSinglePod bool `json:"runInSinglePod,omitempty"`
	// Value is the actual action value for e.g the command to run
	Value string `json:"value"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorkRuleList is a list of stork rules
type StorkRuleList struct {
	meta.TypeMeta `json:",inline"`
	meta.ListMeta `json:"metadata,omitempty"`

	Items []StorkRule `json:"items"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPair represents pairing with other clusters
type ClusterPair struct {
	meta.TypeMeta   `json:",inline"`
	meta.ObjectMeta `json:"metadata,omitempty"`
	Config          api.Config        `json:"config"`
	Options         map[string]string `json:"options"`
	Status          ClusterPairStatus `json:"status"`
	RemoteStorageID string            `json:"remoteStorageId"`
}

// ClusterPairStatusType is the status of the pair
type ClusterPairStatusType string

const (
	// ClusterPairStatusPending for when pairing is still pending
	ClusterPairStatusPending ClusterPairStatusType = "Pending"
	// ClusterPairStatusReady for when pair is ready
	ClusterPairStatusReady ClusterPairStatusType = "Ready"
	// ClusterPairStatusError for when pairing is in error state
	ClusterPairStatusError ClusterPairStatusType = "Error"
	// ClusterPairStatusDegraded for when pairing is degraded
	ClusterPairStatusDegraded ClusterPairStatusType = "Degraded"
	// ClusterPairStatusDeleting for when pairing is being deleted
	ClusterPairStatusDeleting ClusterPairStatusType = "Deleting"
)

// ClusterPairStatus is the status of the cluster pair
type ClusterPairStatus struct {
	// Overall status of the pairing
	// +optional
	OverallStatus ClusterPairStatusType `json:"overallStatus"`
	// Status of the pairing with the scheduler
	// +optional
	SchedulerStatus ClusterPairStatusType `json:"schedulerStatus"`
	// Status of pairing with the storage driver
	// +optional
	StorageStatus ClusterPairStatusType `json:"storageStatus"`
}

// ClusterPairSpec is the desired state of the cluster pair
type ClusterPairSpec struct {
	Config api.Config `json:"config"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterPairList is a list of cluster pairs
type ClusterPairList struct {
	meta.TypeMeta `json:",inline"`
	meta.ListMeta `json:"metadata,omitempty"`

	Items []ClusterPair `json:"items"`
}
