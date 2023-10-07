package base

import (
	"buf.build/gen/go/plantoncloud/planton-cloud-apis/protocolbuffers/go/cloud/planton/apis/v1/code2cloud/deploy/kubecluster/stack/gcp"
	"github.com/pkg/errors"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/core/v1"
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"gopkg.in/yaml.v3"
)

type Input struct {
	IstioAddonBaseInput *gcp.AddonsIstioBase
	Namespace           *v1.Namespace
}

func Resources(ctx *pulumi.Context, input *Input) (*helm.Release, error) {
	release, err := addHelmRelease(ctx, input)
	if err != nil {
		return nil, errors.Wrap(err, "failed to add helm release")
	}
	return release, nil
}

func addHelmRelease(ctx *pulumi.Context, input *Input) (*helm.Release, error) {
	helmVal := getHelmVal()
	helmChart := getHelmChart()
	var helmValInput map[string]interface{}
	helmValBytes, err := yaml.Marshal(helmVal)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal helm val to bytes")
	}
	if err := yaml.Unmarshal(helmValBytes, &helmValInput); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal helm val")
	}
	release, err := helm.NewRelease(ctx, helmChart.ReleaseName, &helm.ReleaseArgs{
		Name:            pulumi.String(helmChart.ReleaseName),
		Namespace:       input.Namespace.Metadata.Name(),
		Chart:           pulumi.String(helmChart.Name),
		Version:         pulumi.String(helmChart.Version),
		CreateNamespace: pulumi.Bool(false),
		Atomic:          pulumi.Bool(true),
		CleanupOnFail:   pulumi.Bool(true),
		WaitForJobs:     pulumi.Bool(true),
		Timeout:         pulumi.Int(180), // 3 minutes
		Values:          pulumi.ToMap(helmValInput),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String(helmChart.Repo),
		},
	}, pulumi.Parent(input.Namespace),
		pulumi.IgnoreChanges([]string{"status", "description", "resourceNames"}),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to add %s helm release", helmChart.ReleaseName)
	}
	return release, nil
}
