package addons

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/plantoncloud-inc/go-commons/cloud/gcp/iam/roles/standard"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/localz"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/outputs"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/vars"
	externalsecretsv1 "github.com/plantoncloud/kubernetes-crd-pulumi-types/pkg/externalsecrets/externalsecrets/v1beta1"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/container"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/serviceaccount"
	pulumikubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	corev1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	metav1 "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/meta/v1"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func ExternalSecrets(ctx *pulumi.Context, locals *localz.Locals,
	createdCluster *container.Cluster, gcpProvider *gcp.Provider,
	kubernetesProvider *pulumikubernetes.Provider) error {

	//create google service account required to create workload identity binding
	createdGoogleServiceAccount, err := serviceaccount.NewAccount(ctx,
		vars.ExternalSecrets.KsaName,
		&serviceaccount.AccountArgs{
			Project:     createdCluster.Project,
			Description: pulumi.String("external-secrets service account for solving dns challenges to issue certificates"),
			AccountId:   pulumi.String(vars.ExternalSecrets.KsaName),
			DisplayName: pulumi.String(vars.ExternalSecrets.KsaName),
		}, pulumi.Parent(createdCluster), pulumi.Provider(gcpProvider))
	if err != nil {
		return errors.Wrap(err, "failed to create external-secrets google service account")
	}

	//export external-secrets gsa email
	ctx.Export(outputs.ExternalSecretsGsaEmail, createdGoogleServiceAccount.Email)

	//create workload-identity binding
	_, err = serviceaccount.NewIAMBinding(ctx,
		fmt.Sprintf("%s-workload-identity", vars.ExternalSecrets.KsaName),
		&serviceaccount.IAMBindingArgs{
			ServiceAccountId: createdGoogleServiceAccount.Name,
			Role:             pulumi.String(standard.Iam_workloadIdentityUser),
			Members: pulumi.StringArray{
				pulumi.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]",
					createdCluster.Project,
					vars.ExternalSecrets.Namespace,
					vars.ExternalSecrets.KsaName),
			},
		},
		pulumi.Parent(createdGoogleServiceAccount),
		pulumi.DependsOn([]pulumi.Resource{createdCluster}))
	if err != nil {
		return errors.Wrap(err, "failed to create workload-identity binding for external-secrets")
	}

	//create namespace resource
	createdNamespace, err := corev1.NewNamespace(ctx,
		vars.ExternalSecrets.Namespace,
		&corev1.NamespaceArgs{
			Metadata: metav1.ObjectMetaPtrInput(
				&metav1.ObjectMetaArgs{
					Name:   pulumi.String(vars.ExternalSecrets.Namespace),
					Labels: pulumi.ToStringMap(locals.KubernetesLabels),
				}),
		},
		pulumi.Provider(kubernetesProvider))
	if err != nil {
		return errors.Wrapf(err, "failed to create external-secrets namespace")
	}

	//create kubernetes service account to be used by the external-secrets.
	//it is not straight forward to add the gsa email as one of the helm values.
	// so, instead, disable service account creation in helm release and create it separately add
	// the Google workload identity annotation to the service account which requires the email id
	// of the Google service account added as part of IAM module.
	createdKubernetesServiceAccount, err := corev1.NewServiceAccount(ctx,
		vars.ExternalSecrets.KsaName,
		&corev1.ServiceAccountArgs{
			Metadata: metav1.ObjectMetaPtrInput(
				&metav1.ObjectMetaArgs{
					Name:      pulumi.String(vars.ExternalSecrets.KsaName),
					Namespace: createdNamespace.Metadata.Name(),
					Annotations: pulumi.StringMap{
						vars.WorkloadIdentityKubeAnnotationKey: createdGoogleServiceAccount.Email,
					},
				}),
		}, pulumi.Parent(createdNamespace))
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes service account")
	}

	//created helm-release
	_, err = helm.NewRelease(ctx, "external-secrets",
		&helm.ReleaseArgs{
			Name:            pulumi.String(vars.ExternalSecrets.HelmChartName),
			Namespace:       pulumi.String(vars.ExternalSecrets.Namespace),
			Chart:           pulumi.String(vars.ExternalSecrets.HelmChartName),
			Version:         pulumi.String(vars.ExternalSecrets.HelmChartVersion),
			CreateNamespace: pulumi.Bool(false),
			Atomic:          pulumi.Bool(false),
			CleanupOnFail:   pulumi.Bool(true),
			WaitForJobs:     pulumi.Bool(true),
			Timeout:         pulumi.Int(180), // 3 minutes
			Values: pulumi.Map{
				"customResourceManagerDisabled": pulumi.Sprintf("%t", false),
				"crds": pulumi.StringMap{
					"create": pulumi.Sprintf("%t", true),
				},
				"env": pulumi.StringMap{
					"POLLER_INTERVAL_MILLISECONDS": pulumi.Sprintf("%d",
						vars.ExternalSecrets.SecretsPollingIntervalSeconds*1000),
					"LOG_LEVEL":       pulumi.String("info"),
					"LOG_MESSAGE_KEY": pulumi.String("msg"),
					"METRICS_PORT":    pulumi.Sprintf("%d", 3001),
				},
				"rbac": pulumi.StringMap{
					"create": pulumi.Sprintf("%t", true),
				},
				"serviceAccount": pulumi.StringMap{
					"create":      pulumi.Sprintf("%t", false),
					"name":        pulumi.String(vars.ExternalSecrets.KsaName),
					"annotations": pulumi.String(""),
				},
				"replicaCount": pulumi.Sprintf("%d", 1),
			},
			RepositoryOpts: helm.RepositoryOptsArgs{
				Repo: pulumi.String(vars.ExternalSecrets.HelmChartRepo),
			},
		}, pulumi.Parent(createdNamespace),
		pulumi.DependsOn([]pulumi.Resource{createdKubernetesServiceAccount}),
		pulumi.IgnoreChanges([]string{"status", "description", "resourceNames"}))
	if err != nil {
		return errors.Wrap(err, "failed to create external-secrets helm release")
	}

	//create cluster-secret-store to configure the gcp project from which the secrets need to be looked up
	_, err = externalsecretsv1.NewClusterSecretStore(ctx, "cluster-secret-store",
		&externalsecretsv1.ClusterSecretStoreArgs{
			Metadata: metav1.ObjectMetaArgs{
				Name:   pulumi.String(vars.ExternalSecrets.GcpSecretsManagerClusterSecretStoreName),
				Labels: pulumi.ToStringMap(locals.KubernetesLabels),
			},
			Spec: externalsecretsv1.ClusterSecretStoreSpecArgs{
				Provider: externalsecretsv1.ClusterSecretStoreSpecProviderArgs{
					Gcpsm: externalsecretsv1.ClusterSecretStoreSpecProviderGcpsmArgs{
						ProjectID: createdCluster.Project,
					},
				},
				RefreshInterval: pulumi.Int(vars.ExternalSecrets.SecretsPollingIntervalSeconds),
			},
		})
	if err != nil {
		return errors.Wrap(err, "failed to create cluster-secret-store")
	}

	return nil
}
