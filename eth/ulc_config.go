package eth

const DefaultUTCMinTrustedFraction = 75

// ULCConfig is a Ultra Light client options.
type ULCConfig struct {
	TrustedServers     []string `toml:",omitempty"` // A list of trusted servers
	MinTrustedFraction int      `toml:",omitempty"` // Minimum percentage of connected trusted servers to validate trusted (1-100)
	MinTrustedNodesNum uint     `toml:",omitempty"` // Minimum num of connected trusted servers to validate trusted (1-...)
}
