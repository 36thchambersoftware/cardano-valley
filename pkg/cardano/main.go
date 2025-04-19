package cardano

import (
	"cardano-valley/pkg/logger"
	"os/exec"
)


type CommandArgs []string

func Run(args CommandArgs) ([]byte, error) {
	logger.Record.Info("CARDANO", "COMMAND", args)
	output, err := exec.Command("/usr/local/bin/cardano-cli", args...).CombinedOutput()
	if err != nil {
		logger.Record.Error("CARDANO", "ERROR", err, "OUTPUT", string(output))
	}
	logger.Record.Debug("CARDANO", "OUTPUT", string(output))

	return output, err
}
