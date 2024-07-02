package addon

import (
	"github.com/pkg/errors"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/certmanager"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/externaldns"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/externalsecrets"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/ingressnginx"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/istio"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/linkerd"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/opencost"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/plantoncloudkubeagent"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/postgresoperator"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/prometheus"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/reflector"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/solroperator"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/strimzi"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/addon/traefik"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/container/cluster"
	"github.com/plantoncloud/kube-cluster-pulumi-blueprint/pkg/gcp/iam"
	code2cloudv1deployk8cmodel "github.com/plantoncloud/planton-cloud-apis/zzgo/cloud/planton/apis/code2cloud/v1/kubecluster/model"
	c2cv1deployk8cstackgcpmodel "github.com/plantoncloud/planton-cloud-apis/zzgo/cloud/planton/apis/code2cloud/v1/kubecluster/stack/gcp/model"
	"github.com/plantoncloud/pulumi-blueprint-golang-commons/pkg/google/gke/pulumigkekubernetesprovider"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/organizations"
	pulumikubernetes "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type Input struct {
	ContainerAddonInput            *c2cv1deployk8cstackgcpmodel.Addons
	WorkspaceDir                   string
	AddedContainerClusterResources *cluster.AddedContainerClusterResources
	AddedIamResources              *iam.AddedIamResources
	AddedContainerClusterProject   *organizations.Project
	KubeClusterAddons              *code2cloudv1deployk8cmodel.KubeClusterAddonsSpec
}

type AddedResources struct {
	IstioAddedResources *istio.AddedResources
}

func Resources(ctx *pulumi.Context, input *Input) (*AddedResources, error) {
	kubernetesProvider, err := pulumigkekubernetesprovider.GetWithAddedClusterWithGsaKey(ctx, input.AddedIamResources.WorkloadDeployerGsaKey,
		input.AddedContainerClusterResources.Cluster, input.AddedContainerClusterResources.NodePools)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup kubernetes provider")
	}

	addonAddedResources, err := clusterAddonResources(ctx, kubernetesProvider, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add kube-cluster addon resources")
	}
	return addonAddedResources, nil
}

func clusterAddonResources(ctx *pulumi.Context, kubernetesProvider *pulumikubernetes.Provider, input *Input) (*AddedResources, error) {
	workspace := input.WorkspaceDir
	addonsInput := input.ContainerAddonInput
	istioAddedResources, err := istio.Resources(ctx, &istio.Input{
		Workspace:          workspace,
		KubernetesProvider: kubernetesProvider,
		IstioAddonInput:    addonsInput.Istio,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to add istio resources")
	}
	if err := certmanager.Resources(ctx, &certmanager.Input{
		Workspace:             workspace,
		KubernetesProvider:    kubernetesProvider,
		CertManagerAddonInput: addonsInput.CertManager,
		AddedCertManagerGsa:   input.AddedIamResources.CertManagerGsa,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add cert-manager resources")
	}
	if err := externalsecrets.Resources(ctx, &externalsecrets.Input{
		Workspace:                    workspace,
		KubernetesProvider:           kubernetesProvider,
		ExternalSecretsAddonInput:    addonsInput.ExternalSecrets,
		AddedExternalSecretsGsa:      input.AddedIamResources.ExternalSecretsGsa,
		AddedContainerClusterProject: input.AddedContainerClusterProject,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add external-secrets addon")
	}

	if err := externaldns.Resources(ctx, &externaldns.Input{
		Workspace:             workspace,
		KubernetesProvider:    kubernetesProvider,
		ExternalDnsAddonInput: addonsInput.ExternalDns,
		AddedExternalDnsGsa:   input.AddedIamResources.ExternalDnsGsa,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add external-dns resources")
	}

	if err := strimzi.Resources(ctx, &strimzi.Input{
		KubernetesProvider: kubernetesProvider,
		StrimziAddonInput:  addonsInput.Strimzi,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add external-secrets addon")
	}

	if err := postgresoperator.Resources(ctx, &postgresoperator.Input{
		KubernetesProvider:         kubernetesProvider,
		PostgresOperatorAddonInput: addonsInput.PostgresOperator,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add postgres-operator resources")
	}

	if err := ingressnginx.Resources(ctx, &ingressnginx.Input{
		Workspace:              workspace,
		KubernetesProvider:     kubernetesProvider,
		IngressNginxAddonInput: addonsInput.IngressNginx,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add ingress-nginx addon")
	}

	if err := traefik.Resources(ctx, &traefik.Input{
		KubernetesProvider: kubernetesProvider,
		TraefikAddonInput:  addonsInput.Traefik,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add traefik resources")
	}

	if err := linkerd.Resources(ctx, &linkerd.Input{
		KubernetesProvider: kubernetesProvider,
		LinkerdAddonInput:  addonsInput.Linkerd,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add linkerd resources")
	}

	if err := reflector.Resources(ctx, &reflector.Input{
		KubernetesProvider:  kubernetesProvider,
		ReflectorAddonInput: addonsInput.Reflector,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add reflector addon")
	}

	if err := prometheus.Resources(ctx, &prometheus.Input{
		KubernetesProvider: kubernetesProvider,
		OpenCostAddonInput: addonsInput.OpenCost,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add prometheus addon")
	}

	if err := opencost.Resources(ctx, &opencost.Input{
		KubernetesProvider: kubernetesProvider,
		OpenCostAddonInput: addonsInput.OpenCost,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add open-cost addon")
	}

	if err := plantoncloudkubeagent.Resources(ctx, &plantoncloudkubeagent.Input{
		KubernetesProvider:              kubernetesProvider,
		PlantonCloudKubeAgentAddonInput: addonsInput.PlantonCloudKubeAgent,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add planton-cloud-kube-agent addon")
	}

	//todo: upgrade to allow kubernetes upgrades
	//if err := kubecost.Resources(ctx, &kubecost.Input{
	//	KubernetesProvider: kubernetesProvider,
	//	KubeCostAddonInput: addonsInput.KubeCost,
	//}); err != nil {
	//	return nil, errors.Wrap(err, "failed to add kube-cost addon")
	//}

	if err := solroperator.Resources(ctx, &solroperator.Input{
		KubernetesProvider:     kubernetesProvider,
		SolrOperatorAddonInput: addonsInput.SolrOperator,
	}); err != nil {
		return nil, errors.Wrap(err, "failed to add solr-operator addon")
	}

	return &AddedResources{
		IstioAddedResources: istioAddedResources,
	}, nil
}
