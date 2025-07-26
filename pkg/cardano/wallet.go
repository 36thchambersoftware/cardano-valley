package cardano

import (
	"cardano-valley/pkg/db"
	"cardano-valley/pkg/logger"
	"errors"
	"fmt"
	"os"
	"path"
)

type (
	Keys struct {
		Address string `bson:"address,omitempty"`
		PaymentKey string `bson:"payment_key,omitempty"`
		SigningPaymentKey string `bson:"signing_payment_key,omitempty"`
		StakeKey string `bson:"stake_key,omitempty"`
		SigningStakeKey string `bson:"signing_stake_key,omitempty"`
		DelegationCertificate string `bson:"delegation_certificate,omitempty"`
	}
)

var (
	KeyPrefix = "wallets/"
	PaymentKeySuffix = "_payment.vkey"
	SigningKeySuffix = "_payment.skey"
	StakeKeySuffix = "_stake.vkey"
	StakeSigningKeySuffix = "_stake.skey"
	AddressSuffix = ".addr"
	DelegationCertificateSuffix = "_delegation.cert"

	ErrWalletExists = errors.New("Wallet already exists")
	ErrWalletDoesNotExist = errors.New("Wallet does not exist")
)

// cardano-cli query utxo --address addr1qy339ne5579p50ee62rpjrtw3khwxwjs7st0yz5dhhzl4lnr8c9t8cselvc44grattsfkemsvrjwrxp5mfevl7qn9s6qz80eg9
// cardano-cli conway transaction build --tx-in c57f25ebf9cf1487b13deeb8449215c499f3d61c2836d84ab92a73b0bbaadd38#1 --tx-out $(< payment2.addr)+500000000 --change-address $(< payment.addr) --out-file tx.raw
// cardano-cli conway transaction sign --tx-body-file tx.raw --signing-key-file payment.skey --out-file tx.signed
// cardano-cli conway transaction submit --tx-file tx.signed

func getFileName(userID, suffix string) string {
	filename := path.Join(KeyPrefix, userID, userID + suffix)
	return filename
}

/**
 * GenerateWallet generates a new wallet with the given ID.
 * It creates the necessary payment and stake keys, and generates a payment address.
 * If the wallet already exists, it skips the generation.
 * @param ID The Discord GuildID or the UserID of the wallet to generate.
 * @return error If there was an error during the generation process.
 */
 func GenerateWallet(ID string) (*Keys, error) {
	logger.Record.Info("WALLET", "Checking if wallet exists...", getFileName(ID, PaymentKeySuffix))
	wallet, err := LoadWallet(ID)
	if err == nil {
		logger.Record.Info("WALLET", "Wallet already exists:", wallet.Address)
		return wallet, ErrWalletExists
	}

	logger.Record.Info("WALLET", "Wallet does not exist", "Generating new wallet...")
	err = os.MkdirAll(getFileName(ID, ""), 0755)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to create wallet directory: ", err)
		return nil, fmt.Errorf("failed to create wallet directory: %w", err)
	}
	// If the wallet does not exist, proceed with generation
	err = generatePaymentKey(ID)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to generate payment key: ", err)
		return nil, err
	}

	err = generateStakeKey(ID)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to generate stake key: ", err)
		return nil, err
	}

	err = generatePaymentAddress(ID)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to generate payment address: ", err)
		return nil, err
	}

	err = generateDelegationCertificate(ID)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to generate delegation certificate: ", err)
		return nil, err
	}

	wallet, err = LoadWallet(ID)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to load wallet: ", err)
		return nil, err
	}

	logger.Record.Info("WALLET", "Wallet generated successfully:", wallet.Address)

	return wallet, nil
}

func generatePaymentKey(ID string) error {
	paymentKey := getFileName(ID, PaymentKeySuffix)
	logger.Record.Info("WALLET", "Trying to generate payment key:", paymentKey)
	if _, err := os.Stat(paymentKey); os.IsNotExist(err) {
		// Generate the payment keys
		signingKey := getFileName(ID, SigningKeySuffix)
		paymentArgs := CommandArgs{
			"address",
			"key-gen",
			"--verification-key-file",
			paymentKey,
			"--signing-key-file",
			signingKey,
		}

		// cardano-cli address key-gen --verification-key-file payment.vkey --signing-key-file payment.skey
		_, err := Run(paymentArgs)
		if err != nil {
			logger.Record.Error("WALLET", "Failed to generate payment key: ", err)
			return err
		}
	} else {
		// File exists, skip generation
		return errors.New("Payment key file already exists, skipping generation.")
	}

	return nil
}

func generateStakeKey(ID string) error {
	stakeKey := getFileName(ID, StakeKeySuffix)
	if _, err := os.Stat(stakeKey); os.IsNotExist(err) {
		// Generate the stake keys
		signingStakeKey := getFileName(ID, StakeSigningKeySuffix)
		stakeArgs := []string{
			"conway",
			"stake-address",
			"key-gen",
			"--verification-key-file",
			stakeKey,
			"--signing-key-file",
			signingStakeKey,
		}

		// cardano-cli conway stake-address key-gen --verification-key-file stake.vkey --signing-key-file stake.skey
		_, err = Run(stakeArgs)
		if err != nil {
			logger.Record.Error("WALLET", "Failed to generate stake key: ", err)
			return err
		}
	} else {
		return errors.New("stake key file already exists, skipping generation")
	}

	return nil
}

func generatePaymentAddress(ID string) (error) {
	address := getFileName(ID, AddressSuffix)
	if _, err := os.Stat(address); os.IsNotExist(err) {
		paymentKey := getFileName(ID, PaymentKeySuffix)
		stakeKey := getFileName(ID, StakeKeySuffix)
		// Generate the payment address
		addressArgs := []string{
			"address",
			"build",
			"--payment-verification-key-file",
			paymentKey,
			"--stake-verification-key-file",
			stakeKey,
			"--mainnet",
			"--out-file",
			address,
		}

		// cardano-cli address build --payment-verification-key-file payment.vkey --stake-verification-key-file stake.vkey --mainnet --out-file payment.addr
		_, err := Run(addressArgs)
		if err != nil {
			logger.Record.Error("WALLET", "Failed to generate payment address: ", err)
			return err
		}
	} else {
		return errors.New("payment address file already exists, skipping generation")
	}

	return nil
}

func generateDelegationCertificate(ID string) error {
	stakeKey := getFileName(ID, StakeKeySuffix)
	delegationCert := getFileName(ID, DelegationCertificateSuffix)
	if _, err := os.Stat(delegationCert); os.IsNotExist(err) {
		// Generate the delegation certificate
		delegationArgs := []string{
			"conway",
			"stake-address",
			"stake-delegation-certificate",
			"--stake-verification-key-file",
			stakeKey,
			"--stake-pool-id",
			"pool19peeq2czwunkwe3s70yuvwpsrqcyndlqnxvt67usz98px57z7fk", // PREEB
			"--out-file",
			delegationCert,
		}

		// cardano-cli conway stake-address stake-delegation-certificate --stake-verification-key-file stake.vkey --stake-pool-id pool17navl486tuwjg4t95vwtlqslx9225x5lguwuy6ahc58x5dnm9ma --out-file delegation.cert
		_, err := Run(delegationArgs)
		if err != nil {
			logger.Record.Error("WALLET", "Failed to generate delegation certificate: ", err)
			return err
		}
	}
	return nil
}

func LoadWallet(ID string) (*Keys, error) {
	paymentKey := getFileName(ID, PaymentKeySuffix)
	signingPaymentKey := getFileName(ID, SigningKeySuffix)
	stakeKey := getFileName(ID, StakeKeySuffix)
	signingStakeKey := getFileName(ID, StakeSigningKeySuffix)
	address := getFileName(ID, AddressSuffix)
	delegationCert := getFileName(ID, DelegationCertificateSuffix)
	
	safePaymentKey, err := readAndEncryptKey(paymentKey)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to read and encrypt payment key file: ", err)
		return nil, err
	}

	safeSigningPaymentKey, err := readAndEncryptKey(signingPaymentKey)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to read and encrypt signing payment key file: ", err)
		return nil, err
	}
	safeStakeKey, err := readAndEncryptKey(stakeKey)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to read and encrypt stake key file: ", err)
		return nil, err
	}
	safeSigningStakeKey, err := readAndEncryptKey(signingStakeKey)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to read and encrypt signing stake key file: ", err)
		return nil, err
	}

	safeDelegationCert, err := readAndEncryptKey(delegationCert)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to read and encrypt delegation certificate file: ", err)
		return nil, err
	}

	addressData, err := os.ReadFile(address)
	if err != nil {
		return nil, fmt.Errorf("failed to read address file: %s", err)
	}

	wallet := &Keys{
		Address:         		string(addressData),
		PaymentKey:      		string(safePaymentKey),
		SigningPaymentKey: 		string(safeSigningPaymentKey),
		StakeKey:        		string(safeStakeKey),
		SigningStakeKey: 		string(safeSigningStakeKey),
		DelegationCertificate:  string(safeDelegationCert),
	}

	return wallet, nil
}

func readAndEncryptKey(keyPath string) (string, error) {
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		return "", fmt.Errorf("key file does not exist: %s", keyPath)
	}
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		logger.Record.Error("WALLET", "Failed to read key file: ", err)
		return "", fmt.Errorf("failed to read key file: %s", err)
	}

	safeKey, err := db.Encrypt(string(keyData))
	if err != nil {
		logger.Record.Error("WALLET", "Failed to encrypt key: ", err)
		return "", fmt.Errorf("failed to encrypt key: %s", err)
	}

	return string(safeKey), nil
}