package chart

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlUtil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
	"strings"
)

func ParseManifestStringToObjects(manifest string) (*ManifestResources, error) {
	objects := &ManifestResources{}
	reader := yamlUtil.NewYAMLReader(bufio.NewReader(strings.NewReader(manifest)))
	for {
		rawBytes, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return objects, nil
			}

			return nil, fmt.Errorf("invalid YAML doc: %w", err)
		}
		rawBytes = bytes.TrimSpace(rawBytes)
		unstructuredObj := unstructured.Unstructured{}
		if err := yaml.Unmarshal(rawBytes, &unstructuredObj); err != nil {
			objects.Blobs = append(objects.Blobs, append(bytes.TrimPrefix(rawBytes, []byte("---\n")), '\n'))
		}

		if len(rawBytes) == 0 || bytes.Equal(rawBytes, []byte("null")) || len(unstructuredObj.Object) == 0 {
			continue
		}

		objects.Items = append(objects.Items, &unstructuredObj)
	}
}

func ConvertUnStructToStructObjects(resources *ManifestResources) ([]metav1.Object, error) {
	structObjects := make([]metav1.Object, 0)
	for _, object := range resources.Items {
		result, err := ConvertUnStructToStructObject(object)
		if err != nil {
			return structObjects, err
		}
		structObjects = append(structObjects, result)
	}

	return structObjects, nil
}

func ConvertUnStructToStructObject(object *unstructured.Unstructured) (metav1.Object, error) {
	var result metav1.Object
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(object.UnstructuredContent(), &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}


