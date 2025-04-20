package cardano

import (
	"cardano-valley/pkg/db"
	"cardano-valley/pkg/logger"
	"errors"
	"fmt"
	"os"
)

type (
	Wallet struct {
		Address string `bson:"address,omitempty"`
		PaymentKey string `bson:"payment_key,omitempty"`
		SigningPaymentKey string `bson:"signing_payment_key,omitempty"`
		StakeKey string `bson:"stake_key,omitempty"`
		SigningStakeKey string `bson:"signing_stake_key,omitempty"`
		DelegationCertificate string `bson:"delegation_certificate,omitempty"`
	}
)

var (
	KeyPrefix = "tmp/"
	PaymentKeySuffix = ".vkey"
	SigningKeySuffix = ".skey"
	StakeKeySuffix = "_stake.vkey"
	StakeSigningKeySuffix = "_stake.skey"
	AddressSuffix = ".addr"
	DelegationCertificateSuffix = "_delegation.cert"

	ERR_WALLET_EXISTS_ERROR = errors.New("Wallet already exists")
)

/**
 * GenerateWallet generates a new wallet with the given ID.
 * It creates the necessary payment and stake keys, and generates a payment address.
 * If the wallet already exists, it skips the generation.
 * @param ID The Discord GuildID or the UserID of the wallet to generate.
 * @return error If there was an error during the generation process.
 */
 func GenerateWallet(ID string) (*Wallet, error) {
	logger.Record.Info("WALLET", "Checking if wallet exists...", ID)
	wallet, err := LoadWallet(ID)
	if err == nil {
		logger.Record.Info("WALLET", "Wallet already exists:", wallet.Address)
		return wallet, ERR_WALLET_EXISTS_ERROR
	}

	logger.Record.Info("WALLET", "Wallet does not exist", "Generating new wallet...")
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

func generatePaymentKey(filename string) error {
	// Check if the payment key file already exists
	// If it does, skip the generation
	// If it doesn't, generate the payment key
	paymentKey := fmt.Sprintf("%s%s%s", KeyPrefix, filename, PaymentKeySuffix)
	if _, err := os.Stat(paymentKey); os.IsNotExist(err) {
		// Generate the payment keys
		signingKey := fmt.Sprintf("%s%s%s", KeyPrefix, filename, SigningKeySuffix)
		paymentArgs := CommandArgs{
			"address",
			"key-gen",
			"--verification-key-file",
			paymentKey,
			"--signing-key-file",
			signingKey,
		}

		//cardano-cli address key-gen --verification-key-file payment.vkey --signing-key-file payment.skey
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

func generateStakeKey(filename string) error {
	// Check if the payment key file already exists
	// If it does, skip the generation
	// If it doesn't, generate the payment key
	stakeKey := fmt.Sprintf("%s%s%s", KeyPrefix, filename, StakeKeySuffix)
	if _, err := os.Stat(stakeKey); os.IsNotExist(err) {
		// Generate the stake keys
		signingStakeKey := fmt.Sprintf("%s%s%s", KeyPrefix, filename, StakeSigningKeySuffix)
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

func generatePaymentAddress(filename string) (error) {
	// Generate the stake keys
	address := fmt.Sprintf("%s%s%s", KeyPrefix, filename, AddressSuffix)
	if _, err := os.Stat(address); os.IsNotExist(err) {
		paymentKey := fmt.Sprintf("%s%s%s", KeyPrefix, filename, PaymentKeySuffix)
		stakeKey := fmt.Sprintf("%s%s%s", KeyPrefix, filename, StakeKeySuffix)
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

func generateDelegationCertificate(filename string) error {
	stakeKey := fmt.Sprintf("%s%s%s", KeyPrefix, filename, StakeKeySuffix)
	delegationCert := fmt.Sprintf("%s%s%s", KeyPrefix, filename, DelegationCertificateSuffix)
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

func LoadWallet(ID string) (*Wallet, error) {
	paymentKey := fmt.Sprintf("%s%s%s", KeyPrefix, ID, PaymentKeySuffix)
	signingPaymentKey := fmt.Sprintf("%s%s%s", KeyPrefix, ID, SigningKeySuffix)
	stakeKey := fmt.Sprintf("%s%s%s", KeyPrefix, ID, StakeKeySuffix)
	signingStakeKey := fmt.Sprintf("%s%s%s", KeyPrefix, ID, StakeSigningKeySuffix)
	address := fmt.Sprintf("%s%s%s", KeyPrefix, ID, AddressSuffix)
	delegationCert := fmt.Sprintf("%s%s%s", KeyPrefix, ID, DelegationCertificateSuffix)
	
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

	wallet := &Wallet{
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