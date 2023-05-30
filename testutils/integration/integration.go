package integration

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	corev1 "k8s.io/api/core/v1"

	"github.com/avast/retry-go/v3"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	apiclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natscontroller "github.com/kyma-project/nats-manager/internal/controller/nats"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/pkg/manager"
	"github.com/kyma-project/nats-manager/testutils"
)

const (
	NATSChartDir             = "../../../resources/nats"
	useExistingCluster       = false
	attachControlPlaneOutput = false
	testEnvStartDelay        = time.Minute
	testEnvStartAttempts     = 10

	TwoMinTimeOut        = 120 * time.Second
	BigPollingInterval   = 3 * time.Second
	BigTimeOut           = 60 * time.Second
	SmallTimeOut         = 10 * time.Second
	SmallPollingInterval = 1 * time.Second

	NATSContainerName  = "nats"
	NATSConfigFileName = "nats.conf"
)

// TestEnvironment provides mocked resources for integration tests.
type TestEnvironment struct {
	Context          context.Context
	EnvTestInstance  *envtest.Environment
	k8sClient        client.Client
	K8sDynamicClient *dynamic.DynamicClient
	KubeClient       *k8s.Client
	ChartRenderer    *chart.Renderer
	NATSManager      *manager.Manager
	Reconciler       *natscontroller.Reconciler
	Logger           *zap.SugaredLogger
	Recorder         *record.EventRecorder
	TestCancelFn     context.CancelFunc
}

func NewTestEnvironment() (*TestEnvironment, error) { //nolint:funlen // Used in testing.
	var err error
	// setup context
	ctx := context.Background()

	// setup logger
	sugaredLogger, err := testutils.NewSugaredLogger()
	if err != nil {
		return nil, err
	}

	testEnv, envTestKubeCfg, err := StartEnvTest()
	if err != nil {
		return nil, err
	}

	// add to Scheme
	err = natsv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	// +kubebuilder:scaffold:scheme

	k8sClient, err := client.New(envTestKubeCfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(envTestKubeCfg)
	if err != nil {
		return nil, err
	}

	// setup ctrl manager
	metricsPort, err := testutils.GetFreePort()
	if err != nil {
		return nil, err
	}

	ctrlMgr, err := ctrl.NewManager(envTestKubeCfg, ctrl.Options{
		Scheme: scheme.Scheme,
		Port:   metricsPort,
	})
	if err != nil {
		return nil, err
	}
	recorder := ctrlMgr.GetEventRecorderFor("nats-manager")

	// init custom kube client wrapper
	apiClientSet, err := apiclientset.NewForConfig(ctrlMgr.GetConfig())
	if err != nil {
		return nil, err
	}
	kubeClient := k8s.NewKubeClient(ctrlMgr.GetClient(), apiClientSet, "nats-manager")

	// create helmRenderer
	helmRenderer, err := chart.NewHelmRenderer(NATSChartDir, sugaredLogger)
	if err != nil {
		return nil, err
	}

	// create NATS manager instance
	natsManager := manager.NewNATSManger(kubeClient, helmRenderer, sugaredLogger)

	// setup reconciler
	natsReconciler := natscontroller.NewReconciler(
		ctrlMgr.GetClient(),
		kubeClient,
		helmRenderer,
		ctrlMgr.GetScheme(),
		sugaredLogger,
		recorder,
		natsManager,
	)
	if err = (natsReconciler).SetupWithManager(ctrlMgr); err != nil {
		return nil, err
	}

	// start manager
	var cancelCtx context.CancelFunc
	go func() {
		var mgrCtx context.Context
		mgrCtx, cancelCtx = context.WithCancel(ctrl.SetupSignalHandler())
		err = ctrlMgr.Start(mgrCtx)
		if err != nil {
			panic(err)
		}
	}()

	return &TestEnvironment{
		Context:          ctx,
		k8sClient:        k8sClient,
		K8sDynamicClient: dynamicClient,
		KubeClient:       &kubeClient,
		ChartRenderer:    &helmRenderer,
		Reconciler:       natsReconciler,
		Logger:           sugaredLogger,
		Recorder:         &recorder,
		NATSManager:      &natsManager,
		EnvTestInstance:  testEnv,
		TestCancelFn:     cancelCtx,
	}, nil
}

func (env TestEnvironment) TearDown() error {
	if env.TestCancelFn != nil {
		env.TestCancelFn()
	}

	// retry to stop the api-server
	sleepTime := 1 * time.Second
	var err error
	const retries = 20
	for i := 0; i < retries; i++ {
		if err = env.EnvTestInstance.Stop(); err == nil {
			break
		}
		time.Sleep(sleepTime)
	}
	return err
}

func (env TestEnvironment) EnsureNamespaceCreation(t *testing.T, namespace string) {
	if namespace == "default" {
		return
	}
	// create namespace
	ns := testutils.NewNamespace(namespace)
	require.NoError(t, env.k8sClient.Create(env.Context, ns))
}

func (env TestEnvironment) EnsureK8sResourceCreated(t *testing.T, obj client.Object) {
	require.NoError(t, env.k8sClient.Create(env.Context, obj))
}

func (env TestEnvironment) EnsureK8sResourceUpdated(t *testing.T, obj client.Object) {
	require.NoError(t, env.k8sClient.Update(env.Context, obj))
}

func (env TestEnvironment) EnsureK8sResourceDeleted(t *testing.T, obj client.Object) {
	require.NoError(t, env.k8sClient.Delete(env.Context, obj))
}

func (env TestEnvironment) EnsureK8sConfigMapExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetConfigMapFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of ConfigMap")
}

func (env TestEnvironment) EnsureK8sSecretExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetSecretFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of Secret")
}

func (env TestEnvironment) EnsureK8sServiceExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetServiceFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of Service")
}

func (env TestEnvironment) EnsureK8sDestinationRuleExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetDestinationRuleFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of DestinationRule")
}

func (env TestEnvironment) EnsureK8sStatefulSetExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := env.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			env.Logger.Errorw("failed to ensure STS", "error", err,
				"name", name, "namespace", namespace)
		}
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of StatefulSet")
}

func (env TestEnvironment) EnsureK8sConfigMapNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetConfigMapFromK8s(name, namespace)
		return err != nil && k8serrors.IsNotFound(err)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure non-existence of ConfigMap")
}

func (env TestEnvironment) EnsureK8sSecretNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetSecretFromK8s(name, namespace)
		return err != nil && k8serrors.IsNotFound(err)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure non-existence of Secret")
}

func (env TestEnvironment) EnsureK8sServiceNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetServiceFromK8s(name, namespace)
		return err != nil && k8serrors.IsNotFound(err)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure non-existence of Service")
}

func (env TestEnvironment) EnsureK8sDestinationRuleNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetDestinationRuleFromK8s(name, namespace)
		return err != nil && k8serrors.IsNotFound(err)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure non-existence of DestinationRule")
}

func (env TestEnvironment) EnsureK8sNATSNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetNATSFromK8s(name, namespace)
		return err != nil && k8serrors.IsNotFound(err)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure non-existence of NATS")
}

func (env TestEnvironment) EnsureK8sStatefulSetNotFound(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		_, err := env.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			env.Logger.Errorw("failed to ensure STS", "error", err,
				"name", name, "namespace", namespace)
		}
		return err != nil && k8serrors.IsNotFound(err)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure non-existence of StatefulSet")
}

func (env TestEnvironment) EnsureK8sStatefulSetHasLabels(t *testing.T, name, namespace string,
	labels map[string]string) {
	require.Eventually(t, func() bool {
		result, err := env.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			env.Logger.Errorw("failed to get STS", "error", err,
				"name", name, "namespace", namespace)
			return false
		}

		for k, v := range labels {
			value, ok := result.Labels[k]
			if !ok || v != value {
				return false
			}
		}
		return true
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure labels")
}

func (env TestEnvironment) EnsureK8sStatefulSetHasAnnotations(t *testing.T, name, namespace string,
	annotations map[string]string) {
	require.Eventually(t, func() bool {
		result, err := env.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			env.Logger.Errorw("failed to get STS", "error", err,
				"name", name, "namespace", namespace)
			return false
		}

		for k, v := range annotations {
			value, ok := result.Annotations[k]
			if !ok || v != value {
				return false
			}
		}
		return true
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure annotations")
}

// EnsureNATSSpecClusterSizeReflected ensures that NATS CR Spec.cluster.size is reflected
// in relevant k8s objects.
func (env TestEnvironment) EnsureNATSSpecClusterSizeReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		stsName := testutils.GetStatefulSetName(nats)
		result, err := env.GetStatefulSetFromK8s(stsName, nats.Namespace)
		if err != nil {
			env.Logger.Errorw("failed to get STS", "error", err,
				"name", stsName, "namespace", nats.Namespace)
			return false
		}
		return nats.Spec.Cluster.Size == int(*result.Spec.Replicas)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure spec.cluster.size")
}

// EnsureNATSSpecResourcesReflected ensures that NATS CR Spec.resources is reflected
// in relevant k8s objects.
func (env TestEnvironment) EnsureNATSSpecResourcesReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		stsName := testutils.GetStatefulSetName(nats)
		result, err := env.GetStatefulSetFromK8s(stsName, nats.Namespace)
		if err != nil {
			env.Logger.Errorw("failed to ensure STS", "error", err,
				"name", stsName, "namespace", nats.Namespace)
			return false
		}

		natsContainer := testutils.FindContainer(result.Spec.Template.Spec.Containers, NATSContainerName)
		require.NotNil(t, natsContainer)

		return reflect.DeepEqual(nats.Spec.Resources, natsContainer.Resources)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of StatefulSet resources")
}

// EnsureNATSSpecDebugTraceReflected ensures that NATS CR Spec.trace and Spec.debug is reflected
// in relevant k8s objects.
func (env TestEnvironment) EnsureNATSSpecDebugTraceReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		// get NATS configMap.
		result, err := env.GetConfigMapFromK8s(testutils.GetConfigMapName(nats), nats.Namespace)
		if err != nil {
			env.Logger.Errorw("failed to get ConfigMap", "error", err,
				"name", testutils.GetConfigMapName(nats), "namespace", nats.Namespace)
			return false
		}

		// get nats config file data from ConfigMap.
		natsConfigStr, ok := result.Data[NATSConfigFileName]
		if !ok {
			return false
		}

		debugCheck := strings.Contains(natsConfigStr, fmt.Sprintf("debug: %t", nats.Spec.Debug))
		traceCheck := strings.Contains(natsConfigStr, fmt.Sprintf("trace: %t", nats.Spec.Trace))
		return debugCheck && traceCheck
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure NATS CR Spec.trace and Spec.debug")
}

// EnsureNATSSpecFileStorageReflected ensures that NATS CR Spec.jetStream.fileStorage is reflected
// in relevant k8s objects.
func (env TestEnvironment) EnsureNATSSpecFileStorageReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		// get NATS configMap.
		result, err := env.GetConfigMapFromK8s(testutils.GetConfigMapName(nats), nats.Namespace)
		if err != nil {
			env.Logger.Errorw("failed to get ConfigMap", "error", err,
				"name", testutils.GetConfigMapName(nats), "namespace", nats.Namespace)
			return false
		}

		// get nats config file data from ConfigMap.
		natsConfigStr, ok := result.Data[NATSConfigFileName]
		if !ok {
			return false
		}

		// check if file storage size is correctly defined in NATS config.
		if !strings.Contains(natsConfigStr, fmt.Sprintf("max_file: %s", nats.Spec.FileStorage.Size.String())) {
			return false
		}

		// now check the PVC info in StatefulSet.
		sts, err := env.GetStatefulSetFromK8s(testutils.GetStatefulSetName(nats), nats.Namespace)
		if err != nil {
			env.Logger.Errorw("failed to get STS", "error", err,
				"name", testutils.GetStatefulSetName(nats), "namespace", nats.Namespace)
			return false
		}

		if *sts.Spec.VolumeClaimTemplates[0].Spec.StorageClassName != nats.Spec.FileStorage.StorageClassName {
			return false
		}

		if sts.Spec.VolumeClaimTemplates[0].Spec.Resources.Requests.
			Storage().String() != nats.Spec.FileStorage.Size.String() {
			return false
		}

		return true
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure NATS CR Spec.jetStream.fileStorage")
}

// EnsureNATSSpecMemStorageReflected ensures that NATS CR Spec.jetStream.memStorage is reflected
// in relevant k8s objects.
func (env TestEnvironment) EnsureNATSSpecMemStorageReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		// get NATS configMap.
		result, err := env.GetConfigMapFromK8s(testutils.GetConfigMapName(nats), nats.Namespace)
		if err != nil {
			env.Logger.Errorw("failed to get ConfigMap", "error", err,
				"name", testutils.GetConfigMapName(nats), "namespace", nats.Namespace)
			return false
		}

		// get nats config file data from ConfigMap.
		natsConfigStr, ok := result.Data[NATSConfigFileName]
		if !ok {
			return false
		}

		// check if mem storage size is correctly defined in NATS config.
		return strings.Contains(natsConfigStr, fmt.Sprintf("max_mem: %s", nats.Spec.MemStorage.Size.String()))
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure NATS CR Spec.jetStream.memStorage")
}

func (env TestEnvironment) GetNATSFromK8s(name, namespace string) (natsv1alpha1.NATS, error) {
	var nats natsv1alpha1.NATS
	err := env.k8sClient.Get(env.Context, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &nats)
	return nats, err
}

func (env TestEnvironment) GetStatefulSetFromK8s(name, namespace string) (*appsv1.StatefulSet, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &appsv1.StatefulSet{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) UpdateStatefulSetStatusOnK8s(sts appsv1.StatefulSet) error {
	return env.k8sClient.Status().Update(env.Context, &sts)
}

func (env TestEnvironment) GetConfigMapFromK8s(name, namespace string) (*corev1.ConfigMap, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &corev1.ConfigMap{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) GetSecretFromK8s(name, namespace string) (*corev1.Secret, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &corev1.Secret{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) GetServiceFromK8s(name, namespace string) (*corev1.Service, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &corev1.Service{}
	if err := env.k8sClient.Get(env.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (env TestEnvironment) GetDestinationRuleFromK8s(name, namespace string) (*unstructured.Unstructured, error) {
	return env.K8sDynamicClient.Resource(testutils.GetDestinationRuleGVR()).Namespace(
		namespace).Get(env.Context, name, metav1.GetOptions{})
}

func (env TestEnvironment) DeleteStatefulSetFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) DeleteServiceFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) DeleteConfigMapFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) DeleteSecretFromK8s(name, namespace string) error {
	return env.k8sClient.Delete(env.Context, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}

func (env TestEnvironment) DeleteDestinationRuleFromK8s(name, namespace string) error {
	return env.K8sDynamicClient.Resource(testutils.GetDestinationRuleGVR()).Namespace(
		namespace).Delete(env.Context, name, metav1.DeleteOptions{})
}

// GetNATSAssert fetches a NATS from k8s and allows making assertions on it.
func (env TestEnvironment) GetNATSAssert(g *gomega.GomegaWithT,
	nats *natsv1alpha1.NATS) gomega.AsyncAssertion {
	return g.Eventually(func() *natsv1alpha1.NATS {
		gotNATS, err := env.GetNATSFromK8s(nats.Name, nats.Namespace)
		if err != nil {
			log.Printf("fetch subscription %s/%s failed: %v", nats.Name, nats.Namespace, err)
			return &natsv1alpha1.NATS{}
		}
		return &gotNATS
	}, BigTimeOut, SmallPollingInterval)
}

func StartEnvTest() (*envtest.Environment, *rest.Config, error) {
	// Reference: https://book.kubebuilder.io/reference/envtest.html
	useExistingCluster := useExistingCluster
	testEnv := &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "..", "..", "config", "crd", "bases"),
			filepath.Join("..", "..", "..", "config", "crd", "external"),
		},
		ErrorIfCRDPathMissing:    true,
		AttachControlPlaneOutput: attachControlPlaneOutput,
		UseExistingCluster:       &useExistingCluster,
	}

	var cfg *rest.Config
	err := retry.Do(func() error {
		defer func() {
			if r := recover(); r != nil {
				log.Println("panic recovered:", r)
			}
		}()

		cfgLocal, startErr := testEnv.Start()
		cfg = cfgLocal
		return startErr
	},
		retry.Delay(testEnvStartDelay),
		retry.DelayType(retry.FixedDelay),
		retry.Attempts(testEnvStartAttempts),
		retry.OnRetry(func(n uint, err error) {
			log.Printf("[%v] try failed to start testenv: %s", n, err)
			if stopErr := testEnv.Stop(); stopErr != nil {
				log.Printf("failed to stop testenv: %s", stopErr)
			}
		}),
	)
	return testEnv, cfg, err
}
