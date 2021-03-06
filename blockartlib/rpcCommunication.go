/*

This file is part of the blockartlib package, and contains all the structs required for RPC calls between blockartlib and an
ink-miner. It is placed in this file for cleaner organization of code.

*/

package blockartlib

import "crypto/ecdsa"

type AddShapeArgs struct {
	// ink-miner only sees internal representation of shapes, conversion is all done by blockartlib before RPC call
	ShapeMeta   ShapeMeta
	ValidateNum uint8
}

type AddShapeReply struct {
	OpHash       string
	BlockHash    string
	InkRemaining uint32

	// RPC errors are all cast to a ServerError
	// So, store actual error here; nil indicates no error
	Error error
}

type GetSvgStringArgs struct {
	OpHash string
}

type GetSvgStringReply struct {
	SvgString string

	// RPC errors are all cast to a ServerError
	// So, store actual error here; nil indicates no error
	Error error
}

type OpenCanvasArgs struct {
	Priv ecdsa.PrivateKey
	Pub  ecdsa.PublicKey
}

type OpenCanvasReply struct {
	CanvasSettings CanvasSettings
}

type DeleteShapeArgs struct {
	ValidateNum uint8
	ShapeHash   string
}

type DeleteShapeReply struct {
	InkRemaining uint32

	// RPC errors are all cast to a ServerError
	// So, store actual error here; nil indicates no error
	Error error
}

type GetShapesReply struct {
	ShapeHashes []string

	// RPC errors are all cast to a ServerError
	// So, store actual error here; nil indicates no error
	Error error
}

type GetChildrenReply struct {
	BlockHashes []string

	// RPC errors are all cast to a ServerError
	// So, store actual error here; nil indicates no error
	Error error
}
