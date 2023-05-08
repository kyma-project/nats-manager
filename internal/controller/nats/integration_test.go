package nats_test

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"testing"
	"time"

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
)

// MockedUnitTestEnvironment provides mocked resources for unit tests.
type IntegrationTestEnvironment struct {
	Context         context.Context
	EnvTestInstance *envtest.Environment
	k8sClient       client.Client
	KubeClient      *k8s.Client
	ChartRenderer   *chart.Renderer
	NATSManager     *manager.Manager
	Reconciler      *natscontroller.Reconciler
	Logger          *zap.SugaredLogger
	Recorder        *record.EventRecorder
	TestCancelFn    context.CancelFunc
}

func NewIntegrationTestEnvironment() (*IntegrationTestEnvironment, error) {
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

	return &IntegrationTestEnvironment{
		Context:         ctx,
		k8sClient:       k8sClient,
		KubeClient:      &kubeClient,
		ChartRenderer:   &helmRenderer,
		Reconciler:      natsReconciler,
		Logger:          sugaredLogger,
		Recorder:        &recorder,
		NATSManager:     &natsManager,
		EnvTestInstance: testEnv,
		TestCancelFn:    cancelCtx,
	}, nil
}

func (ite IntegrationTestEnvironment) TearDown() error {
	if ite.TestCancelFn != nil {
		ite.TestCancelFn()
	}
	return ite.EnvTestInstance.Stop()
}

func (ite IntegrationTestEnvironment) CreateNamespace(ctx context.Context, namespace string) error {
	if namespace == "default" {
		return nil
	}
	// create namespace
	ns := testutils.NewNamespace(namespace)
	err := testEnvironment.k8sClient.Create(ctx, ns)
	if !k8serrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

func (ite IntegrationTestEnvironment) EnsureK8sResourceCreated(t *testing.T, obj client.Object) {
	require.NoError(t, ite.k8sClient.Create(ite.Context, obj))
}

func (ite IntegrationTestEnvironment) GetNATSFromK8s(name, namespace string) (natsv1alpha1.NATS, error) {
	var nats natsv1alpha1.NATS
	err := ite.k8sClient.Get(ite.Context, k8stypes.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &nats)
	return nats, err
}

func (ite IntegrationTestEnvironment) GetStatefulSetFromK8s(name, namespace string) (*appsv1.StatefulSet, error) {
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

func (ite IntegrationTestEnvironment) UpdateStatefulSetStatusOnK8s(sts appsv1.StatefulSet) error {
	return ite.k8sClient.Status().Update(ite.Context, &sts)
}

// GetNATSAssert fetches a NATS from k8s and allows making assertions on it.
func (ite IntegrationTestEnvironment) GetNATSAssert(g *gomega.GomegaWithT,
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
		CRDDirectoryPaths:        []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
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
