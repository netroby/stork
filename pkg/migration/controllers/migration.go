package controllers

import (
	"context"
	"reflect"
	"time"

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

const (
	resyncPeriod = int(60 * time.Minute)
)

// MigrationController migrationcontroller
type MigrationController struct {
	Driver volume.Driver
}

// Init init
func (m *MigrationController) Init(config *rest.Config, client apiextensionsclient.Interface) error {
	err := m.createCRD(client)
	if err != nil {
		return err
	}

	sdk.Watch(stork_crd.SchemeGroupVersion.String(), reflect.TypeOf(stork_crd.Migration{}).Name(), "", resyncPeriod)

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

// Handle updates for Migration objects
func (m *MigrationController) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *stork_crd.Migration:

		logrus.Infof("Update for migration %v", o)
	}
	return nil
}

func (m *MigrationController) createCRD(client apiextensionsclient.Interface) error {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: stork_crd.StorkMigrationResourcePlural + "." + stork.GroupName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   stork.GroupName,
			Version: stork.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: stork_crd.StorkMigrationResourcePlural,
				Kind:   reflect.TypeOf(stork_crd.Migration{}).Name(),
			},
		},
	}
	_, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)

	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}
