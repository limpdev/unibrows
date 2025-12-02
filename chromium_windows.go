//go:build windows

package unibrows

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"

	"unibrows/crypto"

	"github.com/tidwall/gjson"
)

func (c *chromium) getMasterKeyOS() ([]byte, error) {
	localStatePath := filepath.Join(c.profilePath, "..", "Local State")

	content, err := os.ReadFile(localStatePath)
	if err != nil {
		return nil, err
	}

	encryptedKey := gjson.GetBytes(content, "os_crypt.encrypted_key")
	if !encryptedKey.Exists() {
		return nil, fmt.Errorf("encrypted_key not found in Local State")
	}

	key, err := base64.StdEncoding.DecodeString(encryptedKey.String())
	if err != nil {
		return nil, err
	}

	// The key is prefixed with 'DPAPI', which we need to remove.
	return crypto.DecryptWithDPAPI(key[5:])
}
