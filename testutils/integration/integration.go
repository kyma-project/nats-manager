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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	natsconf "github.com/nats-io/nats-server/conf"

	corev1 "k8s.io/api/core/v1"

	"github.com/avast/retry-go/v3"
	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	natscontroller "github.com/kyma-project/nats-manager/internal/controller/nats"
	"github.com/kyma-project/nats-manager/pkg/k8s"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"github.com/kyma-project/nats-manager/pkg/manager"
	"github.com/kyma-project/nats-manager/testutils"
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
)

const (
	NATSChartDir             = "../../../resources/nats"
	useExistingCluster       = false
	attachControlPlaneOutput = false
	testEnvStartDelay        = time.Minute
	testEnvStartAttempts     = 10
	namespacePrefixLength    = 5
	TwoMinTimeOut            = 120 * time.Second
	BigPollingInterval       = 3 * time.Second
	BigTimeOut               = 40 * time.Second
	SmallTimeOut             = 5 * time.Second
	SmallPollingInterval     = 1 * time.Second

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

	//+kubebuilder:scaffold:scheme

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

func (ite TestEnvironment) TearDown() error {
	if ite.TestCancelFn != nil {
		ite.TestCancelFn()
	}

	// retry to stop the api-server
	sleepTime := 1 * time.Second
	var err error
	const retries = 20
	for i := 0; i < retries; i++ {
		if err = ite.EnvTestInstance.Stop(); err == nil {
			break
		}
		time.Sleep(sleepTime)
	}
	return err
}

func (ite TestEnvironment) CreateNamespace(ctx context.Context, namespace string) error {
	if namespace == "default" {
		return nil
	}
	// create namespace
	ns := testutils.NewNamespace(namespace)
	err := ite.k8sClient.Create(ctx, ns)
	if !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (ite TestEnvironment) EnsureK8sResourceCreated(t *testing.T, obj client.Object) {
	require.NoError(t, ite.k8sClient.Create(ite.Context, obj))
}

func (ite TestEnvironment) EnsureK8sResourceUpdated(t *testing.T, obj client.Object) {
	require.NoError(t, ite.k8sClient.Update(ite.Context, obj))
}

func (ite TestEnvironment) EnsureK8sConfigMapExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := ite.GetConfigMapFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of ConfigMap")
}

func (ite TestEnvironment) EnsureK8sSecretExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := ite.GetSecretFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of Secret")
}

func (ite TestEnvironment) EnsureK8sServiceExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := ite.GetServiceFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of Service")
}

func (ite TestEnvironment) EnsureK8sDestinationRuleExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := ite.GetDestinationRuleFromK8s(name, namespace)
		return err == nil && result != nil
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of DestinationRule")
}

func (ite TestEnvironment) EnsureK8sStatefulSetExists(t *testing.T, name, namespace string) {
	require.Eventually(t, func() bool {
		result, err := ite.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			ite.Logger.Errorw("failed to ensure STS", "error", err,
				"name", name, "namespace", namespace)
		}
		return err == nil && result != nil
	}, BigTimeOut, SmallPollingInterval, "failed to ensure existence of StatefulSet")
}

func (ite TestEnvironment) EnsureK8sStatefulSetHasLabels(t *testing.T, name, namespace string,
	labels map[string]string) {
	require.Eventually(t, func() bool {
		result, err := ite.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			ite.Logger.Errorw("failed to get STS", "error", err,
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

func (ite TestEnvironment) EnsureK8sStatefulSetHasAnnotations(t *testing.T, name, namespace string,
	annotations map[string]string) {
	require.Eventually(t, func() bool {
		result, err := ite.GetStatefulSetFromK8s(name, namespace)
		if err != nil {
			ite.Logger.Errorw("failed to get STS", "error", err,
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
func (ite TestEnvironment) EnsureNATSSpecClusterSizeReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		stsName := GetStatefulSetName(nats)
		result, err := ite.GetStatefulSetFromK8s(stsName, nats.Namespace)
		if err != nil {
			ite.Logger.Errorw("failed to get STS", "error", err,
				"name", stsName, "namespace", nats.Namespace)
			return false
		}
		return nats.Spec.Cluster.Size == int(*result.Spec.Replicas)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure spec.cluster.size")
}

// EnsureNATSSpecResourcesReflected ensures that NATS CR Spec.resources is reflected
// in relevant k8s objects.
func (ite TestEnvironment) EnsureNATSSpecResourcesReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		stsName := GetStatefulSetName(nats)
		result, err := ite.GetStatefulSetFromK8s(stsName, nats.Namespace)
		if err != nil {
			ite.Logger.Errorw("failed to ensure STS", "error", err,
				"name", stsName, "namespace", nats.Namespace)
			return false
		}

		natsContainer := FindContainer(result.Spec.Template.Spec.Containers, NATSContainerName)
		require.NotNil(t, natsContainer)

		return reflect.DeepEqual(nats.Spec.Resources, natsContainer.Resources)
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure existence of StatefulSet resources")
}

// EnsureNATSSpecDebugTraceReflected ensures that NATS CR Spec.trace and Spec.debug is reflected
// in relevant k8s objects.
func (ite TestEnvironment) EnsureNATSSpecDebugTraceReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		// get NATS configMap.
		result, err := ite.GetConfigMapFromK8s(GetConfigMapName(nats), nats.Namespace)
		if err != nil {
			ite.Logger.Errorw("failed to get ConfigMap", "error", err,
				"name", GetConfigMapName(nats), "namespace", nats.Namespace)
			return false
		}

		// get nats config file data from ConfigMap.
		natsConfigStr, ok := result.Data[NATSConfigFileName]
		if !ok {
			return false
		}

		// parse the nats config file data.
		natsConfig, err := ParseNATSConf(natsConfigStr)
		if err != nil {
			ite.Logger.Errorw("failed to parse NATS config", "error", err,
				"name", GetConfigMapName(nats), "namespace", nats.Namespace)
			return false
		}

		// get the trace value.
		gotTrace, ok := natsConfig["trace"]
		if !ok {
			return false
		}

		// get the debug value.
		gotDebug, ok := natsConfig["debug"]
		if !ok {
			return false
		}

		return nats.Spec.Trace == gotTrace && nats.Spec.Debug == gotDebug
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure NATS CR Spec.trace and Spec.debug")
}

// EnsureNATSSpecMemStorageReflected ensures that NATS CR Spec.jetStream.memStorage is reflected
// in relevant k8s objects.
func (ite TestEnvironment) EnsureNATSSpecMemStorageReflected(t *testing.T, nats natsv1alpha1.NATS) {
	require.Eventually(t, func() bool {
		// get NATS configMap.
		result, err := ite.GetConfigMapFromK8s(GetConfigMapName(nats), nats.Namespace)
		if err != nil {
			ite.Logger.Errorw("failed to get ConfigMap", "error", err,
				"name", GetConfigMapName(nats), "namespace", nats.Namespace)
			return false
		}

		// get nats config file data from ConfigMap.
		natsConfigStr, ok := result.Data[NATSConfigFileName]
		if !ok {
			return false
		}

		// parse the nats config file data.
		natsConfig, err := ParseNATSConf(natsConfigStr)
		if err != nil {
			ite.Logger.Errorw("failed to parse NATS config", "error", err,
				"name", GetConfigMapName(nats), "namespace", nats.Namespace)
			return false
		}

		// get the trace value.
		gotJetStream, ok := natsConfig["jetstream"].(map[string]interface{})
		if !ok {
			return false
		}

		gotMemStorage, ok := gotJetStream["max_mem"]
		if !ok {
			return false
		}
		return nats.Spec.MemStorage.Size.String() == gotMemStorage
	}, SmallTimeOut, SmallPollingInterval, "failed to ensure NATS CR Spec.jetStream.memStorage")
}

func (ite TestEnvironment) GetNATSFromK8s(name, namespace string) (natsv1alpha1.NATS, error) {
	var nats natsv1alpha1.NATS
	err := ite.k8sClient.Get(ite.Context, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &nats)
	return nats, err
}

func (ite TestEnvironment) GetStatefulSetFromK8s(name, namespace string) (*appsv1.StatefulSet, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &appsv1.StatefulSet{}
	if err := ite.k8sClient.Get(ite.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ite TestEnvironment) UpdateStatefulSetStatusOnK8s(sts appsv1.StatefulSet) error {
	return ite.k8sClient.Status().Update(ite.Context, &sts)
}

func (ite TestEnvironment) GetConfigMapFromK8s(name, namespace string) (*corev1.ConfigMap, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &corev1.ConfigMap{}
	if err := ite.k8sClient.Get(ite.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ite TestEnvironment) GetSecretFromK8s(name, namespace string) (*corev1.Secret, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &corev1.Secret{}
	if err := ite.k8sClient.Get(ite.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ite TestEnvironment) GetServiceFromK8s(name, namespace string) (*corev1.Service, error) {
	nn := k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	result := &corev1.Service{}
	if err := ite.k8sClient.Get(ite.Context, nn, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ite TestEnvironment) GetDestinationRuleFromK8s(name, namespace string) (*unstructured.Unstructured, error) {
	destinationRuleRes := schema.GroupVersionResource{
		Group:    "networking.istio.io",
		Version:  "v1alpha3",
		Resource: "destinationrules",
	}

	// get from k8s
	return ite.K8sDynamicClient.Resource(destinationRuleRes).Namespace(
		namespace).Get(ite.Context, name, metav1.GetOptions{})
}

// GetNATSAssert fetches a NATS from k8s and allows making assertions on it.
func (ite TestEnvironment) GetNATSAssert(g *gomega.GomegaWithT,
	nats *natsv1alpha1.NATS) gomega.AsyncAssertion {
	return g.Eventually(func() *natsv1alpha1.NATS {
		gotNATS, err := ite.GetNATSFromK8s(nats.Name, nats.Namespace)
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

func NewTestNamespace() string {
	return fmt.Sprintf("ns-%s", testutils.GetRandString(namespacePrefixLength))
}

func GetStatefulSetName(nats natsv1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats", nats.Name)
}

func GetConfigMapName(nats natsv1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats-config", nats.Name)
}

func GetSecretName(nats natsv1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats-secret", nats.Name)
}

func GetServiceName(nats natsv1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats", nats.Name)
}

func GetDestinationRuleName(nats natsv1alpha1.NATS) string {
	return fmt.Sprintf("%s-nats", nats.Name)
}

func FindContainer(containers []corev1.Container, name string) *corev1.Container {
	for _, container := range containers {
		if container.Name == name {
			return &container
		}
	}
	return nil
}

func ParseNATSConf(data string) (map[string]interface{}, error) {
	// replace variables with dummy values
	natsConfigStr := strings.Replace(data, "$POD_NAME", "pod1", 1)
	// remove `include "accounts/resolver.conf"`
	natsConfigStr = strings.Replace(natsConfigStr, "include \"accounts/resolver.conf\"", "", 1)
	return natsconf.Parse(natsConfigStr)
}
