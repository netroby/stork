package migration

import (
	"context"
	"fmt"

	"github.com/libopenstorage/stork/drivers/volume"
	stork_crd "github.com/libopenstorage/stork/pkg/apis/stork/v1alpha1"
	"github.com/libopenstorage/stork/pkg/migration/controllers"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"
)

// Migration migration
type Migration struct {
	Driver                volume.Driver
	clusterPairController *controllers.ClusterPairController
	migrationController   *controllers.MigrationController
}

// Init init
func (m *Migration) Init(config *rest.Config, client apiextensionsclient.Interface) error {
	m.clusterPairController = &controllers.ClusterPairController{
		Driver: m.Driver}
	err := m.clusterPairController.Init(config, client)
	if err != nil {
		return fmt.Errorf("error initiliazling clusterpair controller: %v", err)
	}

	m.migrationController = &controllers.MigrationController{
		Driver: m.Driver}
	err = m.migrationController.Init(config, client)
	if err != nil {
		return fmt.Errorf("error initiliazling clusterpair controller: %v", err)
	}
	sdk.Handle(m)
	go sdk.Run(context.TODO())
	return nil
}

// Handle handles updated for registered types
func (m *Migration) Handle(ctx context.Context, event sdk.Event) error {
	switch event.Object.(type) {
	case *stork_crd.ClusterPair:
		return m.clusterPairController.Handle(ctx, event)
	case *stork_crd.Migration:
		return m.migrationController.Handle(ctx, event)
	}
	return nil
}
