package vars

var (
	NetworkProjectApis = []string{
		"compute.googleapis.com",
		"container.googleapis.com",
		"dns.googleapis.com",
	}

	ContainerClusterProjectApis = []string{
		"compute.googleapis.com",
		"container.googleapis.com",
		"secretmanager.googleapis.com",
		"dns.googleapis.com",
	}

	WorkloadIdentityKubeAnnotationKey = "iam.gke.io/gcp-service-account"

	// SubNetworkCidr 10.0.0.0/14
	// this subnet will be divided into two equal halves for pod-secondary-ip-range and service-secondary-ip-range
	//https://jodies.de/ipcalc?host=10.0.0.0&mask1=14&mask2=15
	SubNetworkCidr = "10.0.0.0/14"

	// KubernetesPodSecondaryIpRange https://cloud.google.com/kubernetes-engine/docs/concepts/alias-ips#cluster_sizing_secondary_range_pods
	KubernetesPodSecondaryIpRange = "10.0.0.0/15"
	// KubernetesServiceSecondaryIpRange https://cloud.google.com/kubernetes-engine/docs/concepts/alias-ips#cluster_sizing_secondary_range_svcs
	KubernetesServiceSecondaryIpRange = "10.2.0.0/15"

	ApiServerIpCidr                                     = "172.16.0.0/24"
	ClusterMasterAuthorizedNetworksCidrBlock            = "0.0.0.0/0"
	ClusterMasterAuthorizedNetworksCidrBlockDescription = "kubectl-from-anywhere"
	ApiServerWebhookPort                                = "8443"
	IstioPilotWebhookPort                               = "15017"

	// WorkloadDeployServiceAccountName name of the google service account to
	//be used for deploying workloads to the gke cluster.
	WorkloadDeployServiceAccountName = "workload-deployer"

	CertManager = struct {
		HelmChartName string
		HelmChartRepo string
		//https://github.com/cert-manager/cert-manager/releases/tag/v1.15.1
		HelmChartVersion     string
		KsaName              string
		Namespace            string
		SelfSignedIssuerName string
	}{
		"cert-manager",
		"https://charts.jetstack.io",
		"v1.15.1",
		"cert-manager",
		"cert-manager",
		"self-signed",
	}

	ExternalSecrets = struct {
		HelmChartName string
		HelmChartRepo string
		//https://github.com/cert-manager/cert-manager/releases/tag/v1.15.1
		HelmChartVersion string
		KsaName          string
		Namespace        string
		//caution: setting this frequency may incur additional charges on some platforms
		SecretsPollingIntervalSeconds int
	}{
		"external-secrets",
		"https://charts.external-secrets.io",
		"v0.9.20",
		"external-secrets",
		"external-secrets",
		10,
	}
)
