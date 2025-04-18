package cardano

import (
	"cardano-valley/pkg/logger"
	"os/exec"
)


type CommandArgs []string

func Run(args CommandArgs) ([]byte, error) {
	output, err := exec.Command("/usr/local/bin/cardano-cli", args...).CombinedOutput()
	if err != nil {
		logger.Record.Error("CARDAGO", "PACKAGE", "CARDANO", "ERROR", err, "OUTPUT", string(output))
	}
	logger.Record.Debug("CARDAGO", "PACKAGE", "CARDANO", "OUTPUT", string(output))

	return output, err
}
