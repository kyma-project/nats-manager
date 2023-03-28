package chart

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	yamlUtil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/yaml"
)

func IsStatefulSetObject(u unstructured.Unstructured) bool {
	return u.GetKind() == "StatefulSet"
}

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
