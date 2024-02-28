package nats

import (
	"context"
	"time"

	"go.uber.org/zap"
	kmetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kcontrollerruntime "sigs.k8s.io/controller-runtime"

	nmapiv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	nmctrlurl "github.com/kyma-project/nats-manager/internal/controller/nats/url"
	"github.com/kyma-project/nats-manager/pkg/events"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
)

const RequeueTimeForStatusCheck = 10

func (r *Reconciler) handleNATSReconcile(ctx context.Context,
	nats *nmapiv1alpha1.NATS, log *zap.SugaredLogger,
) (kcontrollerruntime.Result, error) {
	log.Info("handling NATS reconciliation...")

	// set status to processing
	nats.Status.Initialize()
	events.Normal(r.recorder, nats, nmapiv1alpha1.ConditionReasonProcessing, "Initializing NATS resource.")

	// make sure the finalizer exists.
	if !r.containsFinalizer(nats) {
		return r.addFinalizer(ctx, nats)
	}

	log.Info("init NATS resources...")
	// init a release instance (NATS resources to deploy)
	instance, err := r.initNATSInstance(ctx, nats, log)
	if err != nil {
		events.Warn(r.recorder, nats, nmapiv1alpha1.ConditionReasonProcessingError,
			"Error while NATS resources were being initialized: %s", err)
		return kcontrollerruntime.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
	}

	log.Info("deploying NATS resources...")
	// deploy NATS resources
	if err = r.natsManager.DeployInstance(ctx, instance); err != nil {
		events.Warn(r.recorder, nats, nmapiv1alpha1.ConditionReasonProcessingError,
			"Error while NATS resources were deployed: %s", err)
		return kcontrollerruntime.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
	}

	// watchers for dynamic resources managed by controller.
	if instance.IstioEnabled && !r.destinationRuleWatchStarted {
		if err = r.watchDestinationRule(log); err != nil {
			events.Warn(r.recorder, nats, nmapiv1alpha1.ConditionReasonProcessingError,
				"Error while NATS resources were watched: %s", err)
			return kcontrollerruntime.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
		}
		// update flag to keep track if watcher is started.
		r.destinationRuleWatchStarted = true
		log.Info("watcher for DestinationRules started")
	}

	log.Info("handling NATS state in CR...")
	// check if NATS resources are ready and sync the NATS CR status.
	return r.handleNATSState(ctx, nats, instance, log)
}

// handleNATSState checks if NATS resources are ready.
// It also syncs the NATS CR status.
func (r *Reconciler) handleNATSState(ctx context.Context, nats *nmapiv1alpha1.NATS, instance *chart.ReleaseInstance,
	log *zap.SugaredLogger,
) (kcontrollerruntime.Result, error) {
	// Clear the url until the StatefulSet is ready.
	nats.Status.ClearURL()

	// checking if statefulSet is ready.
	isSTSReady, err := r.natsManager.IsNATSStatefulSetReady(ctx, instance)
	if err != nil {
		nats.Status.UpdateConditionStatefulSet(kmetav1.ConditionFalse,
			nmapiv1alpha1.ConditionReasonSyncFailError, err.Error())
		events.Warn(r.recorder, nats, nmapiv1alpha1.ConditionReasonSyncFailError,
			"Failed to sync the resources. StatefulSet is not ready.")
		return kcontrollerruntime.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
	}

	if isSTSReady {
		nats.Status.SetStateReady()
		nats.Status.SetURL(nmctrlurl.Format(nats.Name, nats.Namespace))
		events.Normal(r.recorder, nats, nmapiv1alpha1.ConditionReasonDeployed, "StatefulSet is ready and NATS is deployed.")
	} else {
		nats.Status.SetWaitingStateForStatefulSet()
		events.Normal(r.recorder, nats, nmapiv1alpha1.ConditionReasonDeploying,
			"NATS is being deployed, waiting for StatefulSet to get ready.")
		r.logger.Info("Reconciliation successful: waiting for STS to get ready...")
		return kcontrollerruntime.Result{RequeueAfter: RequeueTimeForStatusCheck * time.Second}, r.syncNATSStatus(ctx, nats, log)
	}

	r.logger.Info("Reconciliation successful")
	return kcontrollerruntime.Result{}, r.syncNATSStatus(ctx, nats, log)
}
