package client

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func New(config *rest.Config, options client.Options) (client.Client, error) {
	return client.New(config, disableCacheForObjects(options))
}

// disableCacheForObjects disables caching for runtime objects that are not created by the EventingManager.
func disableCacheForObjects(options client.Options) client.Options {
	options.Cache = &client.CacheOptions{
		DisableFor: []client.Object{
			&corev1.Secret{},
			&corev1.Service{},
			&corev1.ConfigMap{},
		},
	}
	return options
}
