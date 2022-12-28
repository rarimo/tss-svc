package config

import (
	"os"

	vault "github.com/hashicorp/vault/api"
)

const (
	VaultPathEnv   = "VAULT_PATH"
	VaultTokenEnv  = "VAULT_TOKEN"
	VaultMountPath = "MOUNT_PATH"
)

func (c *config) Vault() *vault.KVv2 {
	return c.vault.Do(func() interface{} {
		conf := vault.DefaultConfig()
		conf.Address = os.Getenv(VaultPathEnv)

		client, err := vault.NewClient(conf)
		if err != nil {
			panic(err)
		}

		client.SetToken(os.Getenv(VaultTokenEnv))

		return client.KVv2(os.Getenv(VaultMountPath))
	}).(*vault.KVv2)
}
