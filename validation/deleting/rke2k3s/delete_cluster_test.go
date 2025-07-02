//go:build (infra.rke2k3s || validation) && !infra.any && !infra.aks && !infra.eks && !infra.gke && !infra.rke1 && !stress && !sanity && !extended

package rke2k3s

import (
	"os"
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	extClusters "github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/config/defaults"
	"github.com/rancher/tests/actions/provisioning"
	"github.com/rancher/tests/actions/provisioninginput"
	resources "github.com/rancher/tests/validation/provisioning/resources/provisioncluster"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DeleteClusterTestSuite struct {
	suite.Suite
	client             *rancher.Client
	standardUserClient *rancher.Client
	session            *session.Session
	cattleConfig       map[string]any
	rke2ClusterConfig  *clusters.ClusterConfig
	k3sClusterConfig   *clusters.ClusterConfig
	rke2ClusterID      string
	k3sClusterID       string
}

func (d *DeleteClusterTestSuite) TearDownSuite() {
	d.session.Cleanup()
}

func (d *DeleteClusterTestSuite) SetupSuite() {
	testSession := session.NewSession()
	d.session = testSession

	client, err := rancher.NewClient("", d.session)
	require.NoError(d.T(), err)

	d.client = client

	d.standardUserClient, err = standard.CreateStandardUser(d.client)
	require.NoError(d.T(), err)

	d.cattleConfig = config.LoadConfigFromFile(os.Getenv(config.ConfigEnvironmentKey))

	d.rke2ClusterConfig = new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, d.cattleConfig, d.rke2ClusterConfig)

	d.k3sClusterConfig = new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, d.cattleConfig, d.k3sClusterConfig)

	nodeRolesStandard := []provisioninginput.MachinePools{
		provisioninginput.EtcdMachinePool,
		provisioninginput.ControlPlaneMachinePool,
		provisioninginput.WorkerMachinePool,
	}

	nodeRolesStandard[0].MachinePoolConfig.Quantity = 3
	nodeRolesStandard[1].MachinePoolConfig.Quantity = 2
	nodeRolesStandard[2].MachinePoolConfig.Quantity = 3

	d.rke2ClusterConfig.MachinePools = nodeRolesStandard
	d.k3sClusterConfig.MachinePools = nodeRolesStandard

	d.rke2ClusterID, err = resources.ProvisionRKE2K3SCluster(d.T(), d.standardUserClient, extClusters.RKE2ClusterType.String(), d.rke2ClusterConfig, true, false)
	require.NoError(d.T(), err)

	d.k3sClusterID, err = resources.ProvisionRKE2K3SCluster(d.T(), d.standardUserClient, extClusters.K3SClusterType.String(), d.k3sClusterConfig, true, false)
	require.NoError(d.T(), err)
}

func (d *DeleteClusterTestSuite) TestDeletingCluster() {
	tests := []struct {
		name      string
		clusterID string
	}{
		{"Deleting RKE2 cluster", d.rke2ClusterID},
		{"Deleting K3S cluster", d.k3sClusterID},
	}

	for _, tt := range tests {
		d.Run(tt.name, func() {
			extClusters.DeleteK3SRKE2Cluster(d.client, tt.clusterID)
			provisioning.VerifyDeleteRKE2K3SCluster(d.T(), d.client, tt.clusterID)
		})
	}
}

func TestDeleteClusterTestSuite(t *testing.T) {
	suite.Run(t, new(DeleteClusterTestSuite))
}
