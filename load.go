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

func LoadRemote(cache *Cache, url string) LoadFunc {
	return func() ([]byte, error) {

		code, ok := cache.Get(url)
		if ok {
			return code, nil
		}

		res, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("could not get response from server: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode > 299 {
			return nil, fmt.Errorf("could not retrieve contract data (status: %s)", res.Status)
		}

		code, err = ioutil.ReadAll(res.Body)
		if err != nil {
			return nil, fmt.Errorf("could not read contract data: %w", err)
		}

		cache.Add(url, code)

		return code, nil
	}
}

func LoadBytes(data []byte) LoadFunc {
	return func() ([]byte, error) {
		return data, nil
	}
}

func ApplyTransforms(load LoadFunc, transforms ...TransformFunc) LoadFunc {
	return func() ([]byte, error) {

		code, err := load()
		if err != nil {
			return nil, fmt.Errorf("could not load code: %w", err)
		}

		for _, transform := range transforms {
			code = transform(code)
		}

		return code, nil
	}
}
