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
	"time"
)

// ManagedFleetNotificationRecordStatus defines the observed state of ManagedFleetNotificationRecord
type ManagedFleetNotificationRecordStatus struct {
	// Managed Fleet Notification name
	ManagementCluster string `json:"managementCluster"`
	// An array structure to record the history for each hosted cluster
	NotificationRecordByName []NotificationRecordByName `json:"notificationRecordByName"`
}

// NotificationRecordByName groups the notification record item by notification name
type NotificationRecordByName struct {
	// Name of the notification
	NotificationName string `json:"notificationName"`
	// Resend interval for the notification
	ResendWait int32 `json:"resendWait"`
	// Notification record item with the notification name
	NotificationRecordItems []NotificationRecordItem `json:"notificationRecordItems"`
}

// NotificationRecordItem defines the basic structure of a notification record
type NotificationRecordItem struct {
	// The uuid for the hosted cluster per entry
	HostedClusterID string `json:"hostedClusterID"`

	// ServiceLogSentCount records the number of service logs sent for the notification
	ServiceLogSentCount int `json:"serviceLogSentCount"`

	// The last service log sent timestamp
	LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=mfnr

// ManagedFleetNotificationRecord is the Schema for the managedfleetnotificationrecords API
type ManagedFleetNotificationRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status ManagedFleetNotificationRecordStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedFleetNotificationRecordList contains a list of ManagedFleetNotificationRecord
type ManagedFleetNotificationRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedFleetNotificationRecord `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ManagedFleetNotificationRecord{}, &ManagedFleetNotificationRecordList{})
}

// GetNotificationRecordByMC gets the notification with the given name
func (fnr *ManagedFleetNotificationRecord) GetNotificationRecordByMC(mc string) (*ManagedFleetNotificationRecord, error) {
	if mc != fnr.Status.ManagementCluster {
		return nil, fmt.Errorf("cannot find the notificaiton with management cluster: %s", mc)
	}

	return fnr, nil
}

func (fnr *ManagedFleetNotificationRecord) GetNotificationRecordByName(mc, name string) (*NotificationRecordByName, error) {
	r, err := fnr.GetNotificationRecordByMC(mc)
	if err != nil {
		return nil, err
	}

	for _, rn := range r.Status.NotificationRecordByName {
		if rn.NotificationName == name {
			return &rn, nil
		}
	}
	return nil, fmt.Errorf("cannot find notification record for name %s", name)
}

// GetNotificationRecordItem gets the record item for the specified hosted cluster
func (fnr *ManagedFleetNotificationRecord) GetNotificationRecordItem(mc, name, clusterID string) (*NotificationRecordItem, error) {
	rn, err := fnr.GetNotificationRecordByName(mc, name)
	if err != nil {
		return nil, err
	}

	for _, ri := range rn.NotificationRecordItems {
		if ri.HostedClusterID == clusterID {
			return &ri, nil
		}
	}

	return nil, fmt.Errorf("cannot find notification item by name: %s for cluster %s", name, clusterID)
}

// HasNotificationRecordItem Checks if the notification record with given management cluster and notification name
// exists for the specified hosted cluster
func (fnr *ManagedFleetNotificationRecord) HasNotificationRecordItem(mc, name, clusterID string) bool {
	if mc != fnr.Status.ManagementCluster {
		return false
	}

	rn, err := fnr.GetNotificationRecordByName(mc, name)
	if err != nil {
		return false
	}

	for _, ri := range rn.NotificationRecordItems {
		if ri.HostedClusterID == clusterID {
			return true
		}
	}

	return false
}

// CanBeSent checks if the service log for the notification can be sent for the given hosted cluster
func (fnr *ManagedFleetNotificationRecord) CanBeSent(mc, name, clusterID string) (bool, error) {
	rn, err := fnr.GetNotificationRecordByName(mc, name)
	if err != nil {
		return false, err
	}

	interval := rn.ResendWait

	hasNotificationSent := fnr.HasNotificationRecordItem(mc, name, clusterID)

	if !hasNotificationSent {
		return true, nil
	}

	ri, err := fnr.GetNotificationRecordItem(mc, name, clusterID)
	if err != nil {
		return false, err
	}

	if ri.LastTransitionTime == nil {
		return true, nil
	}

	nextSend := ri.LastTransitionTime.Time.Add(time.Duration(interval) * time.Hour)

	if time.Now().After(nextSend) {
		return true, nil
	}

	return false, nil
}

// AddNotificationRecordItem adds a new record item to the notification record slice
func (fnr *ManagedFleetNotificationRecord) AddNotificationRecordItem(clusterID string, rn *NotificationRecordByName) {
	ri := NotificationRecordItem{
		ServiceLogSentCount: 0,
		HostedClusterID:     clusterID,
		LastTransitionTime:  nil,
	}

	rn.NotificationRecordItems = append(rn.NotificationRecordItems, ri)
}

// UpdateNotificationRecordItem updates the service log sent count and timestamp for the last time sent
func (fnr *ManagedFleetNotificationRecord) UpdateNotificationRecordItem(ri *NotificationRecordItem) *NotificationRecordItem {
	ri.ServiceLogSentCount += 1
	ri.LastTransitionTime = &metav1.Time{Time: time.Now()}

	return ri
}

// RemoveNotificationRecordItem removes the notification record item from the given notification name
func (fnr *ManagedFleetNotificationRecord) RemoveNotificationRecordItem(ri *NotificationRecordItem, rn *NotificationRecordByName) *NotificationRecordByName {
	var p int
	for i, r := range rn.NotificationRecordItems {
		if r.HostedClusterID == ri.HostedClusterID {
			p = i
		}
		rn.NotificationRecordItems = append(rn.NotificationRecordItems[:p], rn.NotificationRecordItems[p+1:]...)
	}

	return rn
}
