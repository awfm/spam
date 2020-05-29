package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/client"
)

type Promise struct {
	sync.Mutex
	done   chan struct{}
	cli    *client.Client
	txID   flow.Identifier
	result *flow.TransactionResult
}

func NewPromise(cli *client.Client, txID flow.Identifier) *Promise {
	p := &Promise{
		done: make(chan struct{}),
		cli:  cli,
		txID: txID,
	}
	go p.seal()
	return p
}

func (p *Promise) seal() {
	timeout := time.NewTimer(20 * time.Second)

Loop:
	for {
		select {
		case <-p.done:
			break Loop
		case <-time.After(100 * time.Millisecond):
			p.check()
		case <-timeout.C:
			close(p.done)
			break Loop
		}
	}
}

func (p *Promise) check() {
	result, err := p.cli.GetTransactionResult(context.Background(), p.txID)
	if err != nil {
		p.result = &flow.TransactionResult{Error: err}
		return
	}
	if result.Error != nil || result.Status == flow.TransactionStatusSealed {
		p.result = result
		close(p.done)
	}
}

func (p *Promise) Address() (flow.Address, error) {
	<-p.done
	if p.Error() != nil {
		return flow.Address{}, fmt.Errorf("could not execute transaction: %w", p.Error())
	}
	for _, event := range p.result.Events {
		if event.Type == flow.EventAccountCreated {
			creation := flow.AccountCreatedEvent(event)
			return creation.Address(), nil
		}
	}
	return flow.Address{}, fmt.Errorf("transaction didn't create account")
}

func (p *Promise) Error() error {
	<-p.done
	if p.result == nil {
		return fmt.Errorf("transaction was never sealed")
	}
	return p.result.Error
}
