package cluster

import (
	"github.com/pkg/errors"
	"github.com/plantoncloud-inc/kube-cluster-pulumi-blueprint/pkg/aws/network"
	kubeclusterv1 "github.com/plantoncloud/planton-cloud-apis/zzgo/cloud/planton/apis/v1/code2cloud/deploy/kubecluster"
	awsclassic "github.com/pulumi/pulumi-aws/sdk/v6/go/aws"
	"github.com/pulumi/pulumi-eks/sdk/go/eks"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Input struct {
	KubeCluster           *kubeclusterv1.KubeCluster
	Labels                map[string]string
	AddedNetworkResources *network.AddedResources
	AwsProvider           *awsclassic.Provider
}

func Resources(ctx *pulumi.Context, input *Input) error {
	// Create an EKS cluster.
	addedCluster, err := eks.NewCluster(ctx, "my-cluster", &eks.ClusterArgs{
		VpcId:            input.AddedNetworkResources.AddedVpc.VpcId,
		PublicSubnetIds:  input.AddedNetworkResources.AddedVpc.PublicSubnetIds,
		PrivateSubnetIds: input.AddedNetworkResources.AddedVpc.PrivateSubnetIds,
		InstanceType:     pulumi.StringPtr("t2.medium"),
		DesiredCapacity:  pulumi.IntPtr(1),
		MinSize:          pulumi.IntPtr(1),
		MaxSize:          pulumi.IntPtr(2),
		StorageClasses:   "gp2",
	}, pulumi.Provider(input.AwsProvider))

	if err != nil {
		return errors.Wrap(err, "failed to add eks cluster")
	}
	ctx.Export("kubeconfig", addedCluster.Kubeconfig)
	return nil
}
