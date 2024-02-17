package clusterissuer

import (
	"fmt"
	v1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/pkg/errors"
	"github.com/plantoncloud-inc/go-commons/kubernetes/manifest"
	c2cv1deployk8cstackgcpmodel "github.com/plantoncloud/planton-cloud-apis/zzgo/cloud/planton/apis/v1/code2cloud/deploy/kubecluster/stack/gcp/model"
	"github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/helm/v3"
	pulumik8syaml "github.com/pulumi/pulumi-kubernetes/sdk/v4/go/kubernetes/yaml"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	k8sapimachineryv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
)

const (
	SelfSignedIssuerName = "self-signed"
)

type Input struct {
	Workspace              string
	CertManagerAddonInput  *c2cv1deployk8cstackgcpmodel.AddonsCertManager
	CertManagerHelmRelease *helm.Release
}

func Resources(ctx *pulumi.Context, input *Input) error {
	issuerObject := buildClusterIssuerObject()
	resourceName := fmt.Sprintf("cluster-issuer-%s", issuerObject.Name)
	manifestPath := filepath.Join(input.Workspace, fmt.Sprintf("%s.yaml", resourceName))
	if err := manifest.Create(manifestPath, issuerObject); err != nil {
		return errors.Wrapf(err, "failed to create %s manifest file", manifestPath)
	}
	_, err := pulumik8syaml.NewConfigFile(ctx, resourceName,
		&pulumik8syaml.ConfigFileArgs{File: manifestPath}, pulumi.Parent(input.CertManagerHelmRelease))
	if err != nil {
		return errors.Wrap(err, "failed to add self-signed cluster-issuer manifest")
	}
	return nil
}

func buildClusterIssuerObject() *v1.ClusterIssuer {
	return &v1.ClusterIssuer{
		TypeMeta: k8sapimachineryv1.TypeMeta{
			APIVersion: "cert-manager.io/v1",
			Kind:       "ClusterIssuer",
		},
		ObjectMeta: k8sapimachineryv1.ObjectMeta{
			Name: SelfSignedIssuerName,
		},
		Spec: v1.IssuerSpec{
			IssuerConfig: v1.IssuerConfig{
				SelfSigned: &v1.SelfSignedIssuer{},
			},
		},
	}
}
