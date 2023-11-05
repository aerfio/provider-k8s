package restcfgutil

import (
	"context"
	"fmt"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apisv1alpha1 "aerf.io/provider-k8s/apis/v1alpha1"
)

func RestConfigFromProviderConfig(ctx context.Context, pc *apisv1alpha1.ProviderConfig, cli client.Client) (*rest.Config, error) {
	var err error
	var rc *rest.Config
	switch cd := pc.Spec.Credentials; cd.Source {
	case xpv1.CredentialsSourceInjectedIdentity:
		rc, err = ctrl.GetConfig()
		if err != nil {
			return nil, errors.Wrap(err, "couldn't get rest.Config from in-cluster data")
		}
	default:
		data, err := resource.CommonCredentialExtractor(ctx, cd.Source, cli, cd.CommonCredentialSelectors)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get credentials")
		}
		cfg, err := clientcmd.NewClientConfigFromBytes(data)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create clientConfig from raw bytes")
		}

		rc, err = cfg.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create *rest.Config from kubeconfig: %s", err)
		}

	}
	return rc, nil
}
