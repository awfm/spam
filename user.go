package main

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/onflow/flow-go-sdk/crypto"
)

type User struct {
	cli     *client.Client
	priv    crypto.PrivateKey
	account *flow.Account
}

func NewRoot(cli *client.Client, hex string) (*User, error) {

	priv, err := crypto.DecodePrivateKeyHex(crypto.ECDSA_P256, hex)
	if err != nil {
		return nil, fmt.Errorf("could not decode private key: %w", err)
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
		return nil, fmt.Errorf("could not generate seed: %w", err)
	}

	priv, err := crypto.GeneratePrivateKey(crypto.ECDSA_P256, seed)
	if err != nil {
		return nil, fmt.Errorf("could not generate private key: %w", err)
	}

	pub := flow.NewAccountKey().
		FromPrivateKey(priv).
		SetHashAlgo(crypto.SHA3_256).
		SetWeight(flow.AccountKeyWeightThreshold)

	promise, err := root.RunCode(
		LoadCreation(pub),
	)
	if err != nil {
		return nil, fmt.Errorf("could not run code: %w", err)
	}

	address, err := promise.Address()
	if err != nil {
		return nil, fmt.Errorf("could not get address: %w", err)
	}

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

func (u *User) String() string {
	return u.Address().Short()
}

func (u *User) Address() flow.Address {
	return u.account.Address
}

func (u *User) Pub() *flow.AccountKey {
	return u.account.Keys[0]
}

func (u *User) Signer() crypto.Signer {
	return crypto.NewInMemorySigner(u.priv, u.Pub().HashAlgo)
}

func (u *User) Refresh() error {
	account, err := u.cli.GetAccount(context.Background(), u.Address())
	if err != nil {
		return fmt.Errorf("could not get account: %w", err)
	}
	u.account = account
	return nil
}

func (u *User) RunCode(load LoadFunc, signs ...AuthFunc) (*Promise, error) {

	code, err := load()
	if err != nil {
		return nil, fmt.Errorf("could not load code: %w", err)
	}

	txID, err := u.SendTransaction(code, signs...)
	if err != nil {
		return nil, fmt.Errorf("could not execute code: %w", err)
	}

	return NewPromise(u.cli, txID), nil
}

func (u *User) SendTransaction(code []byte, signs ...AuthFunc) (flow.Identifier, error) {

	err := u.Refresh()
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not refresh account: %w", err)
	}

	header, err := u.cli.GetLatestBlockHeader(context.Background(), false)
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not get latest block header: %w", err)
	}

	tx := flow.NewTransaction().
		SetScript(code).
		SetReferenceBlockID(header.ID).
		SetProposalKey(u.Address(), u.Pub().ID, u.Pub().SequenceNumber).
		SetPayer(u.Address())

	for _, sign := range signs {
		err = sign(tx)
		if err != nil {
			return flow.ZeroID, fmt.Errorf("could not sign transaction: %w", err)
		}
	}

	err = tx.SignEnvelope(u.Address(), u.Pub().ID, u.Signer())
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not sign envelope: %w", err)
	}

	err = u.cli.SendTransaction(context.Background(), *tx)
	if err != nil {
		return flow.ZeroID, fmt.Errorf("could not send transaction: %w", err)
	}

	return tx.ID(), nil
}
