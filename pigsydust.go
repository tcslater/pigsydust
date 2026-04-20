// Package pigsydust implements the SAL Pixie / Telink BLE mesh protocol.
//
// It provides primitives for authenticating with a mesh (login + session key
// derivation), building encrypted command packets (AES-CCM), parsing
// notifications, and a high-level [Client] that manages session state on top
// of a user-provided [Transport].
//
// The protocol is implemented per the reference at
// pigsydust-py/docs/PROTOCOL-REFERENCE.md.
//
// # Overview
//
// A client connects to a single mesh node, authenticates with a shared mesh
// name and password, and sends encrypted commands. Any node relays broadcast
// traffic to the rest of the mesh. See [Client] for the high-level API, or
// the subpackages for lower-level building blocks:
//
//   - [pigsydust/crypto]: reversed AES, login, nonces, AES-CCM
//   - [pigsydust/protocol]: opcodes, vendor IDs, characteristic UUIDs
//   - [pigsydust/command]: command builders (on/off, groups, LED, schedules)
//   - [pigsydust/schedule]: schedule/alarm record encoding and timezone math
//
// BLE transports are provided by separate modules (e.g. pigsydust/ble).
package pigsydust
