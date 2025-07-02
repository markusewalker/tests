//go:build (validation || infra.rke2k3s || cluster.any || stress) && !infra.any && !infra.aks && !infra.eks && !infra.gke && !infra.rke1 && !sanity && !extended

package rke2k3s

import (
	"os"
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	extClusters "github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/extensions/defaults/stevetypes"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/config/operations"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/clusters"
	"github.com/rancher/tests/actions/config/defaults"
	"github.com/rancher/tests/actions/provisioninginput"
	"github.com/rancher/tests/validation/certificates"
	resources "github.com/rancher/tests/validation/provisioning/resources/provisioncluster"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CertRotationTestSuite struct {
	suite.Suite
	session            *session.Session
	client             *rancher.Client
	standardUserClient *rancher.Client
	cattleConfig       map[string]any
	rke2ClusterConfig  *clusters.ClusterConfig
	k3sClusterConfig   *clusters.ClusterConfig
	rke2ClusterID      string
	k3sClusterID       string
}

func (c *CertRotationTestSuite) TearDownSuite() {
	c.session.Cleanup()
}

func (c *CertRotationTestSuite) SetupSuite() {
	testSession := session.NewSession()
	c.session = testSession

	client, err := rancher.NewClient("", testSession)
	require.NoError(c.T(), err)

	c.client = client

	c.standardUserClient, err = standard.CreateStandardUser(c.client)
	require.NoError(c.T(), err)

	c.cattleConfig = config.LoadConfigFromFile(os.Getenv(config.ConfigEnvironmentKey))

	c.rke2ClusterConfig = new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, c.cattleConfig, c.rke2ClusterConfig)

	c.k3sClusterConfig = new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, c.cattleConfig, c.k3sClusterConfig)

	nodeRolesStandard := []provisioninginput.MachinePools{
		provisioninginput.EtcdMachinePool,
		provisioninginput.ControlPlaneMachinePool,
		provisioninginput.WorkerMachinePool,
	}

	nodeRolesStandard[0].MachinePoolConfig.Quantity = 3
	nodeRolesStandard[1].MachinePoolConfig.Quantity = 2
	nodeRolesStandard[2].MachinePoolConfig.Quantity = 3

	c.rke2ClusterConfig.MachinePools = nodeRolesStandard
	c.k3sClusterConfig.MachinePools = nodeRolesStandard

	c.rke2ClusterID, err = resources.ProvisionRKE2K3SCluster(c.T(), c.standardUserClient, extClusters.RKE2ClusterType.String(), c.rke2ClusterConfig, true, false)
	require.NoError(c.T(), err)

	c.k3sClusterID, err = resources.ProvisionRKE2K3SCluster(c.T(), c.standardUserClient, extClusters.K3SClusterType.String(), c.k3sClusterConfig, true, false)
	require.NoError(c.T(), err)

}

func (c *CertRotationTestSuite) TestCertRotation() {
	tests := []struct {
		name      string
		clusterID string
	}{
		{"RKE2 cert rotation", c.rke2ClusterID},
		{"K3S cert rotation", c.k3sClusterID},
	}

	for _, tt := range tests {
		cluster, err := c.client.Steve.SteveType(stevetypes.Provisioning).ByID(tt.clusterID)
		require.NoError(c.T(), err)

		c.Run(tt.name, func() {
			require.NoError(c.T(), certificates.RotateCerts(c.client, cluster.Name))
		})
	}
}

func TestCertRotationTestSuite(t *testing.T) {
	suite.Run(t, new(CertRotationTestSuite))
}
