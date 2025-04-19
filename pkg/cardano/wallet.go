package cardano

import (
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
	}
)

var (
	PaymentKeySuffix = ".vkey"
	SigningKeySuffix = ".skey"
	StakeKeySuffix = "_stake.vkey"
	StakeSigningKeySuffix = "_stake.skey"
	AddressSuffix = ".addr"
	DelegationCertificateSuffix = "_delegation.cert"
)

/**
 * GenerateWallet generates a new wallet with the given ID.
 * It creates the necessary payment and stake keys, and generates a payment address.
 * If the wallet already exists, it skips the generation.
 * @param ID The Discord GuildID or the UserID of the wallet to generate.
 * @return error If there was an error during the generation process.
 */
 func GenerateWallet(ID string) (*Wallet, error) {
	err := generatePaymentKey(ID)
	if err != nil {
		return nil, err
	}
	err = generateStakeKey(ID)
	if err != nil {
		return nil, err
	}
	err = generatePaymentAddress(ID)
	if err != nil {
		return nil, err
	}

	wallet, err := LoadWallet(ID)
	if err != nil {
		return nil, err
	}

	// // Encrypt and save the wallet to the db
	// paymentKey, err := db.Encrypt(wallet.PaymentKey)
	// if err != nil {
	// 	return nil, err
	// }
	// signingPaymentKey, err := db.Encrypt(wallet.SigningPaymentKey)
	// if err != nil {
	// 	return err
	// }
	// signingStakeKey, err := db.Encrypt(wallet.SigningStakeKey)
	// if err != nil {
	// 	return err
	// }
	// stakeKey, err := db.Encrypt(wallet.StakeKey)
	// if err != nil {
	// 	return err
	// }

	return wallet, nil
}

func generatePaymentKey(ID string) error {
	// Check if the payment key file already exists
	// If it does, skip the generation
	// If it doesn't, generate the payment key
	paymentKey := fmt.Sprintf("%s%s", ID, PaymentKeySuffix)
	if _, err := os.Stat(paymentKey); os.IsNotExist(err) {
		// Generate the payment keys
		signingKey := fmt.Sprintf("%s%s", ID, SigningKeySuffix)
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
	stakeKey := fmt.Sprintf("%s%s", filename, StakeKeySuffix)
	if _, err := os.Stat(stakeKey); os.IsNotExist(err) {
		// Generate the stake keys
		signingStakeKey := fmt.Sprintf("%s%s", filename, StakeSigningKeySuffix)
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
			return err
		}
	} else {
		return errors.New("Stake key file already exists, skipping generation.")
	}

	return nil
}

func generatePaymentAddress(filename string) (error) {
	// Generate the stake keys
	paymentKey := fmt.Sprintf("%s%s", filename, PaymentKeySuffix)
	if _, err := os.Stat(paymentKey); os.IsNotExist(err) {
		stakeKey := fmt.Sprintf("%s%s", filename, StakeKeySuffix)
		// Generate the payment address
		address := fmt.Sprintf("%s%s", filename, AddressSuffix)
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
			return err
		}
	} else {
		return errors.New("Payment address file already exists, skipping generation.")
	}

	return nil
}

func generateDelegationCertificate(filename string) error {
	stakeKey := fmt.Sprintf("%s%s", filename, StakeKeySuffix)
	delegationCert := fmt.Sprintf("%s%s", filename, DelegationCertificateSuffix)
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
		Run(delegationArgs)
	}
	return nil
}

func LoadWallet(ID string) (*Wallet, error) {
	paymentKey := fmt.Sprintf("%s.vkey", ID)
	signingPaymentKey := fmt.Sprintf("%s.skey", ID)
	stakeKey := fmt.Sprintf("%s.vkey", ID)
	signingStakeKey := fmt.Sprintf("%s.skey", ID)
	address := fmt.Sprintf("%s.addr", ID)

	if _, err := os.Stat(paymentKey); os.IsNotExist(err) {
		return nil, errors.New(fmt.Sprintf("payment key file does not exist: %s", paymentKey))
	}
	if _, err := os.Stat(signingPaymentKey); os.IsNotExist(err) {
		return nil, errors.New(fmt.Sprintf("signing key file does not exist: %s", signingPaymentKey))
	}
	if _, err := os.Stat(stakeKey); os.IsNotExist(err) {
		return nil, errors.New(fmt.Sprintf("stake key file does not exist: %s", stakeKey))
	}
	if _, err := os.Stat(signingStakeKey); os.IsNotExist(err) {
		return nil, errors.New(fmt.Sprintf("signing stake key file does not exist: %s", signingStakeKey))
	}
	if _, err := os.Stat(address); os.IsNotExist(err) {
		return nil, errors.New(fmt.Sprintf("address file does not exist: %s", address))
	}
	paymentKeyData, err := os.ReadFile(paymentKey)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to read payment key file: %s", err))
	}
	signingKeyData, err := os.ReadFile(signingPaymentKey)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to read signing key file: %s", err))
	}
	stakeKeyData, err := os.ReadFile(stakeKey)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to read stake key file: %s", err))
	}
	signingStakeKeyData, err := os.ReadFile(signingStakeKey)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to read signing stake key file: %s", err))
	}
	addressData, err := os.ReadFile(address)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to read address file: %s", err))
	}

	wallet := &Wallet{
		Address:         string(addressData),
		PaymentKey:      string(paymentKeyData),
		SigningPaymentKey: string(signingKeyData),
		StakeKey:        string(stakeKeyData),
		SigningStakeKey: string(signingStakeKeyData),
	}

	return wallet, nil
}