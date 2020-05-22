package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/onflow/flow-go-sdk"
	"github.com/onflow/flow-go-sdk/templates"
)

type LoadFunc func() ([]byte, error)

func LoadCreation(pub *flow.AccountKey) LoadFunc {
	return func() ([]byte, error) {
		return templates.CreateAccount([]*flow.AccountKey{pub}, nil)
	}
}

func LoadRemote(url string, transforms ...TransformFunc) LoadFunc {
	return func() ([]byte, error) {

		res, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("could not get response from server: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode > 299 {
			return nil, fmt.Errorf("could not retrieve contract data (status: %s)", res.Status)
		}

		code, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read contract data: %w", err)
		}

		for _, transform := range transforms {
			code = transform(code)
		}

		return code, nil
	}
}
