# TSS service

Threshold signature service provides the signature for core APPROVED operations. 
It works as a decentralized solution connected to the core for block timestamp bindings.

The TSS network requires a several parties launched by different validators that uses stored in the `rarimocore` 
Cosmos module party addresses and public keys to connect with each other and produce signature.


Fore more information visit [dev-docs](https://rarimo.gitlab.io/dev-docs/docs/developers/tss).

## Launch

Run command: `tss-svc migrate up && tss-svc run service`

Service requires the following configuration to be launched:

1. config.yaml 
   ```yaml
   log:
      disable_sentry: true
      level: debug
   
   db:
      url: "postgres://tss:tss@tss-1-db:5432/tss?sslmode=disable"
   
   listener:
      addr: :9000
   
   core:
      addr: tcp://validator:26657
   
   cosmos:
      addr: validator:9090
   
   session:
      start_block: 15
      start_session_id: 1
   
   swagger:
      addr: :1313
      enabled: true
   
   chain:
      chain_id: "rarimo_201411-2"
      coin_name: "urmo"    
   ```
   

2. EVN: 
   - ConfigPathEnv   = "KV_VIPER_FILE"
   - VaultPathEnv    = "VAULT_PATH"
   - VaultTokenEnv   = "VAULT_TOKEN"
   - VaultMountPath  = "MOUNT_PATH"
   - VaultSecretPath = "SECRET_PATH"


3. Vault secret (JSON)
   ```json
     {
       "tls": true,
       "data": "",
       "pre": "pre-generated-secret-data",
       "account": "rarimo-account-private-key-hex-leading-0x",
       "trial": "trial-ecdsa-private-key-hex-leading-0x"
     }
   ```

## Security

- [Halborn Audit](./Rarimo_Threshold_Signature_Module_Golang_Security_Assessment_Report_Halborn_Final.pdf)