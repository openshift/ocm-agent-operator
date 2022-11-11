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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NotificationSeverity string

const (
	SeverityDebug   NotificationSeverity = "Debug"
	SeverityWarning NotificationSeverity = "Warning"
	SeverityInfo    NotificationSeverity = "Info"
	SeverityError   NotificationSeverity = "Error"
	SeverityFatal   NotificationSeverity = "Fatal"
)

type Notification struct {

	// The name of the notification used to associate with an alert
	Name string `json:"name"`

	// The summary line of the Service Log notification
	Summary string `json:"summary"`

	// The body text of the Service Log notification when the alert is active
	ActiveDesc string `json:"activeBody"`

	// The body text of the Service Log notification when the alert is resolved
	ResolvedDesc string `json:"resolvedBody,omitempty"`

	// +kubebuilder:validation:Enum={"Debug","Info","Warning","Error","Fatal"}
	// The severity of the Service Log notification
	Severity NotificationSeverity `json:"severity"`

	// Measured in hours. The minimum time interval that must elapse between active Service Log notifications
	ResendWait int32 `json:"resendWait"`
}

// ManagedNotificationSpec defines the desired state of ManagedNotification
type ManagedNotificationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// AgentConfig refers to OCM agent config fields separated
	Notifications []Notification `json:"notifications"`
}

// ManagedNotificationStatus defines the observed state of ManagedNotification
type ManagedNotificationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	NotificationRecords NotificationRecords `json:"notificationRecords,omitempty"`
}

type NotificationRecords []NotificationRecord

type NotificationConditionType string

const (
	ConditionAlertFiring    NotificationConditionType = "AlertFiring"
	ConditionAlertResolved  NotificationConditionType = "AlertResolved"
	ConditionServiceLogSent NotificationConditionType = "ServiceLogSent"
)

type Conditions []NotificationCondition
type NotificationCondition struct {
	// +kubebuilder:validation:Enum={"AlertFiring","AlertResolved","ServiceLogSent"}
	// Type of Notification condition
	Type NotificationConditionType `json:"type"`

	// Status of condition, one of True, False, Unknown
	Status corev1.ConditionStatus `json:"status"`

	// Last time the condition transit from one status to another.
	// +kubebuilder:validation:Optional
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`

	// (brief) reason for the condition's last transition.
	// +kubebuilder:validation:Optional
	Reason string `json:"reason,omitempty"`
}

type NotificationRecord struct {
	// Name of the notification
	Name string `json:"name"`

	// +kubebuilder:validation:Optional
	// ServiceLogSentCount records the number of service logs sent for the notification
	ServiceLogSentCount int32 `json:"serviceLogSentCount,omitempty"`

	// Conditions is a set of Condition instances.
	Conditions Conditions `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=managednotifications,scope=Namespaced

// ManagedNotification is the Schema for the managednotifications API
type ManagedNotification struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedNotificationSpec   `json:"spec,omitempty"`
	Status ManagedNotificationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedNotificationList contains a list of ManagedNotification
type ManagedNotificationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedNotification `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedNotification{}, &ManagedNotificationList{})
}

// GetNotificationForName returns a notification matching the given name
// or error if no matching notification can be found.
func (m *ManagedNotification) GetNotificationForName(n string) (*Notification, error) {
	for _, t := range m.Spec.Notifications {
		if t.Name == n {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("notification with name %v not found", n)
}

// GetNotificationRecord returns the history for a notification matching
// the given name or error if no matching notification can be found.
func (m *ManagedNotificationStatus) GetNotificationRecord(n string) (*NotificationRecord, error) {
	for _, t := range m.NotificationRecords {
		if t.Name == n {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("notification record for notification %v not found", n)
}

// HasNotificationRecord returns whether or not a notification status history exists
// with the given name
func (m *ManagedNotificationStatus) HasNotificationRecord(n string) bool {
	for _, t := range m.NotificationRecords {
		if t.Name == n {
			return true
		}
	}
	return false
}

// CanBeSent returns true if a service log from the notification is allowed to be sent
func (m *ManagedNotification) CanBeSent(n string, firing bool) (bool, error) {

	// If no notification exists, one cannot be sent
	t, err := m.GetNotificationForName(n)
	if err != nil {
		return false, err
	}

	hasNotificationRecord := m.Status.HasNotificationRecord(n)

	// If alert is firing
	if firing {
		// If no status history exists for the notification, it is safe to send a notification
		if !hasNotificationRecord {
			return true, nil
		}

		// If a status history exists but can't be fetched, this is an irregular situation
		s, err := m.Status.GetNotificationRecord(n)
		if err != nil {
			return false, err
		}
		// Check if the last time a notification was sent is within the don't-resend window, don't send
		sentCondition := s.Conditions.GetCondition(ConditionServiceLogSent)
		if sentCondition == nil {
			// No service log send recorded yet, it can be sent
			return true, nil
		}
		now := time.Now()
		nextresend := sentCondition.LastTransitionTime.Time.Add(time.Duration(t.ResendWait) * time.Hour)
		if now.Before(nextresend) {
			return false, nil
		}
	} else {
		// If not status history, we should not send the resolved notification
		if !hasNotificationRecord {
			return false, nil
		}

		// If a status history exists but can't be fetched, this is an irregular situation
		s, err := m.Status.GetNotificationRecord(n)
		if err != nil {
			return false, err
		}

		// If resolved body is empty, do not send SL for resolved alert
		if len(t.ResolvedDesc) == 0 {
			return false, nil
		}

		// If alert is not firing, only firing status notification can be sent
		firingStatus := s.Conditions.GetCondition(ConditionAlertFiring).Status
		if firingStatus != corev1.ConditionTrue {
			return false, nil
		}
	}

	return true, nil
}

// GetCondition searches the set of conditions for the condition with the given
// ConditionType and returns it. If the matching condition is not found,
// GetCondition returns nil.
func (conditions Conditions) GetCondition(t NotificationConditionType) *NotificationCondition {
	for _, condition := range conditions {
		if condition.Type == t {
			return &condition
		}
	}
	return nil
}

// NewNotificationRecord adds a new notification record status for the given name
func (m *ManagedNotificationStatus) NewNotificationRecord(n string) {
	r := NotificationRecord{
		Name:                n,
		ServiceLogSentCount: 0,
		Conditions:          []NotificationCondition{},
	}
	m.NotificationRecords = append(m.NotificationRecords, r)
}

// GetNotificationRecord retrieves the notification record associated with the given name
func (nrs NotificationRecords) GetNotificationRecord(name string) *NotificationRecord {
	for _, n := range nrs {
		if n.Name == name {
			return &n
		}
	}
	return nil
}

// SetNotificationRecord adds or overwrites the supplied notification record
func (nrs *NotificationRecords) SetNotificationRecord(rec NotificationRecord) {
	rec.ServiceLogSentCount++
	for i, n := range *nrs {
		if n.Name == rec.Name {
			(*nrs)[i] = rec
			return
		}
	}
	*nrs = append(*nrs, rec)
}

// SetStatus updates the status for a given notification record type
func (nr *NotificationRecord) SetStatus(nct NotificationConditionType, reason string, cs corev1.ConditionStatus, t *metav1.Time) error {
	condition := NotificationCondition{
		Type:               nct,
		Status:             cs,
		LastTransitionTime: t,
		Reason:             reason,
	}
	nr.Conditions.SetCondition(condition)
	return nil
}

// SetCondition adds or updates a condition in a notification record
func (c *Conditions) SetCondition(new NotificationCondition) {
	for i, condition := range *c {
		if condition.Type == new.Type {
			(*c)[i] = new
			return
		}
	}
	*c = append(*c, new)
}
