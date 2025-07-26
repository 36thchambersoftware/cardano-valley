package cardano

import (
	"cardano-valley/pkg/logger"
	"os/exec"
)


type (
	CommandArgs []string
	Holder struct {
		Address string `json:"address,omitempty"` // Address of the holder
		Quantity uint64 `json:"quantity,omitempty"` // Quantity of the asset held
	}
	
	Asset string // policyid.assetname
	Assets []Asset
)

const (
	NETWORK = "--mainnet"
)

func init() {
	// // Check if cardano-cli is installed
	// if _, err := exec.LookPath("cardano-cli"); err != nil {
	// 	logger.Record.Error("CARDANO", "CARDANO-CLI NOT FOUND", err)
	// } else {
	// 	logger.Record.Info("CARDANO", "CARDANO-CLI FOUND")
	// }
}

func Run(args CommandArgs) ([]byte, error) {
	logger.Record.Info("CARDANO", "COMMAND", args)
	output, err := exec.Command("/usr/local/bin/cardano-cli", args...).CombinedOutput()
	if err != nil {
		logger.Record.Error("CARDANO", "ERROR", err, "OUTPUT", string(output))
	}
	logger.Record.Info("CARDANO", "OUTPUT", string(output))

	return output, err
}