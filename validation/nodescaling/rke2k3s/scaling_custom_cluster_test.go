//go:build (validation || infra.rke2k3s || cluster.custom || stress) && !infra.any && !infra.aks && !infra.eks && !infra.gke && !infra.rke1 && !cluster.any && !cluster.nodedriver && !sanity && !extended

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
	"github.com/rancher/tests/actions/machinepools"
	"github.com/rancher/tests/actions/provisioninginput"
	"github.com/rancher/tests/actions/scalinginput"
	"github.com/rancher/tests/validation/nodescaling"
	resources "github.com/rancher/tests/validation/provisioning/resources/provisioncluster"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CustomClusterNodeScalingTestSuite struct {
	suite.Suite
	client             *rancher.Client
	standardUserClient *rancher.Client
	session            *session.Session
	scalingConfig      *scalinginput.Config
	cattleConfig       map[string]any
	rke2ClusterConfig  *clusters.ClusterConfig
	k3sClusterConfig   *clusters.ClusterConfig
	rke2ClusterID      string
	k3sClusterID       string
}

func (s *CustomClusterNodeScalingTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *CustomClusterNodeScalingTestSuite) SetupSuite() {
	testSession := session.NewSession()
	s.session = testSession

	client, err := rancher.NewClient("", s.session)
	require.NoError(s.T(), err)

	s.client = client

	s.standardUserClient, err = standard.CreateStandardUser(s.client)
	require.NoError(s.T(), err)

	s.cattleConfig = config.LoadConfigFromFile(os.Getenv(config.ConfigEnvironmentKey))

	s.rke2ClusterConfig = new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, s.cattleConfig, s.rke2ClusterConfig)

	s.k3sClusterConfig = new(clusters.ClusterConfig)
	operations.LoadObjectFromMap(defaults.ClusterConfigKey, s.cattleConfig, s.k3sClusterConfig)

	s.scalingConfig = new(scalinginput.Config)
	config.LoadConfig(scalinginput.ConfigurationFileKey, s.scalingConfig)

	nodeRolesStandard := []provisioninginput.MachinePools{
		provisioninginput.EtcdMachinePool,
		provisioninginput.ControlPlaneMachinePool,
		provisioninginput.WorkerMachinePool,
	}

	nodeRolesStandard[0].MachinePoolConfig.Quantity = 3
	nodeRolesStandard[1].MachinePoolConfig.Quantity = 2
	nodeRolesStandard[2].MachinePoolConfig.Quantity = 3

	s.rke2ClusterConfig.MachinePools = nodeRolesStandard
	s.k3sClusterConfig.MachinePools = nodeRolesStandard

	s.rke2ClusterID, err = resources.ProvisionRKE2K3SCluster(s.T(), s.standardUserClient, extClusters.RKE2ClusterType.String(), s.rke2ClusterConfig, true, true)
	require.NoError(s.T(), err)

	s.k3sClusterID, err = resources.ProvisionRKE2K3SCluster(s.T(), s.standardUserClient, extClusters.K3SClusterType.String(), s.k3sClusterConfig, true, true)
	require.NoError(s.T(), err)
}

func (s *CustomClusterNodeScalingTestSuite) TestScalingCustomClusterNodes() {
	nodeRolesEtcd := machinepools.NodeRoles{
		Etcd:     true,
		Quantity: 1,
	}

	nodeRolesControlPlane := machinepools.NodeRoles{
		ControlPlane: true,
		Quantity:     1,
	}

	nodeRolesEtcdControlPlane := machinepools.NodeRoles{
		Etcd:         true,
		ControlPlane: true,
		Quantity:     1,
	}

	nodeRolesWorker := machinepools.NodeRoles{
		Worker:   true,
		Quantity: 1,
	}

	nodeRolesWindows := machinepools.NodeRoles{
		Windows:  true,
		Quantity: 1,
	}

	tests := []struct {
		name      string
		nodeRoles machinepools.NodeRoles
		clusterID string
	}{
		{"RKE2 control plane by 1", nodeRolesControlPlane, s.rke2ClusterID},
		{"RKE2 etcd by 1", nodeRolesEtcd, s.rke2ClusterID},
		{"RKE2 etcd and control plane by 1", nodeRolesEtcdControlPlane, s.rke2ClusterID},
		{"RKE2 worker by 1", nodeRolesWorker, s.rke2ClusterID},
		{"RKE2 windows by 1", nodeRolesWindows, s.rke2ClusterID},
		{"K3S control plane by 1", nodeRolesControlPlane, s.k3sClusterID},
		{"K3S etcd by 1", nodeRolesEtcd, s.k3sClusterID},
		{"K3S etcd and control plane by 1", nodeRolesEtcdControlPlane, s.k3sClusterID},
		{"K3S worker by 1", nodeRolesWorker, s.k3sClusterID},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			nodescaling.ScalingRKE2K3SCustomClusterPools(s.T(), s.client, tt.clusterID, s.scalingConfig.NodeProvider, tt.nodeRoles)
		})
	}
}

func (s *CustomClusterNodeScalingTestSuite) TestScalingCustomClusterNodesDynamicInput() {
	if s.scalingConfig.MachinePools.NodeRoles == nil {
		s.T().Skip()
	}

	clusterID, err := extClusters.GetV1ProvisioningClusterByName(s.client, s.client.RancherConfig.ClusterName)
	require.NoError(s.T(), err)

	nodescaling.ScalingRKE2K3SCustomClusterPools(s.T(), s.client, clusterID, s.scalingConfig.NodeProvider, *s.scalingConfig.MachinePools.NodeRoles)
}

func TestCustomClusterNodeScalingTestSuite(t *testing.T) {
	suite.Run(t, new(CustomClusterNodeScalingTestSuite))
}
