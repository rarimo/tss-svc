---
layout: default
title: TSS overview
---

# TSS overview

For information about launch and configuration check the [`README file`](./README.md).

To perform cross-chain transfers, all operations should be signed with ECDSA secp256k1 threshold (t-n) signature.
This signature is produced by core multi-sig services depending on the core validated state.
All public signature parameters (including public key) should be defined and stored on the core system.

During parties work they should connect to the core to receive the new events of operation entry creation.
That operation will be put into the mempool sorted by operation timestamp.
After that some set of operations will be extracted from the mempool and signed by parties.
After producing the signature the confirmation message will be sent to the core with the information about the signed operation.

Core operation entry contains the information about some data to sign. Operation can have the following types:
- transfer operation (transfer token from one chain to another)
- change key by adding party
- change key by removing party

----

## Protocol

### Timestamps

To reach the consensus and prevent system failures all steps should depend on any time source. Unfortunately, using timestamps is not possible because it can create lots of troubles with defining current time between parties, so we will use block number from core to define the time position of the current step. Every party should be connected to the core validator and synchronize their block number to use it like a timestamp in the whole process.
Also, all steps will have strongly defined time bounds in blocks. We will define the following time bounds:
- Waiting for the pool from proposer
- Waiting for the acceptances
- Waiting for the signing rounds

We are defining the constant bounds for every session stage so the result session duration is also constant.

### Catchup

Also, there will be such situations where some parties lost their connections or some requests have not reached them for some reasons.
To come back into the flow that parties can calculate current session id and session time bounds depending on setup information (start block and start id) and current id.
If some party can not continue participating in the current session it will request other parties about the session info and sleep until they finish the current session.
Current session will be finished after the session deadline (can be derived from steps time bounds).

### Preparing the pool

To perform the correct signature the parties should reach a consensus on what operations to sign in the current session.
To get an agreement on that the deterministic defined party should propose the pool before the signing process will be launched.

Let’s define the function `f(prev_sign, parties, session_id)` that accepts the last produced signature, parties set and the session id and produces the proposer of the next pool. Session id is an incremental value.
Every party will calculate that value and accept the pool only from the defined proposer. If the party has not received the pool from the proposer, it will catch up with the other parties and sleep until they finish that session.

### Accepting the pool
After receiving the pool every party shares with other parties their acceptances - the ECDSA signed pool hash with the party private key. For processing the next step parties should receive minimum t exceptions.
If party has not received a minimum amount of acceptances, it will catch up with the other parties and sleep until they finish that session.

----

## Communication

Every party should have the following public endpoints:
- Get current session information (pool, proposer, steps with time bounds, accepted parties list, signed parties list, status) - for example /session/current
- Get session by id information. (pool, proposer, steps with time bounds, accepted parties list, signed parties list, status) - for example /session/{id}

Every party should have the following protected endpoints (reachable by other parties with their ECDSA signature)
-  Submit request.

From the core side, every party should have an opportunity to fetch the last block information and last produced signature and also, submit the confirmation message.

### Requests
Every submit request will contain next values: sender (derived from signature), request type, request body. Parties will parse request body using request type descriptor. Supported request types are:
- Pool proposal
- Pool acceptance
- Sign steps operations
- Key regeneration steps operations

----

## First key generation

The first key generation requires a launched core with a preconfigured set of parties. Parties information contains the ip address of the party service, Rarimo core account address and trial public key. Also, all parties have the raw flag set as true.

1. Parties start working in keygen mode and produce the ECDSA private shares and common ECDSA public key.
2. After key generation, every party submits their ECDSA public key to the core and the corresponding raw flag will be set as false.
3. After all parties submit their public keys, the common public key will be generated.

Until at least one party has as enabled raw flag there will not be possible to submit any confirmation.

### Sign session flow

1. Parties calculate the next proposer using deterministic function f(prev_signature, parties, session_id).
2. Proposer selects the pool to sign
3. Proposer shares the pool between all parties
4. Parties share the acceptance of that pool (the signature using party private key)
5. Parties that have received minimum t acceptances start the threshold signature process.
6. After the signature process finishes every party can send the confirmation transaction to the core.

### Reshare session flow

1. Parties calculate the next proposer using deterministic function `f(prev_signature, parties, session_id)`.
2. Proposer checks if reshare needed (if parties set have changed)
3. Proposer shares the proposal request to reshare keys
4. Parties share the acceptance of that pool (the signature using party private key)
5. If all parties accepted the proposal they start the keygen process.
6. After the keygen process finishes parties from old set start a sign session to sign new key.
7. After the sign process finishes parties from old set start a sign session to sign the operation.
8. After the signature process finishes every party can send the new operation (change parties set) and confirmation transactions to the core.

----

## Offenders

The `rarimocore` module functionality provides parties with opportunity to report about malicious party behaviour.
It works in the following way:

- Check that sender and offender is active parties
- Check that violation has not been created yet
- Save violation report.
- Iterate over existing reports, increment party violations if there are more than threshold violations for same session.
- If party violations reaches `maxViolationsCount` (in params) then change party status to `Frozen` and set `UpdateIsRequired` flag.

Currently, parties support only following violations:

- Party rejects the submitted request (offline or other reason)
- Proposal submitter is not current session proposer.
- Received invalid proposal (for some reasons).
- Received invalid acceptance (for some reasons).
- Received sign request from not a signer.
- Invalid sign request (wrong data to sign)
- Can not apply keygen/sign request (tss-lib returns an error)
