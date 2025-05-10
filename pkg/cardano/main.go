package cardano

import (
	"cardano-valley/pkg/logger"
	"os/exec"
)


type CommandArgs []string

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
	logger.Record.Debug("CARDANO", "OUTPUT", string(output))

	return output, err
}
