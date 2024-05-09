package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KollektorSpecResource struct {
	Type          string `json:"type"`
	Name          string `json:"name"`
	ContainerName string `json:"containerName,omitempty"`
}

type KollektorSpecSource struct {
	Repo      string `json:"repo"`
	ChartRepo string `json:"chartRepo,omitempty"`
}

type KollektorSpec struct {
	Source   KollektorSpecSource   `json:"source"`
	Resource KollektorSpecResource `json:"resource"`
}

type KollektorStatus struct {
	Current    string             `json:"current"`
	Latest     string             `json:"latest"`
	IsLatest   string             `json:"isLatest"`
	Conditions []metav1.Condition `json:"conditions"`
}

type Kollektor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KollektorSpec   `json:"spec"`
	Status KollektorStatus `json:"status,omitempty"`
}

type KollektorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kollektor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Kollektor{}, &KollektorList{})
}
