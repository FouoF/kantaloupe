package utils

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
)

type JSONPatchOperation string

const (
	JSONPatchOperationAdd     JSONPatchOperation = "add"
	JSONPatchOperationRemove  JSONPatchOperation = "remove"
	JSONPatchOperationReplace JSONPatchOperation = "replace"
	JSONPatchOperationMove    JSONPatchOperation = "move"
	JSONPatchOperationCopy    JSONPatchOperation = "copy"
	JSONPatchOperationTest    JSONPatchOperation = "test"
)

type JSONPatch struct {
	Operation JSONPatchOperation `json:"op"`
	Path      string             `json:"path,omitempty"`
	Value     interface{}        `json:"value,omitempty"`
	From      string             `json:"from,omitempty"`
}

type JSONPatchList []*JSONPatch

func AddJSONPatch(jps ...*JSONPatch) JSONPatchList {
	list := make([]*JSONPatch, 0)
	list = append(list, jps...)
	return list
}

func (jps JSONPatchList) ToBytes() ([]byte, error) {
	b, err := json.Marshal(jps)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// LabelsConvert converts labels data structure to JSONPatchType.
func LabelsConvert(labels map[string]string) ([]byte, error) {
	return AddJSONPatch(&JSONPatch{
		Operation: JSONPatchOperationReplace,
		Path:      "/metadata/labels",
		Value:     labels,
	}).ToBytes()
}

// AnnotationsConvert converts labels data structure to JSONPatchType.
func AnnotationsConvert(annotations map[string]string) ([]byte, error) {
	return AddJSONPatch(&JSONPatch{
		Operation: JSONPatchOperationReplace,
		Path:      "/metadata/annotations",
		Value:     annotations,
	}).ToBytes()
}

// NodeTaintsConvert converts node taints data structure to JSONPatchType.
func NodeTaintsConvert(taints []*corev1.Taint) ([]byte, error) {
	return AddJSONPatch(&JSONPatch{
		Operation: JSONPatchOperationReplace,
		Path:      "/spec/taints",
		Value:     taints,
	}).ToBytes()
}

// NodeScheduleConvert converts node spec.unschedulable to JSONPatchType.
func NodeScheduleConvert(unschedulable bool) ([]byte, error) {
	return AddJSONPatch(&JSONPatch{
		Operation: JSONPatchOperationReplace,
		Path:      "/spec/unschedulable",
		Value:     unschedulable,
	}).ToBytes()
}
