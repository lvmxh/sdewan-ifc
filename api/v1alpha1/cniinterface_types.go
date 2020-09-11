/*


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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CniInterfaceSpec defines the desired state of CniInterface
type CniInterfaceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of CniInterface. Edit CniInterface_types.go to remove/update
	Name        string               `json:"name,omitempty"`
	Status      string               `json:"status,omitempty"`
	MacAddress  string               `json:"mac_address,omitempty"`
	IpAddress   []string             `json:"ip_address,omitempty"`
	CnfSelector metav1.LabelSelector `json:"cnfSelector,omitempty"`
}

// CniInterfaceStatus defines the observed state of CniInterface
type CniInterfaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ReceivedPackets string   `json:"received_packets,omitempty"`
	SendPackets     string   `json:"send_packets,omitempty"`
	Status          string   `json:"status,omitempty"`
	MacAddress      string   `json:"mac_address,omitempty"`
	IpAddress       []string `json:"ip_address,omitempty"`
}

// +kubebuilder:object:root=true

// CniInterface is the Schema for the cniinterfaces API
type CniInterface struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CniInterfaceSpec   `json:"spec,omitempty"`
	Status CniInterfaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CniInterfaceList contains a list of CniInterface
type CniInterfaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CniInterface `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CniInterface{}, &CniInterfaceList{})
}
