package pkg

import (
	"github.com/pkg/errors"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/localz"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/projects"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// sharedVpcIam sets up IAM permissions as explained in
// https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#managing_firewall_resources
// create iam resources to allow the Google container engine service account in kube-cluster project to update
// firewall rules in shared project
func sharedVpcIam(ctx *pulumi.Context,
	locals *localz.Locals,
	createdClusterProject, createdNetworkProject *organizations.Project,
	createdSubNetwork *compute.Subnetwork) ([]pulumi.Resource, error) {

	_, err := projects.NewIAMCustomRole(
		ctx,
		"network-admin-role",
		&projects.IAMCustomRoleArgs{
			Description: pulumi.String("This role allows to administer network and security of the host project. " +
				"Intended for use by GKE automation on service projects."),
			Project: createdNetworkProject.ProjectId,
			Permissions: pulumi.StringArray{
				pulumi.String("compute.firewalls.create"),
				pulumi.String("compute.firewalls.delete"),
				pulumi.String("compute.firewalls.get"),
				pulumi.String("compute.firewalls.list"),
				pulumi.String("compute.firewalls.update"),
				pulumi.String("compute.networks.updatePolicy"),
			},
			RoleId: pulumi.String("network.admin"),
			Title:  pulumi.String("Host Project Network and Security Admin"),
		}, pulumi.Parent(createdSubNetwork))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create custom-iam role for network-admin-role on host project")
	}

	//   - serviceAccount:SERVICE_PROJECT_1_NUM@cloudservices.gserviceaccount.com
	//   - serviceAccount:service-SERVICE_PROJECT_1_NUM@container-engine-robot.iam.gserviceaccount.com
	//
	// https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-shared-vpc#enabling_and_granting_roles
	createdIamMemberSubnetCloudServices, err := compute.NewSubnetworkIAMMember(
		ctx,
		"subnetwork-iam-policy-cloudservices",
		&compute.SubnetworkIAMMemberArgs{
			Member: pulumi.Sprintf(
				"serviceAccount:%s@cloudservices.gserviceaccount.com",
				createdClusterProject.Number,
			),
			Project:    createdNetworkProject.ProjectId,
			Region:     pulumi.String(locals.GkeCluster.Spec.Region),
			Role:       pulumi.String("roles/compute.networkUser"),
			Subnetwork: createdSubNetwork.SelfLink,
		}, pulumi.Parent(createdSubNetwork))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add gke service accounts as iam members for subnetwork")
	}

	createdIamMemberSubnetContainerEngine, err := compute.NewSubnetworkIAMMember(
		ctx,
		"subnetwork-iam-policy-container-engine-robot",
		&compute.SubnetworkIAMMemberArgs{
			Member: pulumi.Sprintf(
				"serviceAccount:service-%s@container-engine-robot.iam.gserviceaccount.com",
				createdClusterProject.Number,
			),
			Project:    createdNetworkProject.ProjectId,
			Region:     pulumi.String(locals.GkeCluster.Spec.Region),
			Role:       pulumi.String("roles/compute.networkUser"),
			Subnetwork: createdSubNetwork.SelfLink,
		}, pulumi.Parent(createdSubNetwork))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add gke service accounts as iam members for subnetwork")
	}

	createdIamMemberContainerEngineServiceAgent, err := projects.NewIAMMember(ctx,
		"host-service-agent-role",
		&projects.IAMMemberArgs{
			Member: pulumi.Sprintf(
				"serviceAccount:service-%s@container-engine-robot.iam.gserviceaccount.com",
				createdClusterProject.Number,
			),
			Project: createdNetworkProject.ProjectId,
			Role:    pulumi.String("roles/container.hostServiceAgentUser"),
		}, pulumi.Parent(createdSubNetwork))
	if err != nil {
		return nil, errors.Wrap(err, "failed to add network host service agent role")
	}

	//bind network admin role to container engine robot service accounts that are auto created for each service project.
	createdNetworkAdminIamBinding, err := projects.NewIAMBinding(
		ctx,
		"network-admin",
		&projects.IAMBindingArgs{
			Members: pulumi.StringArray{
				pulumi.Sprintf(
					"serviceAccount:service-%s@container-engine-robot.iam.gserviceaccount.com",
					createdClusterProject.Number,
				),
			},
			Project: createdNetworkProject.ProjectId,
			Role: pulumi.Sprintf(
				"projects/%s/roles/network.admin",
				createdNetworkProject.ProjectId),
		}, pulumi.Parent(createdSubNetwork))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create role binding for network-admin role")
	}

	return []pulumi.Resource{
		createdIamMemberSubnetCloudServices,
		createdIamMemberSubnetContainerEngine,
		createdIamMemberContainerEngineServiceAgent,
		createdNetworkAdminIamBinding,
	}, nil
}
