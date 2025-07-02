//go:build (validation || infra.rke1 || cluster.nodedriver || extended) && !infra.any && !infra.aks && !infra.eks && !infra.gke && !infra.rke2k3s && !cluster.any && !cluster.custom && !sanity && !stress

package rke1

import (
	"testing"

	"github.com/rancher/shepherd/clients/rancher"
	"github.com/rancher/shepherd/extensions/clusters"
	"github.com/rancher/shepherd/pkg/config"
	"github.com/rancher/shepherd/pkg/session"
	"github.com/rancher/tests/actions/provisioninginput"
	nodepools "github.com/rancher/tests/actions/rke1/nodepools"
	"github.com/rancher/tests/actions/scalinginput"
	"github.com/rancher/tests/validation/nodescaling"
	resources "github.com/rancher/tests/validation/provisioning/resources/provisioncluster"
	standard "github.com/rancher/tests/validation/provisioning/resources/standarduser"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RKE1NodeScalingTestSuite struct {
	suite.Suite
	session            *session.Session
	client             *rancher.Client
	standardUserClient *rancher.Client
	scalingConfig      *scalinginput.Config
	provisioningConfig *provisioninginput.Config
	rke1ClusterID      string
}

func (s *RKE1NodeScalingTestSuite) TearDownSuite() {
	s.session.Cleanup()
}

func (s *RKE1NodeScalingTestSuite) SetupSuite() {
	testSession := session.NewSession()
	s.session = testSession

	s.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, s.provisioningConfig)

	s.scalingConfig = new(scalinginput.Config)
	config.LoadConfig(scalinginput.ConfigurationFileKey, s.scalingConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(s.T(), err)

	s.client = client

	s.standardUserClient, err = standard.CreateStandardUser(s.client)
	require.NoError(s.T(), err)

	nodeRolesStandard := []provisioninginput.NodePools{
		provisioninginput.EtcdNodePool,
		provisioninginput.ControlPlaneNodePool,
		provisioninginput.WorkerNodePool,
	}

	nodeRolesStandard[0].NodeRoles.Quantity = 3
	nodeRolesStandard[1].NodeRoles.Quantity = 2
	nodeRolesStandard[2].NodeRoles.Quantity = 3

	s.provisioningConfig.NodePools = nodeRolesStandard

	s.rke1ClusterID, err = resources.ProvisionRKE1Cluster(s.T(), s.standardUserClient, s.provisioningConfig, true, false)
	require.NoError(s.T(), err)
}

func (s *RKE1NodeScalingTestSuite) TestScalingRKE1NodePools() {
	nodeRolesEtcd := nodepools.NodeRoles{
		Etcd:     true,
		Quantity: 1,
	}

	nodeRolesControlPlane := nodepools.NodeRoles{
		ControlPlane: true,
		Quantity:     1,
	}

	nodeRolesWorker := nodepools.NodeRoles{
		Worker:   true,
		Quantity: 1,
	}

	tests := []struct {
		name      string
		nodeRoles nodepools.NodeRoles
		clusterID string
	}{
		{"Scaling RKE1 control plane by 1", nodeRolesControlPlane, s.rke1ClusterID},
		{"Scaling RKE1 etcd node by 1", nodeRolesEtcd, s.rke1ClusterID},
		{"Scaling RKE1 worker by 1", nodeRolesWorker, s.rke1ClusterID},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			nodescaling.ScalingRKE1NodePools(s.T(), s.client, tt.clusterID, tt.nodeRoles)
		})
	}
}

func (s *RKE1NodeScalingTestSuite) TestScalingRKE1NodePoolsDynamicInput() {
	if s.scalingConfig.NodePools.NodeRoles == nil {
		s.T().Skip()
	}

	clusterID, err := clusters.GetClusterIDByName(s.client, s.client.RancherConfig.ClusterName)
	require.NoError(s.T(), err)

	nodescaling.ScalingRKE1NodePools(s.T(), s.client, clusterID, *s.scalingConfig.NodePools.NodeRoles)
}

func TestRKE1NodeScalingTestSuite(t *testing.T) {
	suite.Run(t, new(RKE1NodeScalingTestSuite))
}
