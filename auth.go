package main

import (
	"fmt"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/crypto"
)

type AuthFunc func(*flow.Transaction) error

func AddAuthorizer(address flow.Address) AuthFunc {
	return func(tx *flow.Transaction) error {
		_ = tx.AddAuthorizer(address)
		return nil
	}
}

func SignPayload(address flow.Address, id int, signer crypto.Signer) AuthFunc {
	return func(tx *flow.Transaction) error {
		err := tx.SignPayload(address, id, signer)
		if err != nil {
			return fmt.Errorf("could not sign envelope: %w", err)
		}
		return nil
	}
}
