# TSS service

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Threshold signature service provides the signature for core APPROVED operations. 
It works as a decentralized solution connected to the core for block timestamp bindings.

The TSS network requires a several parties launched by different validators that uses stored in the `rarimocore` 
Cosmos module party addresses and public keys to connect with each other and produce signature.

Fore more information check [overview](./OVERVIEW.md).

## Launch

To become an active TSS you need to follow that steps:

1. Generate TSS account (it is recommended to generate the new one and do not use it in other services):
  ```shell
  rarimo-core keys add <key-name> --keyring-backend test --home=$TSS_HOME
  ```

Also, you need to parse mnemonic to get corresponding private key:
  ```shell
  rarimo-core tx rarimocore parse-mnemonic 'mnemonic phrase'
  ```

2. Pre-setup secret parameters, execute and save response JSON:
  ```shell
  tss-svc run paramgen
  ```

3. Generate trial ECDSA private key, execute and store results:
  ```shell
  tss-svc run prvgen
  ```

4. Setup the Vault service and create secret for your tss (type KV version 2). Secret should contain the following credentials:

* "data": "Leave empty"

* "pre": "Generated pre params JSON"

* "account": "Your Rarimo account hex key"

* "trial": "Generated Trial ECDSA private key hex"

JSON example:
  ```json
  {
    "tls": true,
    "data": "",
    "pre": "pre-generated-secret-data",
    "account": "rarimo-account-private-key-hex-leading-0x",
    "trial": "trial-ecdsa-private-key-hex-leading-0x"
  }
  ```

5. Create a configuration file (config.yaml) with the following structure:

  ```yaml
  log:
    disable_sentry: true
    level: debug

  ## PostreSQL connection

  db:
    url: "postgres://tss:tss@tss-2-db:5432/tss?sslmode=disable"

  ## Port to listen for incoming GRPC requests

  listener:
    addr: :9000

  ## Core connections

  core:
    addr: tcp://validator:26657

  cosmos:
    addr: validator:9090

  ## Session configuration (should be the same for all services accross the system)

  session:
    start_block: 15
    start_session_id: 1

  ## Swagger doc configuration

  swagger:
    addr: :1313
    enabled: false

  ## Chain configuration

  chain:
    chain_id: "rarimo_201411-2"
    coin_name: "urmo"
  ```

Set up host environment:
  ```yaml
    - name: KV_VIPER_FILE
    value: /config/config.yaml # is the path to your config file
    - name: VAULT_PATH
    value: http://vault-internal:8200 # your vault endpoint
    - name: VAULT_TOKEN
    value: "" # your vault token ("root"/"read/write")
    - name: MOUNT_PATH
    value: secret
    - name: SECRET_PATH
    value: tss1 # name of the secret path vault (type KV version 2)
  ```

6. To launch the service use the following command
  ```shell
  tss-svc migrate up && tss-svc run service
  ```

Example of docker-compose file:
  ```yaml
  tss-1:
    image: registry.github.com/rarimo/tss-svc:v1.0.3
    restart: on-failure
    depends_on:
      - tss-1-db
    ports:
      - "9001:9000"
      - "1313:1313"
    volumes:
      - ./config/tss/tss1.yaml:/config.yaml
    environment:
      - KV_VIPER_FILE=/config.yaml
      - VAULT_PATH=http://vault:8200
      - VAULT_TOKEN=dev-only-token
      - MOUNT_PATH=secret
      - SECRET_PATH=tss1
    entrypoint: sh -c "tss-svc migrate up && tss-svc run service"

  tss-1-db:
    image: postgres:13
    restart: unless-stopped
    environment:
      - POSTGRES_USER=tss
      - POSTGRES_PASSWORD=tss
      - POSTGRES_DB=tss
      - PGDATA=/pgdata
    volumes:
      - tss-1-data:/pgdata
  ```

7. Stake tokens to become an active party. Execute
  ```shell
  rarimo-core tx rarimocore stake [tss-account-addr] [tss url] [trial ECDSA pub key] --from $ADDRESS --chain-id rarimo-201411-2 --home=$RARIMO_HOME --keyring-backend=test --fees 0urmo --node=$RARIMO_NODE
  ```

You can stake tokens directly from your TSS account or by any other account that has enough funds.
Currently, to become an active party you will need exactly 100000000000urmo tokens (100000 RMO).

After some period your TSS will generate new keys with other active parties and become an active party.

To unstake token execute the following command (from tss account or delegator account):
```shell
rarimo-cored tx rarimocore unstake [tss-account-addr] --from $ADDRESS --chain-id rarimo-201411-2 --home=$RARIMO_HOME --keyring-backend=test --fees 0urmo --node=$RARIMO_NODE
```
