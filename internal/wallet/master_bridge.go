package wallet

// MasterKeyFromMnemonic exposes a stable 32B master key derivation from
// a BIP-39 mnemonic for use by other packages.
func MasterKeyFromMnemonic(mnemonic string) ([]byte, error) {
	return masterKeyFromMnemonic(mnemonic)
}
