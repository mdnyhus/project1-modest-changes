package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"strconv"
)

func main() {
	for i := 0; i < 20; i++ {
		keyPointer, _ := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
		privKey := *keyPointer
		pubByte, _ := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
		privByte, _ := x509.MarshalECPrivateKey(&privKey)
		pubStr := hex.EncodeToString(pubByte)
		privStr := hex.EncodeToString(privByte)
		fmt.Println("=======" + strconv.Itoa(i+1) + "=======")
		fmt.Println("PUBLIC: " + pubStr)
		fmt.Println("PRIVATE: " + privStr)
	}
}
