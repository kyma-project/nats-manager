package chart

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//go:generate mockery --name=Renderer --outpkg=mocks --case=underscore
type Renderer interface {
	// RenderManifest of the given chart.
	RenderManifest(*ReleaseInstance) (string, error)

	// RenderManifestAsUnStructured of the given chart as unstructured objects.
	RenderManifestAsUnStructured(*ReleaseInstance) (*ManifestResources, error)

	// RenderManifestAsObjects of the given chart as structured objects.
	RenderManifestAsObjects(*ReleaseInstance) ([]metav1.Object, error)
}

// ManifestResources holds a collection of objects, so that we can filter / sequence them.
type ManifestResources struct {
	Items []*unstructured.Unstructured
	Blobs [][]byte
}
