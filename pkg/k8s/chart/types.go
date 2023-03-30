package chart

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:generate mockery --name=Renderer --outpkg=mocks --case=underscore
type Renderer interface {
	// RenderManifest of the given chart.
	RenderManifest(*ReleaseInstance) (string, error)

	// RenderManifestAsUnstructured of the given chart as unstructured objects.
	RenderManifestAsUnstructured(*ReleaseInstance) (*ManifestResources, error)
}

// ManifestResources holds a collection of objects.
type ManifestResources struct {
	Items []*unstructured.Unstructured
	Blobs [][]byte
}
