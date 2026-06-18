package main

// Publisher tooling for the GhOSt store (ADR 0009). A store maintainer runs:
//
//   ghostd store-keygen                       # once: prints a keypair
//   ghostd store-sign index.json <privkey>    # writes index.json.sig
//
// The public key is pinned in each client's store config; ghostd verifies the
// index signature against it before trusting any entry.

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func storeKeygen() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keygen:", err)
		os.Exit(1)
	}
	fmt.Println("# GhOSt store keypair — keep the private key secret.")
	fmt.Println("publicKey: ", base64.StdEncoding.EncodeToString(pub))
	fmt.Println("privateKey:", base64.StdEncoding.EncodeToString(priv))
}

func storeSign(args []string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: ghostd store-sign <index.json> <privkey-base64>")
		os.Exit(2)
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read index:", err)
		os.Exit(1)
	}
	priv, err := base64.StdEncoding.DecodeString(args[1])
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		fmt.Fprintln(os.Stderr, "invalid private key (want base64 Ed25519)")
		os.Exit(1)
	}
	sig := ed25519.Sign(ed25519.PrivateKey(priv), data)
	out := args[0] + ".sig"
	if err := os.WriteFile(out, []byte(base64.StdEncoding.EncodeToString(sig)), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write sig:", err)
		os.Exit(1)
	}
	fmt.Println("wrote", out)
}
