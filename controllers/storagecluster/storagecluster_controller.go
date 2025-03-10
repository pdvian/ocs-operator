package storagecluster

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	nbv1 "github.com/noobaa/noobaa-operator/v5/pkg/apis/noobaa/v1alpha1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/operator-framework/operator-lib/conditions"
	ocsv1 "github.com/red-hat-storage/ocs-operator/api/v1"
	"github.com/red-hat-storage/ocs-operator/controllers/util"
	cephv1 "github.com/rook/rook/pkg/apis/ceph.rook.io/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log = ctrl.Log.WithName("controllers").WithName("StorageCluster")
)

func (r *StorageClusterReconciler) initializeImageVars() error {
	r.images.Ceph = os.Getenv("CEPH_IMAGE")
	r.images.NooBaaCore = os.Getenv("NOOBAA_CORE_IMAGE")
	r.images.NooBaaDB = os.Getenv("NOOBAA_DB_IMAGE")

	if r.images.Ceph == "" {
		err := fmt.Errorf("CEPH_IMAGE environment variable not found")
		r.Log.Error(err, "Missing CEPH_IMAGE environment variable for ocs initialization.")
		return err
	} else if r.images.NooBaaCore == "" {
		err := fmt.Errorf("NOOBAA_CORE_IMAGE environment variable not found")
		r.Log.Error(err, "Missing NOOBAA_CORE_IMAGE environment variable for ocs initialization.")
		return err
	} else if r.images.NooBaaDB == "" {
		err := fmt.Errorf("NOOBAA_DB_IMAGE environment variable not found")
		r.Log.Error(err, "Missing NOOBAA_DB_IMAGE environment variable for ocs initialization.")
		return err
	}
	return nil
}

func (r *StorageClusterReconciler) initializeServerVersion() error {
	clientset, err := kubernetes.NewForConfig(config.GetConfigOrDie())
	if err != nil {
		r.Log.Error(err, "Failed creation of clientset for determining serverversion.")
		return err
	}
	r.serverVersion, err = clientset.Discovery().ServerVersion()
	if err != nil {
		r.Log.Error(err, "Failed getting the serverversion.")
		return err
	}
	return nil
}

// ImageMap holds mapping information between component image name and the image url
type ImageMap struct {
	Ceph       string
	NooBaaCore string
	NooBaaDB   string
}

// StorageClusterReconciler reconciles a StorageCluster object
//nolint
type StorageClusterReconciler struct {
	client.Client
	ctx                context.Context
	Log                logr.Logger
	Scheme             *runtime.Scheme
	serverVersion      *version.Info
	conditions         []conditionsv1.Condition
	phase              string
	nodeCount          int
	platform           *Platform
	images             ImageMap
	recorder           *util.EventReporter
	OperatorCondition  conditions.Condition
	IsNoobaaStandalone bool
}

// SetupWithManager sets up a controller with manager
func (r *StorageClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.initializeImageVars(); err != nil {
		return err
	}

	if err := r.initializeServerVersion(); err != nil {
		return err
	}

	r.platform = &Platform{}
	r.recorder = util.NewEventReporter(mgr.GetEventRecorderFor("controller_storagecluster"))

	// Compose a predicate that is an OR of the specified predicates
	scPredicate := util.ComposePredicates(
		predicate.GenerationChangedPredicate{},
		util.MetadataChangedPredicate{},
	)

	pvcPredicate := predicate.Funcs{
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&ocsv1.StorageCluster{}, builder.WithPredicates(scPredicate)).
		Owns(&cephv1.CephCluster{}).
		Owns(&nbv1.NooBaa{}).
		Owns(&corev1.PersistentVolumeClaim{}, builder.WithPredicates(pvcPredicate)).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(
			&source.Kind{Type: &ocsv1.OCSInitialization{}},
			&handler.EnqueueRequestForObject{},
		).
		Complete(r)
}
