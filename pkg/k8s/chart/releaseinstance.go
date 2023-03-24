package chart

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"

	"github.com/imdario/mergo"
)

func NewReleaseInstance(name, namespace string, configuration map[string]interface{}) *ReleaseInstance {
	return &ReleaseInstance{
		Name:          name,
		Namespace:     namespace,
		Configuration: configuration,
	}
}

type ReleaseInstance struct {
	Name          string
	Namespace     string
	Configuration map[string]interface{}
	RenderedManifests  ManifestResources
}

func (c *ReleaseInstance) GetConfiguration() (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for key, value := range c.Configuration {
		if err := mergo.Merge(&result, c.convertToNestedMap(key, value), mergo.WithOverride); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// GetStatefulSets returns a list of statefulsets from rendered manifests
func (c *ReleaseInstance) GetStatefulSets() []*unstructured.Unstructured {
	var result []*unstructured.Unstructured
	for _, r := range c.RenderedManifests.Items {
		if IsStatefulSetObject(*r) {
			result = append(result, r)
		}
	}
	return result
}

func (c *ReleaseInstance) SetRenderedManifests(renderedManifests ManifestResources) {
	c.RenderedManifests = renderedManifests
}

// convertToNestedMap converts a key with dot-notation into a nested map (e.g. a.b.c=value become [a:[b:[c:value]]])
func (c *ReleaseInstance) convertToNestedMap(key string, value interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	tokens := strings.Split(key, ".")
	lastNestedMap := result
	for depth, token := range tokens {
		switch depth {
		case len(tokens) - 1: //last token reached, stop nesting
			lastNestedMap[token] = value
		default:
			lastNestedMap[token] = make(map[string]interface{})
			lastNestedMap = lastNestedMap[token].(map[string]interface{})
		}
	}
	return result
}
