package nodepool

import (
	kubernetesclusterv1state "buf.build/gen/go/plantoncloud/planton-cloud-apis/protocolbuffers/go/cloud/planton/apis/v1/code2cloud/deploy/kubecluster/state"
	wordpb "buf.build/gen/go/plantoncloud/planton-cloud-apis/protocolbuffers/go/cloud/planton/apis/v1/commons/english/rpc/enums"
	"github.com/pkg/errors"
	"github.com/plantoncloud-inc/kube-cluster-pulumi-blueprint/pkg/gcp/container/cluster/nodepool/tag"
	puluminameoutputgcp "github.com/plantoncloud-inc/pulumi-stack-runner-go-sdk/pkg/name/provider/cloud/gcp/output"
	"github.com/pulumi/pulumi-gcp/sdk/v6/go/gcp/container"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

const (
	autoRepair  = true
	autoUpgrade = true
)

type Input struct {
	KubeClusterId string
	GcpZone       string
	Cluster       *container.Cluster
	NodePools     []*kubernetesclusterv1state.KubeClusterNodePoolGcpState
	Labels        map[string]string
}

func Resources(ctx *pulumi.Context, input *Input) ([]pulumi.Resource, error) {
	addedNodePools := make([]pulumi.Resource, 0)
	for _, np := range input.NodePools {
		addedNodePool, err := addNodePool(ctx, input, np)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add %s node-pool", np.Name)
		}
		addedNodePools = append(addedNodePools, addedNodePool)
	}
	return addedNodePools, nil
}

func addNodePool(ctx *pulumi.Context, input *Input, clusterNodePoolInput *kubernetesclusterv1state.KubeClusterNodePoolGcpState) (*container.NodePool, error) {
	addedNodePool, err := container.NewNodePool(ctx, clusterNodePoolInput.Name, &container.NodePoolArgs{
		Location:  pulumi.String(input.GcpZone),
		Project:   input.Cluster.Project,
		Cluster:   input.Cluster.Name,
		NodeCount: pulumi.Int(clusterNodePoolInput.MinNodeCount),
		Autoscaling: container.NodePoolAutoscalingPtrInput(&container.NodePoolAutoscalingArgs{
			MinNodeCount: pulumi.Int(clusterNodePoolInput.MinNodeCount),
			MaxNodeCount: pulumi.Int(clusterNodePoolInput.MaxNodeCount),
		}),
		Management: container.NodePoolManagementPtrInput(&container.NodePoolManagementArgs{
			AutoRepair:  pulumi.Bool(autoRepair),
			AutoUpgrade: pulumi.Bool(autoUpgrade),
		}),
		NodeConfig: &container.NodePoolNodeConfigArgs{
			Labels:      pulumi.ToStringMap(input.Labels),
			MachineType: pulumi.String(clusterNodePoolInput.MachineType),
			Metadata:    pulumi.StringMap{"disable-legacy-endpoints": pulumi.String("true")},
			OauthScopes: getOauthScopes(),
			Preemptible: pulumi.Bool(clusterNodePoolInput.IsSpotEnabled),
			Tags:        pulumi.StringArray{pulumi.String(tag.Get(input.KubeClusterId))},
			WorkloadMetadataConfig: container.NodePoolNodeConfigWorkloadMetadataConfigPtrInput(
				&container.NodePoolNodeConfigWorkloadMetadataConfigArgs{
					Mode: pulumi.String("GKE_METADATA")}),
		},
		UpgradeSettings: container.NodePoolUpgradeSettingsPtrInput(&container.NodePoolUpgradeSettingsArgs{
			MaxSurge:       pulumi.Int(2),
			MaxUnavailable: pulumi.Int(1),
		}),
	},
		pulumi.Parent(input.Cluster),
		pulumi.IgnoreChanges([]string{"nodeCount"}),
		pulumi.DeleteBeforeReplace(true),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add node pool")
	}
	ctx.Export(GetNodePoolNameOutputName(clusterNodePoolInput.Name), addedNodePool.Name)
	ctx.Export(GetNodePoolMachineTypeOutputName(clusterNodePoolInput.Name), addedNodePool.NodeConfig.MachineType())
	ctx.Export(GetNodePoolIsSpotInstancesOutputName(clusterNodePoolInput.Name), addedNodePool.NodeConfig.Preemptible())
	return addedNodePool, nil
}
func getOauthScopes() pulumi.StringArrayInput {
	scopes := pulumi.StringArray{
		pulumi.String("https://www.googleapis.com/auth/monitoring"),
		pulumi.String("https://www.googleapis.com/auth/monitoring.write"),
		pulumi.String("https://www.googleapis.com/auth/devstorage.read_only"),
		pulumi.String("https://www.googleapis.com/auth/logging.write"),
	}
	return scopes
}

func GetNodePoolNameOutputName(nodePoolName string) string {
	return puluminameoutputgcp.Name(container.NodePool{}, nodePoolName, wordpb.Word_name.String())
}

func GetNodePoolMachineTypeOutputName(nodePoolName string) string {
	return puluminameoutputgcp.Name(container.NodePool{}, nodePoolName, wordpb.Word_machine.String(), wordpb.Word_type.String())
}

func GetNodePoolIsSpotInstancesOutputName(nodePoolName string) string {
	return puluminameoutputgcp.Name(container.NodePool{}, nodePoolName, wordpb.Word_spot.String(), wordpb.Word_instances.String())
}
