// Package crypto implements the Telink BLE mesh cryptographic primitives
// used by the SAL Pixie protocol.
//
// All operations use a "reversed AES" convention where key, plaintext,
// and ciphertext bytes are reversed before and after standard AES-128-ECB.
// This reversal applies to every AES operation: login, session key
// derivation, and per-packet AES-CCM.
//
// # Login Handshake
//
// The client generates 8 random bytes (randA), computes an encrypted
// request via [LoginRequest], writes it to CHAR_PAIR, and reads the
// response via [ParseLoginResponse] to obtain randB.
//
// # Session Key
//
// [DeriveSessionKey] combines the mesh credentials with both login
// nonces (randA, randB). Note the key/plaintext assignment is the
// opposite of the login request — this asymmetry is the most common
// implementation mistake.
//
// # Packet Encryption
//
// Commands are encrypted with [Encrypt] using AES-CCM (CBC-MAC then
// CTR mode). Notifications are decrypted with [Decrypt]. The nonce
// construction differs between commands and notifications — see
// [CommandNonce] and [NotificationNonce].
package crypto
