# TSS service

Threshold signature service provides the signature for core APPROVED operations. 
It works as a decentralized solution connected to the core for block timestamp bindings. 
The TSS network requires a several parties launched by different validators that uses stored in the `rarimocore` 
Cosmos module party addresses and public keys to connect with each other and produce signature. 

## Launch

Service can be launched in two modes:
- keygen mode `tss run keygen`
- default mode `tss run service`

Service requires the following configuration to be launched:

1. config.yaml
    ```yaml
    log:
      disable_sentry: true
      level: debug
    
    db:
      url: "postgres://tss:tss@localhost:5432/tss?sslmode=disable"
    
    listener:
      addr: :9000
    
    core:
      addr: tcp://localhost:26657
    
    cosmos:
      addr: localhost:9090
    
    session:
      start_block: 500
      start_session_id: 1
    ```
   
2. Vault EVN: 
   - VaultPathEnv    = "VAULT_PATH"
   - VaultTokenEnv   = "VAULT_TOKEN"
   - VaultMountPath  = "MOUNT_PATH"
   - VaultSecretPath = "SECRET_PATH"

3. Vault secret
    - data
    - pre
    - account
    - trial
    - tls
