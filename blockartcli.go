package main

import (
	"fmt"
	"os"
	"./blockartlib"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: go run blockartcli.go [miner ip:port] [server ip:port] [pubKey] [privKey]")
	}

	minerAddr := os.Args[1]
	serverAddr := os.Args[2]
	pubKeyArg := os.Args[3]
	privKeyArg := os.Args[4]

	pubKeyStr, err := hex.DecodeString(pubKey)
	if err != nil {
		panic(err)
	}
	pubKeyParsed, err := x509.ParsePKIXPublicKey(pubKeyStr)
	if err != nil {
		panic(err)
	}
	pubKey := *pubKeyParsed.(*ecdsa.PublicKey)

	privKeyStr, err := hex.DecodeString(priv)
	if err != nil {
		panic(err)
	}
	privKeyParsed, err := x509.ParseECPrivateKey(privKeyStr)
	if err != nil {
		panic(err)
	}
	privKey := *privKeyParsed

	canvas, settings, err := blockartlib.OpenCanvas(minerAddr, privKey)
}
