package types

const (
	// MaxSessionAssetBytes 定义 session_asset 在读写链路中的统一大小上限（20 MiB）。
	MaxSessionAssetBytes int64 = 20 * 1024 * 1024
	// MaxSessionAssetsTotalBytes 定义单次请求允许携带的 session_asset 原始总字节上限（20 MiB）。
	MaxSessionAssetsTotalBytes int64 = 20 * 1024 * 1024
)

// SessionAssetLimits 描述 session_asset 在单文件与单次请求维度的限制。
type SessionAssetLimits struct {
	MaxSessionAssetBytes       int64
	MaxSessionAssetsTotalBytes int64
}

// DefaultSessionAssetLimits 返回 session_asset 限制的默认值。
func DefaultSessionAssetLimits() SessionAssetLimits {
	return SessionAssetLimits{
		MaxSessionAssetBytes:       MaxSessionAssetBytes,
		MaxSessionAssetsTotalBytes: MaxSessionAssetsTotalBytes,
	}
}

// NormalizeSessionAssetLimits 归一化 session_asset 限制并施加硬上限兜底。
func NormalizeSessionAssetLimits(limits SessionAssetLimits) SessionAssetLimits {
	normalized := limits
	if normalized.MaxSessionAssetBytes <= 0 {
		normalized.MaxSessionAssetBytes = MaxSessionAssetBytes
	}
	if normalized.MaxSessionAssetsTotalBytes <= 0 {
		normalized.MaxSessionAssetsTotalBytes = MaxSessionAssetsTotalBytes
	}
	if normalized.MaxSessionAssetBytes > MaxSessionAssetBytes {
		normalized.MaxSessionAssetBytes = MaxSessionAssetBytes
	}
	if normalized.MaxSessionAssetsTotalBytes > MaxSessionAssetsTotalBytes {
		normalized.MaxSessionAssetsTotalBytes = MaxSessionAssetsTotalBytes
	}
	if normalized.MaxSessionAssetsTotalBytes < normalized.MaxSessionAssetBytes {
		normalized.MaxSessionAssetsTotalBytes = normalized.MaxSessionAssetBytes
	}
	return normalized
}
