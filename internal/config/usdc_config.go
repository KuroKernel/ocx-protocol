package config

import (
	"fmt"
	"os"
	"strconv"
)

// USDCConfig represents USDC settlement configuration
type USDCConfig struct {
	// Network configuration
	PolygonRPCURL    string `json:"polygon_rpc_url"`
	EthereumRPCURL   string `json:"ethereum_rpc_url"`
	ChainID          int64  `json:"chain_id"`
	
	// Contract addresses
	USDCContractAddr   string `json:"usdc_contract_address"`
	EscrowContractAddr string `json:"escrow_contract_address"`
	
	// Security
	PrivateKeyHex string `json:"private_key_hex"`
	
	// Fee configuration
	ProtocolFeeBPS int `json:"protocol_fee_bps"` // Basis points (100 bps = 1%)
	
	// Settlement parameters
	ConfirmationBlocks int    `json:"confirmation_blocks"`
	MaxGasPrice        int64  `json:"max_gas_price"`
	GasLimit           int64  `json:"gas_limit"`
	
	// Dispute resolution
	DisputeTimeoutHours int `json:"dispute_timeout_hours"`
	ArbitrationFeeBPS   int `json:"arbitration_fee_bps"`
}

// NewUSDCConfig creates USDC configuration from environment variables
func NewUSDCConfig() (*USDCConfig, error) {
	config := &USDCConfig{
		// Default values
		ChainID:             137, // Polygon mainnet
		ProtocolFeeBPS:      250, // 2.5%
		ConfirmationBlocks:  12,
		MaxGasPrice:         100000000000, // 100 gwei
		GasLimit:            200000,
		DisputeTimeoutHours: 72,  // 3 days
		ArbitrationFeeBPS:   100, // 1%
	}
	
	// Load from environment variables
	if rpcURL := os.Getenv("OCX_POLYGON_RPC_URL"); rpcURL != "" {
		config.PolygonRPCURL = rpcURL
	} else {
		config.PolygonRPCURL = "https://polygon-rpc.com"
	}
	
	if ethRPCURL := os.Getenv("OCX_ETHEREUM_RPC_URL"); ethRPCURL != "" {
		config.EthereumRPCURL = ethRPCURL
	} else {
		config.EthereumRPCURL = "https://mainnet.infura.io/v3/your-project-id"
	}
	
	if usdcAddr := os.Getenv("OCX_USDC_CONTRACT_ADDRESS"); usdcAddr != "" {
		config.USDCContractAddr = usdcAddr
	} else {
		config.USDCContractAddr = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174" // USDC on Polygon
	}
	
	if escrowAddr := os.Getenv("OCX_ESCROW_CONTRACT_ADDRESS"); escrowAddr != "" {
		config.EscrowContractAddr = escrowAddr
	} else {
		return nil, fmt.Errorf("OCX_ESCROW_CONTRACT_ADDRESS environment variable is required")
	}
	
	if privateKey := os.Getenv("OCX_PRIVATE_KEY"); privateKey != "" {
		config.PrivateKeyHex = privateKey
	} else {
		return nil, fmt.Errorf("OCX_PRIVATE_KEY environment variable is required")
	}
	
	// Parse optional integer configurations
	if chainIDStr := os.Getenv("OCX_CHAIN_ID"); chainIDStr != "" {
		if chainID, err := strconv.ParseInt(chainIDStr, 10, 64); err == nil {
			config.ChainID = chainID
		}
	}
	
	if protocolFeeBPSStr := os.Getenv("OCX_PROTOCOL_FEE_BPS"); protocolFeeBPSStr != "" {
		if feeBPS, err := strconv.Atoi(protocolFeeBPSStr); err == nil {
			config.ProtocolFeeBPS = feeBPS
		}
	}
	
	if confirmationBlocksStr := os.Getenv("OCX_CONFIRMATION_BLOCKS"); confirmationBlocksStr != "" {
		if blocks, err := strconv.Atoi(confirmationBlocksStr); err == nil {
			config.ConfirmationBlocks = blocks
		}
	}
	
	if maxGasPriceStr := os.Getenv("OCX_MAX_GAS_PRICE"); maxGasPriceStr != "" {
		if gasPrice, err := strconv.ParseInt(maxGasPriceStr, 10, 64); err == nil {
			config.MaxGasPrice = gasPrice
		}
	}
	
	if gasLimitStr := os.Getenv("OCX_GAS_LIMIT"); gasLimitStr != "" {
		if gasLimit, err := strconv.ParseInt(gasLimitStr, 10, 64); err == nil {
			config.GasLimit = gasLimit
		}
	}
	
	if disputeTimeoutStr := os.Getenv("OCX_DISPUTE_TIMEOUT_HOURS"); disputeTimeoutStr != "" {
		if timeout, err := strconv.Atoi(disputeTimeoutStr); err == nil {
			config.DisputeTimeoutHours = timeout
		}
	}
	
	if arbitrationFeeBPSStr := os.Getenv("OCX_ARBITRATION_FEE_BPS"); arbitrationFeeBPSStr != "" {
		if feeBPS, err := strconv.Atoi(arbitrationFeeBPSStr); err == nil {
			config.ArbitrationFeeBPS = feeBPS
		}
	}
	
	return config, nil
}

// Validate validates the USDC configuration
func (c *USDCConfig) Validate() error {
	if c.PolygonRPCURL == "" {
		return fmt.Errorf("polygon RPC URL is required")
	}
	
	if c.USDCContractAddr == "" {
		return fmt.Errorf("USDC contract address is required")
	}
	
	if c.EscrowContractAddr == "" {
		return fmt.Errorf("escrow contract address is required")
	}
	
	if c.PrivateKeyHex == "" {
		return fmt.Errorf("private key is required")
	}
	
	if c.ProtocolFeeBPS < 0 || c.ProtocolFeeBPS > 1000 {
		return fmt.Errorf("protocol fee must be between 0 and 1000 basis points")
	}
	
	if c.ChainID <= 0 {
		return fmt.Errorf("chain ID must be positive")
	}
	
	return nil
}

// GetNetworkName returns the network name based on chain ID
func (c *USDCConfig) GetNetworkName() string {
	switch c.ChainID {
	case 1:
		return "ethereum"
	case 137:
		return "polygon"
	case 56:
		return "binance"
	case 43114:
		return "avalanche"
	default:
		return "unknown"
	}
}

// IsMainnet returns true if the configuration is for mainnet
func (c *USDCConfig) IsMainnet() bool {
	mainnetChainIDs := map[int64]bool{
		1:     true, // Ethereum
		137:   true, // Polygon
		56:    true, // Binance Smart Chain
		43114: true, // Avalanche
	}
	return mainnetChainIDs[c.ChainID]
}

// GetProtocolFeePercentage returns the protocol fee as a percentage
func (c *USDCConfig) GetProtocolFeePercentage() float64 {
	return float64(c.ProtocolFeeBPS) / 10000.0
}

// GetArbitrationFeePercentage returns the arbitration fee as a percentage
func (c *USDCConfig) GetArbitrationFeePercentage() float64 {
	return float64(c.ArbitrationFeeBPS) / 10000.0
}
