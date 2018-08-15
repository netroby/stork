package controllers

import (
	"context"
	"fmt"
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
	return nil
}

// Handle updates for Migration objects
func (m *MigrationController) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *stork_crd.Migration:
		logrus.Debugf("Update for migration %v Deleted: %v", o, event.Deleted)
		migration := o
		if event.Deleted {
			return m.Driver.CancelMigration(migration)
		}

		if migration.Spec.ClusterPair == "" {
			return fmt.Errorf("clusterPair to migrate to cannot be empty")
		}

		if migration.Status.Stage == "" {
			migration.Status = stork_crd.MigrationStatus{
				Stage:  stork_crd.MigrationStageInitializing,
				Status: stork_crd.MigrationStatusPending,
			}
		}
		if migration.Status.Stage == stork_crd.MigrationStageInitializing ||
			migration.Status.Stage == stork_crd.MigrationStageVolumes {
			migration.Status.Stage = stork_crd.MigrationStageVolumes
			// Trigger the migration if we don't have any status
			if migration.Status.Volumes == nil {
				volumeInfos, err := m.Driver.StartMigration(migration)
				if err != nil {
					return err
				}
				if volumeInfos == nil {
					volumeInfos = make([]*stork_crd.VolumeInfo, 0)
				}
				migration.Status.Volumes = volumeInfos
				err = sdk.Update(migration)
				if err != nil {
					return err
				}
			}

			// Now check the status
			volumeInfos, err := m.Driver.GetMigrationStatus(migration)
			if err != nil {
				return err
			}
			if volumeInfos == nil {
				volumeInfos = make([]*stork_crd.VolumeInfo, 0)
			}
			migration.Status.Volumes = volumeInfos
			err = sdk.Update(migration)
			if err != nil {
				return err
			}

			// Now check if there is any failure or success
			// TODO: On failure of one volume cancel other migrations
			for _, vInfo := range volumeInfos {
				if vInfo.Status == stork_crd.MigrationStatusInProgress {
					return fmt.Errorf("Migration still in progress")
				} else if vInfo.Status == stork_crd.MigrationStatusFailed {
					migration.Status.Stage = stork_crd.MigrationStageFinal
					migration.Status.Status = stork_crd.MigrationStatusFailed
				}
			}

			if migration.Status.Status != stork_crd.MigrationStatusFailed {
				migration.Status.Stage = stork_crd.MigrationStageFinal
				migration.Status.Status = stork_crd.MigrationStatusSuccessful
			}
			err = sdk.Update(migration)
			if err != nil {
				return err
			}
		}
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
