package cardano

import (
	"cardano-valley/pkg/logger"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// func SendAll(from, to, signingPaymentKey string) (*cgo.Hash32, error) {
// 	protocolParams, err := Node.ProtocolParams()
// 	if err != nil {
// 		logger.Record.Error("WALLET", "Failed to get protocol parameters: ", err)
// 		return nil, err
// 	}

// 	txBuilder := cgo.NewTxBuilder(protocolParams)

// 	sender, err := cgo.NewAddress(from)
// 	if err != nil {
// 		logger.Record.Error("WALLET", "Failed to create sender address: ", err)
// 		return nil, err
// 	}
// 	receiver, err := cgo.NewAddress(to)
// 	if err != nil {
// 		logger.Record.Error("WALLET", "Failed to create receiver address: ", err)
// 		return nil, err
// 	}
// 	sk, err := crypto.NewPrvKey(signingPaymentKey)
// 	if err != nil {
// 		logger.Record.Error("WALLET", "Failed to create signing key: ", err)
// 		return nil, err
// 	}
// 	txHash, err := cgo.NewHash32("txhash")
// 	if err != nil {
// 		logger.Record.Error("WALLET", "Failed to create transaction hash: ", err)
// 		return nil, err
// 	}

// 	txInput := cgo.NewTxInput(txHash, 0, cgo.NewValue(20e6))
// 	txOut := cgo.NewTxOutput(receiver, cgo.NewValue(10e6))

// 	txBuilder.AddAuxiliaryData(&cgo.AuxiliaryData{
// 		Metadata: cgo.Metadata{
// 			0: map[string]interface{}{
// 				"cardano-valley": "harvest",
// 			},
// 		},
// 	})

// 	txBuilder.AddInputs(txInput)
// 	txBuilder.AddOutputs(txOut)
// 	txBuilder.SetTTL(100000)
// 	txBuilder.AddChangeIfNeeded(sender)
// 	txBuilder.Sign(sk)

// 	tx, err := txBuilder.Build()
// 	if err != nil {
// 		logger.Record.Error("WALLET", "Failed to build transaction: ", err)
// 		return nil, err
// 	}

// 	signedHash, err := Node.SubmitTx(tx)
// 	if err != nil {
// 		logger.Record.Error("WALLET", "Failed to submit transaction: ", err)
// 		return nil, err
// 	}

// 	return signedHash, nil
// }

// func getAddressUTXOs(address string) ([]byte, error) {
// 	utxoArgs := []string{
// 		"query",
// 		"utxo",
// 		"--address",
// 		address,
// 		"--mainnet",
// 	}

// 	// cardano-cli query utxo --address addr1qy339ne5579p50ee62rpjrtw3khwxwjs7st0yz5dhhzl4lnr8c9t8cselvc44grattsfkemsvrjwrxp5mfevl7qn9s6qz80eg9
// 	utxoBytes, err := Run(utxoArgs)
// 	if err != nil {
// 		logger.Record.Error("WALLET", "Failed to get UTXOs: ", err)
// 		return nil, err
// 	}

// 	return utxoBytes, nil
// }

type (
    UTxOValue map[string]map[string]uint64

    UTxOEntry struct {
		Address           string     `json:"address"`
		Datum             any        `json:"datum"`
		DatumHash         any        `json:"datumhash"`
		InlineDatum       any        `json:"inlineDatum"`
		InlineDatumRaw    any        `json:"inlineDatumRaw"`
		ReferenceScript   any        `json:"referenceScript"`
		Value             UTxOValue  `json:"value"`
	}

 	UTxOMap map[string]UTxOEntry

	TxOutMap map[string]struct{
		Asset Asset
		Amount uint64
	}
)

func QueryUTxOJson(addr string) (UTxOMap, error) {
	tmp := "utxos.json"
	args := CommandArgs{
		"query", "utxo",
		"--address", addr,
		"--out-file", tmp,
		"--output-json",
		NETWORK,
	}
	args = append(args, strings.Split(NETWORK, " ")...)
	output, err := Run(args)
	if err != nil {
		logger.Record.Error("CARDANO", "Failed to query UTxO: ", err)
		return nil, err
	}

	var utxos UTxOMap
	if err := json.Unmarshal(output, &utxos); err != nil {
		logger.Record.Error("CARDANO", "Failed to unmarshal UTxO JSON: ", err)
		return nil, err
	}
	return utxos, nil
}

func BuildRawTransaction(txIns []string, txOut string, changeAddr string, outFile string) error {
	args := []string{
		"conway",
		"transaction", "build",
		"--alonzo-era",
		"--change-address", changeAddr,
		"--out-file", outFile,
	}
	for _, txIn := range txIns {
		args = append(args, "--tx-in", txIn)
	}
	args = append(args, "--tx-out", txOut)
	args = append(args, strings.Split(NETWORK, " ")...)
	output, err := Run(args)
	if err != nil {
		logger.Record.Error("CARDANO", "Failed to build transaction: ", err)
		return err
	}
	logger.Record.Info("CARDANO", "Transaction built successfully: ", string(output))
	return nil
}

func SignTransaction(rawFile, skeyFile, outFile string) error {
	args := []string{
		"conway", "transaction", "sign",
		"--tx-body-file", rawFile,
		"--signing-key-file", skeyFile,
		"--out-file", outFile,
	}
	args = append(args, strings.Split(NETWORK, " ")...)
	output, err := Run(args)
	if err != nil {
		logger.Record.Error("CARDANO", "Failed to sign transaction: ", err)
		return err
	}
	logger.Record.Info("CARDANO", "Transaction signed successfully: ", string(output))
	return nil
}

func SubmitTransaction(signedFile string) error {
	args := []string{
		"conway", "transaction", "submit",
		"--tx-file", signedFile,
	}
	args = append(args, strings.Split(NETWORK, " ")...)
	output, err := Run(args)
	if err != nil {
		logger.Record.Error("CARDANO", "Failed to submit transaction: ", err)
		return err
	}
	logger.Record.Info("CARDANO", "Transaction submitted successfully: ", string(output))
	return nil
}

func FormatAssetString(value UTxOValue) string {
	var sb strings.Builder
	for policyID, assets := range value {
		if policyID == "lovelace" {
			continue
		}
		for assetNameHex, qty := range assets {
			sb.WriteString(" + ")
			sb.WriteString(fmt.Sprintf("%d %s.%s", qty, policyID, assetNameHex))
		}
	}
	return strings.TrimPrefix(sb.String(), " + ")
}

func SelectUTxOsForMinADA(utxos UTxOMap, minADA uint64) ([]string, uint64) {
	selected := []string{}
	total := uint64(0)
	keys := make([]string, 0, len(utxos))
	for k := range utxos {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		val := utxos[k].Value
		lovelace := val["lovelace"][""]
		total += lovelace
		selected = append(selected, k)
		if total >= minADA {
			break
		}
	}
	return selected, total
}

func ExampleSendTx(fromAddr, toAddress, changeAddress, skeyPath string, minADA uint64) error {
	utxos, err := QueryUTxOJson(fromAddr)
	if err != nil {
		return errors.New("utxo query failed: " + err.Error())
	}

	txIns, totalADA := SelectUTxOsForMinADA(utxos, minADA)
	if totalADA < minADA {
		return errors.New("not enough ADA available")
	}

	var assetParts []string
	for _, txIn := range txIns {
		assetParts = append(assetParts, FormatAssetString(utxos[txIn].Value))
	}

	assetStr := strings.Join(assetParts, " + ")
	txOut := fmt.Sprintf("%s+%d+%s", toAddress, minADA, assetStr)
	txRaw := "tx.raw"
	txSigned := "tx.signed"

	err = BuildRawTransaction(txIns, txOut, changeAddress, txRaw)
	if err != nil {
		return errors.New("build failed: " + err.Error())
	}

	err = SignTransaction(txRaw, skeyPath, txSigned)
	if err != nil {
		return errors.New("sign failed: " + err.Error())
	}

	err = SubmitTransaction(txSigned)
	if err != nil {
		return errors.New("submit failed: " + err.Error())
	}

	return nil
}

func SelectUTxOsWithAssets(utxos UTxOMap, requiredAssets map[string]map[string]uint64) ([]string, error) {
	selected := []string{}
	accumulated := make(map[string]map[string]uint64)

	for k, utxo := range utxos {
		selected = append(selected, k)
		for policy, assets := range utxo.Value {
			if policy == "lovelace" {
				continue
			}
			if _, ok := accumulated[policy]; !ok {
				accumulated[policy] = make(map[string]uint64)
			}
			for name, qty := range assets {
				accumulated[policy][name] += qty
			}
		}
		if containsAllAssets(accumulated, requiredAssets) {
			return selected, nil
		}
	}
	return nil, errors.New("could not find enough tokens to satisfy requirements")
}

func containsAllAssets(acc, required map[string]map[string]uint64) bool {
	for policy, assets := range required {
		for name, reqQty := range assets {
			if acc[policy][name] < reqQty {
				return false
			}
		}
	}
	return true
}

func ExampleSendTxWithAssets(fromAddr, toAddress, changeAddress, skeyPath string, minADA uint64) error {
	utxos, err := QueryUTxOJson(fromAddr)
	if err != nil {
		return errors.New("utxo query failed: " + err.Error())
	}

	txIns, totalADA := SelectUTxOsForMinADA(utxos, minADA)
	if totalADA < minADA {
		return errors.New("not enough ADA available")
	}

	var assetParts []string
	for _, txIn := range txIns {
		assetParts = append(assetParts, FormatAssetString(utxos[txIn].Value))
	}

	assetStr := strings.Join(assetParts, " + ")
	txOut := fmt.Sprintf("%s+%d+%s", toAddress, minADA, assetStr)
	txRaw := "tx.raw"
	txSigned := "tx.signed"

	err = BuildRawTransaction(txIns, txOut, changeAddress, txRaw)
	if err != nil {
		return errors.New("build failed: " + err.Error())
	}

	err = SignTransaction(txRaw, skeyPath, txSigned)
	if err != nil {
		return errors.New("sign failed: " + err.Error())
	}

	err = SubmitTransaction(txSigned)
	if err != nil {
		return errors.New("submit failed: " + err.Error())
	}

	return nil
}


// Mirrors the airdrop functionality from lookout-below
func BuildTxAdaOnly(changeAddr string, holders []Holder, adaPerNFT uint64) error {
	txOuts := []string{}
	for _, h := range holders {
		total := adaPerNFT * h.Quantity
		slog.Default().Info("Calculating transaction output",
			"holder_address", h.Address,
			"nft_count", h.Quantity,
			"ada_per_nft", adaPerNFT,
			"total_ada", total,
		)
		txOuts = append(txOuts, "--tx-out", fmt.Sprintf("%s+%d", h.Address, total))
	}

	// 1. Query UTXOs in JSON format
	utxoCmd := exec.Command("cardano-cli", "query", "utxo",
		"--address", changeAddr,
		"--mainnet",
		"--out-file", "/dev/stdout",
		"--output-json",
	)

	utxoOutput, err := utxoCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to query UTXOs: %w", err)
	}

	// 2. Parse JSON into map
	var utxos UTxOMap
	if err := json.Unmarshal(utxoOutput, &utxos); err != nil {
		return fmt.Errorf("failed to parse UTXO JSON: %w", err)
	}

	// 3. Construct --tx-in arguments
	txIns := []string{}
	for utxo := range utxos {
		// key is like "txhash#txix"
		txIns = append(txIns, "--tx-in", utxo)
	}
	if len(txIns) == 0 {
		return fmt.Errorf("no UTXOs found at address %s", changeAddr)
	}

	slog.Default().Info("Building transaction",
		"wallet_address", changeAddr,
		"holders_count", len(holders),
		"ada_per_nft", adaPerNFT,
		"txOuts", txOuts,
	)

	// 4. Build full CLI command
	args := append([]string{
		"conway", "transaction", "build",
		"--mainnet",
		"--change-address", changeAddr,
		"--out-file", "airdrop-tx.raw",
	}, append(txIns, txOuts...)...)

	slog.Default().Info("Executing cardano-cli command", "args", args)

	cmd := exec.Command("cardano-cli", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}


func BuildTxFromBalance(to string, assets TxOutMap) error {
	txOuts := []string{}
	for asset, amount := range assets {
		slog.Default().Info("Calculating transaction output",
			"asset", asset,
			"amount", amount,
		)
		//--tx-out addr_test1vp6jz+"1000 11375f8ee31c280e1f2ec6fe11a73bca79d7a6a64f18e1e6980f0c74.637573746f6d636f696e"
		txOuts = append(txOuts, "--tx-out", fmt.Sprintf("%s+\"%d %s\"", to, amount, asset))
	}

	// 1. Query UTXOs in JSON format
	utxoCmd := exec.Command("cardano-cli", "query", "utxo",
		"--address", to,
		"--mainnet",
		"--out-file", "/dev/stdout",
		"--output-json",
	)

	utxoOutput, err := utxoCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to query UTXOs: %w", err)
	}

	// 2. Parse JSON into map
	var utxos UTxOMap
	if err := json.Unmarshal(utxoOutput, &utxos); err != nil {
		return fmt.Errorf("failed to parse UTXO JSON: %w", err)
	}

	// 3. Construct --tx-in arguments
	txIns := []string{}
	for utxo := range utxos {
		// key is like "txhash#txix"
		txIns = append(txIns, "--tx-in", utxo)
	}
	if len(txIns) == 0 {
		return fmt.Errorf("no UTXOs found at address %s", to)
	}

	// slog.Default().Info("Building transaction",
	// 	"wallet_address", to,
	// 	"holders_count", len(holders),
	// 	"ada_per_nft", adaPerNFT,
	// 	"txOuts", txOuts,
	// )

	// 4. Build full CLI command
	args := append([]string{
		"conway", "transaction", "build",
		"--mainnet",
		"--change-address", to,
		"--out-file", "airdrop-tx.raw",
	}, append(txIns, txOuts...)...)

	slog.Default().Info("Executing cardano-cli command", "args", args)

	cmd := exec.Command("cardano-cli", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}