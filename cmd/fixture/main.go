// Emit deterministic crypto fixtures for cross-language verification.
package main

import (
	"fmt"

	"github.com/tcslater/pigsydust/command"
	"github.com/tcslater/pigsydust/crypto"
)

func main() {
	name := "Smart Light"
	pass := "400380"
	randA := [8]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	randB := [8]byte{0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18}
	gwMAC := [6]byte{0x00, 0x21, 0x4D, 0x5D, 0x26, 0x7D}
	sno := [3]byte{0x00, 0xB6, 0x08}

	sk := crypto.DeriveSessionKey(name, pass, randA, randB)
	fmt.Printf("sk=%x\n", sk)

	nonce := crypto.CommandNonce(gwMAC, sno)
	fmt.Printf("cmd_nonce=%x\n", nonce)

	cmd := command.OnOff(9, true)
	pt := cmd.Encode()
	fmt.Printf("plaintext=%x\n", pt)

	pkt := crypto.Encrypt(sk, nonce, sno, pt)
	fmt.Printf("packet=%x\n", pkt)
}
