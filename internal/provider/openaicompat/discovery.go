package openaicompat

import (
	"context"

	"neo-code/internal/provider"
	httpdiscovery "neo-code/internal/provider/discovery/http"
)

// fetchModels 调用通用 discovery HTTP 引擎，并输出标准化原始模型对象列表。
func (p *Provider) fetchModels(ctx context.Context) ([]map[string]any, error) {
	discoveryProtocol, discoveryEndpointPath, responseProfile, err := provider.ResolveDriverDiscoveryConfig(
		p.cfg.Driver,
		p.cfg.DiscoveryEndpointPath,
	)
	if err != nil {
		return nil, provider.NewDiscoveryConfigError(err.Error())
	}
	authStrategy, apiVersion := provider.ResolveDriverAuthConfig(p.cfg.Driver)

	return httpdiscovery.DiscoverRawModels(ctx, p.client, httpdiscovery.RequestConfig{
		BaseURL:           p.cfg.BaseURL,
		EndpointPath:      discoveryEndpointPath,
		DiscoveryProtocol: discoveryProtocol,
		ResponseProfile:   responseProfile,
		AuthStrategy:      authStrategy,
		APIKey:            p.cfg.APIKey,
		APIVersion:        apiVersion,
	})
}
