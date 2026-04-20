package provider

// DriverProtocolDefaults 描述各 driver 在运行期固定采用的协议、鉴权与发现解析默认值。
type DriverProtocolDefaults struct {
	ChatProtocol      string
	DiscoveryProtocol string
	AuthStrategy      string
	ResponseProfile   string
	APIVersion        string
}

// ResolveDriverProtocolDefaults 根据 driver 返回运行期固定协议默认值，避免由外部配置决定协议分支。
func ResolveDriverProtocolDefaults(driver string) DriverProtocolDefaults {
	switch NormalizeProviderDriver(driver) {
	case DriverGemini:
		return DriverProtocolDefaults{
			ChatProtocol:      ChatProtocolOpenAIChatCompletions,
			DiscoveryProtocol: DiscoveryProtocolGeminiModels,
			AuthStrategy:      AuthStrategyBearer,
			ResponseProfile:   DiscoveryResponseProfileGemini,
		}
	case DriverAnthropic:
		return DriverProtocolDefaults{
			ChatProtocol:      ChatProtocolAnthropicMessages,
			DiscoveryProtocol: DiscoveryProtocolAnthropicModels,
			AuthStrategy:      AuthStrategyAnthropic,
			ResponseProfile:   DiscoveryResponseProfileGeneric,
		}
	case DriverOpenAICompat:
		return DriverProtocolDefaults{
			ChatProtocol:      ChatProtocolOpenAIChatCompletions,
			DiscoveryProtocol: DiscoveryProtocolOpenAIModels,
			AuthStrategy:      AuthStrategyBearer,
			ResponseProfile:   DiscoveryResponseProfileOpenAI,
		}
	default:
		return DriverProtocolDefaults{
			ChatProtocol:      ChatProtocolOpenAIChatCompletions,
			DiscoveryProtocol: DiscoveryProtocolCustomHTTPJSON,
			AuthStrategy:      AuthStrategyBearer,
			ResponseProfile:   DiscoveryResponseProfileGeneric,
		}
	}
}

// ResolveDriverDiscoveryConfig 解析 discovery 请求所需配置，并在端点为空时注入 driver 约定默认值。
func ResolveDriverDiscoveryConfig(driver string, endpointPath string) (string, string, string, error) {
	defaults := ResolveDriverProtocolDefaults(driver)
	normalizedEndpointPath, err := NormalizeProviderDiscoveryEndpointPath(endpointPath)
	if err != nil {
		return "", "", "", err
	}
	if normalizedEndpointPath == "" {
		normalizedEndpointPath = defaultDiscoveryEndpointPath(defaults.DiscoveryProtocol)
	}
	return defaults.DiscoveryProtocol, normalizedEndpointPath, defaults.ResponseProfile, nil
}

// ResolveDriverAuthConfig 返回 driver 运行期鉴权策略及附加版本配置。
func ResolveDriverAuthConfig(driver string) (string, string) {
	defaults := ResolveDriverProtocolDefaults(driver)
	return defaults.AuthStrategy, defaults.APIVersion
}
