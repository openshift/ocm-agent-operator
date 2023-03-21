/*
Copyright 2022.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AgentConfig struct {
	// OcmBaseUrl defines the OCM api endpoint for OCM agent to access
	OcmBaseUrl string `json:"ocmBaseUrl"`

	// Services defines the supported OCM services, eg, service_log, cluster_management
	Services []string `json:"services"`
}

// OcmAgentSpec defines the desired state of OcmAgent
type OcmAgentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// AgentConfig refers to OCM agent config fields separated
	AgentConfig AgentConfig `json:"agentConfig"`

	// OcmAgentImage defines the image which will be used by the OCM Agent
	OcmAgentImage string `json:"ocmAgentImage"`

	// TokenSecret points to the secret name which stores the access token to OCM server
	TokenSecret string `json:"tokenSecret"`

	// Replicas defines the replica count for the OCM Agent service
	Replicas int32 `json:"replicas"`

	// FleetMode indicates if the OCM agent is running in fleet mode, default to false
	FleetMode bool `json:"fleetMode,omitempty"`
}

// OcmAgentStatus defines the observed state of OcmAgent
type OcmAgentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ServiceStatus indicates the status of OCM Agent service
	ServiceStatus string `json:"serviceStatus"`

	AvailableReplicas int32 `json:"availableReplicas"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=ocmagents,scope=Namespaced

// OcmAgent is the Schema for the ocmagents API
type OcmAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OcmAgentSpec   `json:"spec,omitempty"`
	Status OcmAgentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OcmAgentList contains a list of OcmAgent
type OcmAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OcmAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OcmAgent{}, &OcmAgentList{})
}
