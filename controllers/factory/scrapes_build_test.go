package factory

import (
	victoriametricsv1beta1 "github.com/VictoriaMetrics/operator/api/v1beta1"
	"gopkg.in/yaml.v2"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func Test_addTLStoYaml(t *testing.T) {
	type args struct {
		cfg       yaml.MapSlice
		namespace string
		tls       *victoriametricsv1beta1.TLSConfig
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "check ca only added to config",
			args: args{
				namespace: "default",
				cfg:       yaml.MapSlice{},
				tls: &victoriametricsv1beta1.TLSConfig{
					CA: victoriametricsv1beta1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							Key: "ca",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "tls-secret",
							},
						},
					},
					Cert: victoriametricsv1beta1.SecretOrConfigMap{},
				},
			},
			want: `tls_config:
  insecure_skip_verify: false
  ca_file: /etc/vmagent-tls/certs/default_tls-secret_ca
`,
		},
		{
			name: "check ca,cert and key added to config",
			args: args{
				namespace: "default",
				cfg:       yaml.MapSlice{},
				tls: &victoriametricsv1beta1.TLSConfig{
					CA: victoriametricsv1beta1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							Key: "ca",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "tls-secret",
							},
						},
					},
					Cert: victoriametricsv1beta1.SecretOrConfigMap{
						Secret: &v1.SecretKeySelector{
							Key: "cert",
							LocalObjectReference: v1.LocalObjectReference{
								Name: "tls-secret",
							},
						},
					},
					KeySecret: &v1.SecretKeySelector{
						Key: "key",
						LocalObjectReference: v1.LocalObjectReference{
							Name: "tls-secret",
						},
					},
				},
			},
			want: `tls_config:
  insecure_skip_verify: false
  ca_file: /etc/vmagent-tls/certs/default_tls-secret_ca
  cert_file: /etc/vmagent-tls/certs/default_tls-secret_cert
  key_file: /etc/vmagent-tls/certs/default_tls-secret_key
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addTLStoYaml(tt.args.cfg, tt.args.namespace, tt.args.tls)
			gotBytes, err := yaml.Marshal(got)
			if err != nil {
				t.Errorf("cannot marshal tlsConfig to yaml format: %e", err)
				return
			}
			if !reflect.DeepEqual(string(gotBytes), tt.want) {
				t.Errorf("addTLStoYaml() \ngot: \n%v \nwant \n%v", string(gotBytes), tt.want)
			}
		})
	}
}

func Test_generateServiceScrapeConfig(t *testing.T) {
	type args struct {
		m                        *victoriametricsv1beta1.VMServiceScrape
		ep                       victoriametricsv1beta1.Endpoint
		i                        int
		apiserverConfig          *victoriametricsv1beta1.APIServerConfig
		basicAuthSecrets         map[string]BasicAuthCredentials
		bearerTokens             map[string]BearerToken
		overrideHonorLabels      bool
		overrideHonorTimestamps  bool
		ignoreNamespaceSelectors bool
		enforcedNamespaceLabel   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "generate simple config",
			args: args{
				m: &victoriametricsv1beta1.VMServiceScrape{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-scrape",
						Namespace: "default",
					},
					Spec: victoriametricsv1beta1.VMServiceScrapeSpec{
						Endpoints: []victoriametricsv1beta1.Endpoint{
							{
								Port: "8080",
								TLSConfig: &victoriametricsv1beta1.TLSConfig{
									CA: victoriametricsv1beta1.SecretOrConfigMap{
										Secret: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: "tls-secret",
											},
											Key: "ca",
										},
									},
								},
								BearerTokenFile: "/var/run/tolen",
							},
						},
					},
				},
				ep: victoriametricsv1beta1.Endpoint{
					Port: "8080",
					TLSConfig: &victoriametricsv1beta1.TLSConfig{
						Cert: victoriametricsv1beta1.SecretOrConfigMap{},
						CA: victoriametricsv1beta1.SecretOrConfigMap{
							Secret: &v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "tls-secret",
								},
								Key: "ca",
							},
						},
					},
					BearerTokenFile: "/var/run/tolen",
				},
				i:                        0,
				apiserverConfig:          nil,
				basicAuthSecrets:         nil,
				bearerTokens:             map[string]BearerToken{},
				overrideHonorLabels:      false,
				overrideHonorTimestamps:  false,
				ignoreNamespaceSelectors: false,
				enforcedNamespaceLabel:   "",
			},
			want: `job_name: default/test-scrape/0
honor_labels: false
kubernetes_sd_configs:
- role: endpoints
  namespaces:
    names:
    - default
tls_config:
  insecure_skip_verify: false
  ca_file: /etc/vmagent-tls/certs/default_tls-secret_ca
bearer_token_file: /var/run/tolen
relabel_configs:
- action: keep
  source_labels:
  - __meta_kubernetes_endpoint_port_name
  regex: "8080"
- source_labels:
  - __meta_kubernetes_endpoint_address_target_kind
  - __meta_kubernetes_endpoint_address_target_name
  separator: ;
  regex: Node;(.*)
  replacement: ${1}
  target_label: node
- source_labels:
  - __meta_kubernetes_endpoint_address_target_kind
  - __meta_kubernetes_endpoint_address_target_name
  separator: ;
  regex: Pod;(.*)
  replacement: ${1}
  target_label: pod
- source_labels:
  - __meta_kubernetes_namespace
  target_label: namespace
- source_labels:
  - __meta_kubernetes_service_name
  target_label: service
- source_labels:
  - __meta_kubernetes_pod_name
  target_label: pod
- source_labels:
  - __meta_kubernetes_service_name
  target_label: job
  replacement: ${1}
- target_label: endpoint
  replacement: "8080"
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateServiceScrapeConfig(tt.args.m, tt.args.ep, tt.args.i, tt.args.apiserverConfig, tt.args.basicAuthSecrets, tt.args.bearerTokens, tt.args.overrideHonorLabels, tt.args.overrideHonorTimestamps, tt.args.ignoreNamespaceSelectors, tt.args.enforcedNamespaceLabel)
			gotBytes, err := yaml.Marshal(got)
			if err != nil {
				t.Errorf("cannot marshal ServiceScrapeConfig to yaml,err :%e", err)
				return
			}
			if !reflect.DeepEqual(string(gotBytes), tt.want) {
				t.Errorf("generateServiceScrapeConfig() \ngot = \n%v, \nwant \n%v", string(gotBytes), tt.want)
			}
		})
	}
}
