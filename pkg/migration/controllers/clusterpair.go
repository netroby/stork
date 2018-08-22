package controllers

import (
	"context"
	"reflect"

	"github.com/libopenstorage/stork/drivers/volume"
	stork "github.com/libopenstorage/stork/pkg/apis/stork"
	storkv1 "github.com/libopenstorage/stork/pkg/apis/stork/v1alpha1"
	"github.com/libopenstorage/stork/pkg/controller"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/portworx/sched-ops/k8s"
	"github.com/sirupsen/logrus"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

//ClusterPairController pair
type ClusterPairController struct {
	Driver volume.Driver
}

//Init init
func (c *ClusterPairController) Init() error {
	err := c.createCRD()
	if err != nil {
		return err
	}

	return controller.Register(
		&schema.GroupVersionKind{
			Group:   stork.GroupName,
			Version: stork.Version,
			Kind:    reflect.TypeOf(storkv1.ClusterPair{}).Name(),
		},
		"",
		resyncPeriod,
		c)
}

// Handle updates for ClusterPair objects
func (c *ClusterPairController) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *storkv1.ClusterPair:

		clusterPair := o
		if event.Deleted {
			if clusterPair.Status.RemoteStorageID != "" {
				return c.Driver.DeletePair(clusterPair)
			}
		}

		if clusterPair.Status.StorageStatus == storkv1.ClusterPairStatusInitial {
			logrus.Infof("New cluster pair created %v", clusterPair.Name)
			remoteID, err := c.Driver.CreatePair(clusterPair)
			if err != nil {
				return err
			}
			clusterPair.Status.StorageStatus = storkv1.ClusterPairStatusReady
			clusterPair.Status.RemoteStorageID = remoteID
		}
		if clusterPair.Status.SchedulerStatus == storkv1.ClusterPairStatusInitial {
			// TODO: Verify we can talk to the scheduler on the other side
			clusterPair.Status.SchedulerStatus = storkv1.ClusterPairStatusReady
		}

		if clusterPair.Status.StorageStatus == storkv1.ClusterPairStatusReady &&
			clusterPair.Status.SchedulerStatus == storkv1.ClusterPairStatusReady {
			//clusterPair.Status.OverallStatus = stork_crd.ClusterPairStatusReady
		}
		err := sdk.Update(clusterPair)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ClusterPairController) createCRD() error {
	resource := k8s.CustomResource{
		Name:    storkv1.StorkClusterPairResourceName,
		Plural:  storkv1.StorkClusterPairResourcePlural,
		Group:   stork.GroupName,
		Version: stork.Version,
		Scope:   apiextensionsv1beta1.ClusterScoped,
		Kind:    reflect.TypeOf(storkv1.ClusterPair{}).Name(),
	}
	err := k8s.Instance().CreateCRD(resource)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return k8s.Instance().ValidateCRD(resource)
}
