package anthropic

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"neo-code/internal/provider"
	httpdiscovery "neo-code/internal/provider/discovery/http"
	providertypes "neo-code/internal/provider/types"
)

// DriverName 是 Anthropic 协议驱动的唯一标识。
const DriverName = provider.DriverAnthropic

// Driver 返回 Anthropic 协议驱动定义。
func Driver() provider.DriverDefinition {
	return provider.DriverDefinition{
		Name: DriverName,
		Build: func(ctx context.Context, cfg provider.RuntimeConfig) (provider.Provider, error) {
			return nil, provider.NewDiscoveryConfigError(
				fmt.Sprintf("anthropic driver: chat protocol %q is not supported yet", provider.ResolveDriverProtocolDefaults(cfg.Driver).ChatProtocol),
			)
		},
		Discover: func(ctx context.Context, cfg provider.RuntimeConfig) ([]providertypes.ModelDescriptor, error) {
			discoveryProtocol, discoveryEndpointPath, responseProfile, err := provider.ResolveDriverDiscoveryConfig(
				cfg.Driver,
				cfg.DiscoveryEndpointPath,
			)
			if err != nil {
				return nil, provider.NewDiscoveryConfigError(err.Error())
			}
			authStrategy, apiVersion := provider.ResolveDriverAuthConfig(cfg.Driver)

			rawModels, err := httpdiscovery.DiscoverRawModels(ctx, &http.Client{Timeout: 90 * time.Second}, httpdiscovery.RequestConfig{
				BaseURL:           cfg.BaseURL,
				EndpointPath:      discoveryEndpointPath,
				DiscoveryProtocol: discoveryProtocol,
				ResponseProfile:   responseProfile,
				AuthStrategy:      authStrategy,
				APIKey:            cfg.APIKey,
				APIVersion:        apiVersion,
			})
			if err != nil {
				return nil, err
			}

			descriptors := make([]providertypes.ModelDescriptor, 0, len(rawModels))
			for _, raw := range rawModels {
				descriptor, ok := providertypes.DescriptorFromRawModel(raw)
				if !ok {
					continue
				}
				descriptors = append(descriptors, descriptor)
			}
			return providertypes.MergeModelDescriptors(descriptors), nil
		},
		ValidateCatalogIdentity: validateCatalogIdentity,
	}
}

// validateCatalogIdentity 在 catalog 路径上执行 Anthropic 静态校验。
func validateCatalogIdentity(identity provider.ProviderIdentity) error {
	if _, err := provider.NormalizeProviderChatEndpointPath(identity.ChatEndpointPath); err != nil {
		return provider.NewDiscoveryConfigError(err.Error())
	}
	if _, _, _, err := provider.ResolveDriverDiscoveryConfig(identity.Driver, identity.DiscoveryEndpointPath); err != nil {
		return provider.NewDiscoveryConfigError(err.Error())
	}
	return nil
}
