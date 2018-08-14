package controllers

import (
	"context"
	"reflect"

	"github.com/libopenstorage/stork/drivers/volume"
	stork "github.com/libopenstorage/stork/pkg/apis/stork"
	stork_crd "github.com/libopenstorage/stork/pkg/apis/stork/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

//ClusterPairController pair
type ClusterPairController struct {
	Driver volume.Driver
}

//Init init
func (c *ClusterPairController) Init(config *rest.Config, client apiextensionsclient.Interface) error {
	err := c.createCRD(client)
	if err != nil {
		return err
	}

	sdk.Watch(stork_crd.SchemeGroupVersion.String(), reflect.TypeOf(stork_crd.ClusterPair{}).Name(), "", resyncPeriod)

	/*
		storkClient, err := stork_client.NewForConfig(config)
		remotePair, err := storkClient.StorkV1alpha1().ClusterPairs("default").Get("localcluster", metav1.GetOptions{})
		if err != nil {
			logrus.Errorf("Error getting pair: %v", err)
			return nil
		}
			remoteClientConfig := clientcmd.NewNonInteractiveClientConfig(remotePair.Config, remotePair.Config.CurrentContext, &clientcmd.ConfigOverrides{}, nil)
			remoteConfig, err := remoteClientConfig.ClientConfig()
			if err != nil {
				return err
			}

			remoteK8sClient, err := clientset.NewForConfig(remoteConfig)
			if err != nil {
				logrus.Fatalf("Error getting client, %v", err)
			}
			podList, err := remoteK8sClient.CoreV1().Pods("").List(metav1.ListOptions{})
			for _, p := range podList.Items {
				log.PodLog(&p).Infof("listing pods")
			}*/
	return nil
}

// Handle updates for ClusterPair objects
func (c *ClusterPairController) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *stork_crd.ClusterPair:

		clusterPair := o
		if event.Deleted {
			if clusterPair.Status.RemoteStorageID != "" {
				return c.Driver.DeletePair(clusterPair)
			}
		}

		if clusterPair.Status.StorageStatus == "" {
			logrus.Infof("New cluster pair created %v", clusterPair.Name)
			remoteID, err := c.Driver.CreatePair(clusterPair)
			if err != nil {
				return err
			}
			clusterPair.Status.StorageStatus = stork_crd.ClusterPairStatusReady
			clusterPair.Status.RemoteStorageID = remoteID
		}
		if clusterPair.Status.SchedulerStatus == "" {
			// TODO: Verify we can talk to the scheduler on the other side
			clusterPair.Status.SchedulerStatus = stork_crd.ClusterPairStatusReady
		}

		if clusterPair.Status.StorageStatus == stork_crd.ClusterPairStatusReady &&
			clusterPair.Status.SchedulerStatus == stork_crd.ClusterPairStatusReady {
			//clusterPair.Status.OverallStatus = stork_crd.ClusterPairStatusReady
		}
		err := sdk.Update(clusterPair)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *ClusterPairController) createCRD(client apiextensionsclient.Interface) error {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: stork_crd.StorkClusterPairResourcePlural + "." + stork.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   stork.GroupName,
			Version: stork.Version,
			Scope:   apiextensionsv1beta1.ClusterScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: stork_crd.StorkClusterPairResourcePlural,
				Kind:   reflect.TypeOf(stork_crd.ClusterPair{}).Name(),
			},
		},
	}
	_, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)

	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
