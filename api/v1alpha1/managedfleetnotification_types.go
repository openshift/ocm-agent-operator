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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FleetNotification defines the desired spec of ManagedFleetNotification
type FleetNotification struct {
	// The name of the notification used to associate with an alert
	Name string `json:"name"`

	// The summary line of the notification
	Summary string `json:"summary"`

	// The body text of the notification when the alert is active
	NotificationMessage string `json:"notificationMessage"`

	// LogType is a categorization property that can be used to group service logs for aggregation and managing notification preferences.
	LogType string `json:"logType,omitempty"`

	// References useful for context or remediation - this could be links to documentation, KB articles, etc
	References []NotificationReferenceType `json:"references,omitempty"`

	// +kubebuilder:validation:Enum={"Debug","Info","Warning","Error","Fatal"}
	// Re-use the severity definitation in managednotification_types
	Severity NotificationSeverity `json:"severity"`

	// Measured in hours. The minimum time interval that must elapse between active notifications
	ResendWait int32 `json:"resendWait"`

	// Whether or not limited support should be sent for this notification
	LimitedSupport bool `json:"limitedSupport"`
}

type ManagedFleetNotificationSpec struct {
	FleetNotification FleetNotification `json:"fleetNotification"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=mfn

// ManagedFleetNotification is the Schema for the managedfleetnotifications API
type ManagedFleetNotification struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ManagedFleetNotificationSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedFleetNotificationList contains a list of ManagedFleetNotification
type ManagedFleetNotificationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedFleetNotification `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedFleetNotification{}, &ManagedFleetNotificationList{})
}

// GetNotificationByName returns a notification matching the given name
// or error if no matching notification can be found.
func (fn *ManagedFleetNotification) GetNotificationByName(name string) (*ManagedFleetNotification, error) {
	if name == fn.Spec.FleetNotification.Name {
		return fn, nil
	}
	return nil, fmt.Errorf("notification with name %v not found", name)
}
