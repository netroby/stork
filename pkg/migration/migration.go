package migration

import (
	"fmt"

	"github.com/libopenstorage/stork/drivers/volume"
	"github.com/libopenstorage/stork/pkg/migration/controllers"
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
	return nil
}
