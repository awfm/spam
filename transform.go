package main

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/templates"
)

type TransformFunc func([]byte) []byte

func ReplaceImport(placeholder string, address flow.Address) TransformFunc {
	return func(code []byte) []byte {
		return bytes.ReplaceAll(code, []byte(fmt.Sprintf(" %s", placeholder)), []byte(fmt.Sprintf(" 0x%s", address.Hex())))
	}
}

func ReplaceRecipient(placeholder string, address flow.Address) TransformFunc {
	return func(code []byte) []byte {
		return bytes.ReplaceAll(code, []byte(fmt.Sprintf("getAccount(%s)", placeholder)), []byte(fmt.Sprintf("getAccount(0x%s)", address.Hex())))
	}
}

func ReplaceAmount(placeholder string, amount uint64) TransformFunc {
	return func(code []byte) []byte {
		return bytes.ReplaceAll(code, []byte(placeholder), []byte(strconv.FormatUint(uint64(amount), 10)+".0"))
	}
}

func DeployContract(pubs ...*flow.AccountKey) TransformFunc {
	return func(code []byte) []byte {
		code, _ = templates.CreateAccount(pubs, code)
		return code
	}
}
