package gemini

import (
	"context"

	"neo-code/internal/provider"
	"neo-code/internal/provider/openaicompat"
	providertypes "neo-code/internal/provider/types"
)

// DriverName 是 Gemini 协议驱动的唯一标识。
const DriverName = provider.DriverGemini

// Driver 返回 Gemini 协议驱动定义。
func Driver() provider.DriverDefinition {
	compatDriver := openaicompat.Driver()

	return provider.DriverDefinition{
		Name: DriverName,
		Build: func(ctx context.Context, cfg provider.RuntimeConfig) (provider.Provider, error) {
			return compatDriver.Build(ctx, cfg)
		},
		Discover: func(ctx context.Context, cfg provider.RuntimeConfig) ([]providertypes.ModelDescriptor, error) {
			return compatDriver.Discover(ctx, cfg)
		},
		ValidateCatalogIdentity: validateCatalogIdentity,
	}
}

// validateCatalogIdentity 在 catalog 路径上执行 Gemini 静态校验，避免无效快照误导选择流程。
func validateCatalogIdentity(identity provider.ProviderIdentity) error {
	if _, err := provider.NormalizeProviderChatEndpointPath(identity.ChatEndpointPath); err != nil {
		return provider.NewDiscoveryConfigError(err.Error())
	}
	if _, _, _, err := provider.ResolveDriverDiscoveryConfig(identity.Driver, identity.DiscoveryEndpointPath); err != nil {
		return provider.NewDiscoveryConfigError(err.Error())
	}
	return nil
}
