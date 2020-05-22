package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
	"github.com/onflow/flow-go-sdk/templates"
)

type User struct {
	cli     *client.Client
	priv    crypto.PrivateKey
	account *flow.Account
}

func NewRoot(cli *client.Client, hex string) (*User, error) {

	priv, err := crypto.DecodePrivateKeyHex(crypto.ECDSA_P256, hex)
	if err != nil {
		return nil, fmt.Errorf("could not decode private hex key: %w", err)
	}

	address := flow.HexToAddress("01")
	account, err := cli.GetAccount(context.Background(), address)
	if err != nil {
		return nil, fmt.Errorf("could not get account: %w", err)
	}

	u := &User{
		cli:     cli,
		priv:    priv,
		account: account,
	}

	return u, nil
}

func NewRandom(cli *client.Client, root *User) (*User, error) {

	seed := make([]byte, crypto.MinSeedLength)
	_, err := rand.Read(seed)
	if err != nil {
		return nil, fmt.Errorf("could not generate random seed: %w", err)
	}

	priv, err := crypto.GeneratePrivateKey(crypto.ECDSA_P256, seed)
	if err != nil {
		return nil, fmt.Errorf("could not generate private key: %w", err)
	}

	pub := flow.NewAccountKey().
		FromPrivateKey(priv).
		SetHashAlgo(crypto.SHA3_256).
		SetWeight(flow.AccountKeyWeightThreshold)

	script, err := templates.CreateAccount([]*flow.AccountKey{pub}, nil)
	if err != nil {
		return nil, fmt.Errorf("could not generate account creation script: %w", err)
	}

	txID, err := root.ExecuteScript(script)
	if err != nil {
		return nil, fmt.Errorf("could not execute script: %w", err)
	}

	event, err := root.GetEvent(txID, flow.EventAccountCreated)
	if err != nil {
		return nil, fmt.Errorf("could not get transaction result: %w", err)
	}

	address := flow.AccountCreatedEvent(event).Address()
	account, err := cli.GetAccount(context.Background(), address)
	if err != nil {
		return nil, fmt.Errorf("could not get account: %w", err)
	}

	u := &User{
		cli:     cli,
		priv:    priv,
		account: account,
	}

	return u, nil
}

func (u *User) Address() flow.Address {
	return u.account.Address
}

func (u *User) Pub() *flow.AccountKey {
	return u.account.Keys[0]
}

func (u *User) ID() int {
	return u.Pub().ID
}

func (u *User) Seq() uint64 {
	return u.Pub().SequenceNumber
}

func (u *User) Algo() crypto.HashAlgorithm {
	return u.Pub().HashAlgo
}

func (u *User) Signer() crypto.Signer {
	return crypto.NewInMemorySigner(u.priv, u.Algo())
}

func (u *User) ExecuteScript(script []byte) (flow.Identifier, error) {

	header, err := u.cli.GetLatestBlockHeader(context.Background(), false)
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not get latest block header: %w", err)
	}

	tx := flow.NewTransaction().
		SetScript(script).
		SetReferenceBlockID(header.ID).
		SetProposalKey(u.Address(), u.ID(), u.Seq()).
		SetPayer(u.Address())

	err = tx.SignEnvelope(u.Address(), u.ID(), u.Signer())
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not sign envelope: %w", err)
	}

	err = u.cli.SendTransaction(context.Background(), *tx)
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not send transaction: %w", err)
	}

	u.Pub().SequenceNumber++

	return tx.ID(), nil
}

func (u *User) GetEvent(txID flow.Identifier, etype string) (flow.Event, error) {

Loop:
	for {

		time.Sleep(100 * time.Millisecond)

		result, err := u.cli.GetTransactionResult(context.Background(), txID)
		if err != nil {
			return flow.Event{}, fmt.Errorf("could not get result: %w", err)
		}

		switch result.Status {
		case flow.TransactionStatusUnknown, flow.TransactionStatusPending, flow.TransactionStatusExecuted:
			continue Loop
		case flow.TransactionStatusFinalized, flow.TransactionStatusSealed:
			// continue in same iteration
		default:
			return flow.Event{}, fmt.Errorf("invalid transaction status (%s)", result.Status)
		}

		for _, event := range result.Events {
			if event.Type == etype {
				return event, nil
			}
		}

		return flow.Event{}, fmt.Errorf("event type not found")
	}
}
