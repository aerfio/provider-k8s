package restcfgutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apisv1alpha1 "aerf.io/provider-k8s/apis/v1alpha1"
)

func TestRestConfigFromProviderConfig(t *testing.T) {
	tests := []struct {
		name    string
		pc      *apisv1alpha1.ProviderConfig
		cli     client.Client
		setupFn func(t *testing.T)
		wantErr bool
	}{
		{
			name: "CredentialsSourceInjectedIdentity",
			pc: &apisv1alpha1.ProviderConfig{
				Spec: apisv1alpha1.ProviderConfigSpec{
					Credentials: apisv1alpha1.ProviderCredentials{
						Source: xpv1.CredentialsSourceInjectedIdentity,
					},
				},
			},
			setupFn: func(t *testing.T) {
				tmpDir := t.TempDir()
				kubeconfigFileName := filepath.Join(tmpDir, "kubeconfig")
				require.NoError(t, os.WriteFile(kubeconfigFileName, []byte(kubeConfigFixture), 0o600))
				t.Setenv("KUBECONFIG", kubeconfigFileName)
			},
		},
		{
			name: "CredentialsSourceSecret with existing secret key",
			pc: &apisv1alpha1.ProviderConfig{
				Spec: apisv1alpha1.ProviderConfigSpec{
					Credentials: apisv1alpha1.ProviderCredentials{
						Source: xpv1.CredentialsSourceSecret,
						CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
							SecretRef: &xpv1.SecretKeySelector{
								SecretReference: xpv1.SecretReference{
									Name:      "name",
									Namespace: "ns",
								},
								Key: "key",
							},
						},
					},
				},
			},
			cli: &test.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				secret := obj.(*corev1.Secret)
				secret.Data = map[string][]byte{
					"key": []byte(kubeConfigFixture),
				}
				return nil
			}},
		},
		{
			name: "CredentialsSourceSecret with non existing secret key",
			pc: &apisv1alpha1.ProviderConfig{
				Spec: apisv1alpha1.ProviderConfigSpec{
					Credentials: apisv1alpha1.ProviderCredentials{
						Source: xpv1.CredentialsSourceSecret,
						CommonCredentialSelectors: xpv1.CommonCredentialSelectors{
							SecretRef: &xpv1.SecretKeySelector{
								SecretReference: xpv1.SecretReference{
									Name:      "name",
									Namespace: "ns",
								},
								Key: "key",
							},
						},
					},
				},
			},
			cli: &test.MockClient{MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
				return nil
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RestConfigFromProviderConfig(context.Background(), tt.pc, tt.cli)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RestConfigFromProviderConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

const kubeConfigFixture = `apiVersion: v1
clusters:
- cluster:
    server: https://example.com
  name: k8s
contexts:
- context:
    cluster: k8s
    user: k8s
  name: k8s
current-context: k8s
kind: Config
preferences: {}
users:
- name: k8s
  user:
    token: kubeconfig-u-token
`
