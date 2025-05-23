/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crdtypes "github.com/ovn-org/ovn-kubernetes/go-controller/pkg/crd/types"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=networkqoses
// +kubebuilder::singular=networkqos
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=".status.status"
// +kubebuilder:subresource:status
// NetworkQoS is a CRD that allows the user to define a DSCP marking and metering
// for pods ingress/egress traffic on its namespace to specified CIDRs,
// protocol and port. Traffic belong these pods will be checked against
// each Rule in the namespace's NetworkQoS, and if there is a match the traffic
// is marked with relevant DSCP value and enforcing specified policing
// parameters.
type NetworkQoS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Spec   `json:"spec,omitempty"`
	Status Status `json:"status,omitempty"`
}

// Spec defines the desired state of NetworkQoS
type Spec struct {
	// networkSelector selects the networks on which the pod IPs need to be added to the source address set.
	// NetworkQoS controller currently supports `NetworkAttachmentDefinitions` type only.
	// +optional
	// +kubebuilder:validation:XValidation:rule="self == oldSelf", message="networkSelector is immutable"
	// +kubebuilder:validation:XValidation:rule="self.all(sel, sel.networkSelectionType == 'ClusterUserDefinedNetworks' || sel.networkSelectionType == 'NetworkAttachmentDefinitions')", message="Unsupported network selection type"
	NetworkSelectors crdtypes.NetworkSelectors `json:"networkSelectors,omitempty"`

	// podSelector applies the NetworkQoS rule only to the pods in the namespace whose label
	// matches this definition. This field is optional, and in case it is not set
	// results in the rule being applied to all pods in the namespace.
	// +optional
	PodSelector metav1.LabelSelector `json:"podSelector,omitempty"`

	// priority is a value from 0 to 100 and represents the NetworkQoS' priority.
	// QoSes with numerically higher priority takes precedence over those with lower.
	// +kubebuilder:validation:Maximum:=100
	// +kubebuilder:validation:Minimum:=0
	Priority int `json:"priority"`

	// egress a collection of Egress NetworkQoS rule objects. A total of 20 rules will
	// be allowed in each NetworkQoS instance. The relative precedence of egress rules
	// within a single NetworkQos object (all of which share the priority) will be
	// determined by the order in which the rule is written. Thus, a rule that appears
	// first in the list of egress rules would take the lower precedence.
	// +kubebuilder:validation:MaxItems=20
	Egress []Rule `json:"egress"`
}

type Rule struct {
	// dscp marking value for matching pods' traffic.
	// +kubebuilder:validation:Maximum:=63
	// +kubebuilder:validation:Minimum:=0
	DSCP int `json:"dscp"`

	// classifier The classifier on which packets should match
	// to apply the NetworkQoS Rule.
	// This field is optional, and in case it is not set the rule is applied
	// to all egress traffic regardless of the destination.
	// +optional
	Classifier Classifier `json:"classifier"`

	// +optional
	Bandwidth Bandwidth `json:"bandwidth"`
}

type Classifier struct {
	// +optional
	To []Destination `json:"to"`

	// +optional
	Ports []*Port `json:"ports"`
}

// Bandwidth controls the maximum of rate traffic that can be sent
// or received on the matching packets.
type Bandwidth struct {
	// rate The value of rate limit in kbps. Traffic over the limit
	// will be dropped.
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=4294967295
	// +optional
	Rate uint32 `json:"rate"`

	// burst The value of burst rate limit in kilobits.
	// This also needs rate to be specified.
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=4294967295
	// +optional
	Burst uint32 `json:"burst"`
}

// Port specifies destination protocol and port on which NetworkQoS
// rule is applied
type Port struct {
	// protocol (tcp, udp, sctp) that the traffic must match.
	// +kubebuilder:validation:Pattern=^TCP|UDP|SCTP$
	// +optional
	Protocol string `json:"protocol"`

	// port that the traffic must match
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=65535
	// +optional
	Port *int32 `json:"port"`
}

// Destination describes a peer to apply NetworkQoS configuration for the outgoing traffic.
// Only certain combinations of fields are allowed.
// +kubebuilder:validation:XValidation:rule="!(has(self.ipBlock) && (has(self.podSelector) || has(self.namespaceSelector)))",message="Can't specify both podSelector/namespaceSelector and ipBlock"
type Destination struct {
	// podSelector is a label selector which selects pods. This field follows standard label
	// selector semantics; if present but empty, it selects all pods.
	//
	// If namespaceSelector is also set, then the NetworkQoS as a whole selects
	// the pods matching podSelector in the Namespaces selected by NamespaceSelector.
	// Otherwise it selects the pods matching podSelector in the NetworkQoS's own namespace.
	// +optional
	PodSelector *metav1.LabelSelector `json:"podSelector,omitempty" protobuf:"bytes,1,opt,name=podSelector"`

	// namespaceSelector selects namespaces using cluster-scoped labels. This field follows
	// standard label selector semantics; if present but empty, it selects all namespaces.
	//
	// If podSelector is also set, then the NetworkQoS as a whole selects
	// the pods matching podSelector in the namespaces selected by namespaceSelector.
	// Otherwise it selects all pods in the namespaces selected by namespaceSelector.
	// +optional
	NamespaceSelector *metav1.LabelSelector `json:"namespaceSelector,omitempty" protobuf:"bytes,2,opt,name=namespaceSelector"`

	// ipBlock defines policy on a particular IPBlock. If this field is set then
	// neither of the other fields can be.
	// +optional
	IPBlock *networkingv1.IPBlock `json:"ipBlock,omitempty" protobuf:"bytes,3,rep,name=ipBlock"`
}

// Status defines the observed state of NetworkQoS
type Status struct {
	// A concise indication of whether the NetworkQoS resource is applied with success.
	// +optional
	Status string `json:"status,omitempty"`

	// An array of condition objects indicating details about status of NetworkQoS object.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:path=networkqoses
// +kubebuilder::singular=networkqos
// NetworkQoSList contains a list of NetworkQoS
type NetworkQoSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NetworkQoS `json:"items"`
}
