package nats

import (
	"context"
	"time"

	"github.com/kyma-project/nats-manager/pkg/events"

	natsv1alpha1 "github.com/kyma-project/nats-manager/api/v1alpha1"
	"github.com/kyma-project/nats-manager/pkg/k8s/chart"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

const RequeueTimeForStatusCheck = 10

func (r *Reconciler) handleNATSReconcile(ctx context.Context,
	nats *natsv1alpha1.NATS, log *zap.SugaredLogger) (ctrl.Result, error) {
	log.Info("handling NATS reconciliation...")

	// set status to processing
	nats.Status.Initialize()
	events.Normal(r.recorder, nats, natsv1alpha1.ConditionReasonProcessing, "Initializing NATS resource.")

	// make sure the finalizer exists.
	if !r.containsFinalizer(nats) {
		return r.addFinalizer(ctx, nats)
	}

	log.Info("init NATS resources...")
	// init a release instance (NATS resources to deploy)
	instance, err := r.initNATSInstance(ctx, nats, log)
	if err != nil {
		events.Warn(r.recorder, nats, natsv1alpha1.ConditionReasonProcessingError,
			"Error while NATS resources were being initialized: %s", err)
		return ctrl.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
	}

	log.Info("deploying NATS resources...")
	// deploy NATS resources
	if err = r.natsManager.DeployInstance(ctx, instance); err != nil {
		events.Warn(r.recorder, nats, natsv1alpha1.ConditionReasonProcessingError,
			"Error while NATS resources were deployed: %s", err)
		return ctrl.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
	}

	// watchers for dynamic resources managed by controller.
	if instance.IstioEnabled && !r.destinationRuleWatchStarted {
		if err = r.watchDestinationRule(log); err != nil {
			events.Warn(r.recorder, nats, natsv1alpha1.ConditionReasonProcessingError,
				"Error while NATS resources were watched: %s", err)
			return ctrl.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
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
func (r *Reconciler) handleNATSState(ctx context.Context, nats *natsv1alpha1.NATS, instance *chart.ReleaseInstance,
	log *zap.SugaredLogger) (ctrl.Result, error) {
	// checking if statefulSet is ready.
	isSTSReady, err := r.natsManager.IsNATSStatefulSetReady(ctx, instance)
	if err != nil {
		nats.Status.UpdateConditionStatefulSet(metav1.ConditionFalse,
			natsv1alpha1.ConditionReasonSyncFailError, err.Error())
		events.Warn(r.recorder, nats, natsv1alpha1.ConditionReasonSyncFailError,
			"Failed to sync the resources. StatefulSet is not ready.")
		return ctrl.Result{}, r.syncNATSStatusWithErr(ctx, nats, err, log)
	}

	if isSTSReady {
		nats.Status.SetStateReady()
		events.Normal(r.recorder, nats, natsv1alpha1.ConditionReasonDeployed, "StatefulSet is ready and NATS is deployed.")
	} else {
		nats.Status.SetWaitingStateForStatefulSet()
		events.Normal(r.recorder, nats, natsv1alpha1.ConditionReasonDeploying,
			"NATS is being deployed, waiting for StatefulSet to get ready.")
		r.logger.Info("Reconciliation successful: waiting for STS to get ready...")
		return ctrl.Result{RequeueAfter: RequeueTimeForStatusCheck * time.Second}, r.syncNATSStatus(ctx, nats, log)
	}

	r.logger.Info("Reconciliation successful")
	return ctrl.Result{}, r.syncNATSStatus(ctx, nats, log)
}
