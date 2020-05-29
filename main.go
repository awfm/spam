package main

import (
	"flag"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/onflow/flow-ft/contracts"
	"github.com/onflow/flow-go-sdk/client"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
)

const (
	FungibleTokenPlaceholder = "0x02"
	FlowTokenPlaceholder     = "0x03"

	MintRecipientPlaceholder     = "0x03"
	TransferRecipientPlaceholder = "0x04"

	AllowedAmountPlaceholder  = "100.0"
	MintAmountPlaceholder     = "10.0"
	TransferAmountPlaceholder = "10.0"

	SetupAccountTransaction   = "https://raw.githubusercontent.com/onflow/flow-ft/master/transactions/setup_account.cdc"
	MintTokensTransaction     = "https://raw.githubusercontent.com/onflow/flow-ft/master/transactions/mint_tokens.cdc"
	TransferTokensTransaction = "https://raw.githubusercontent.com/onflow/flow-ft/master/transactions/transfer_tokens.cdc"
)

func main() {

	rpc := flag.String("rpc", "127.0.0.1:3569", "RPC server address of the access node")
	hex := flag.String("hex", "", "hex-encoded private key for the service account")
	num := flag.Uint("num", 100, "number of user accounts to initialize")
	tps := flag.Uint("tps", 10, "number of transaction per second to send")

	flag.Parse()

	zerolog.TimestampFunc = func() time.Time { return time.Now().UTC() }
	log := zerolog.New(os.Stderr)

	cli, err := client.New(*rpc, grpc.WithInsecure())
	if err != nil {
		log.Fatal().Err(err).Msg("could not connect to access node")
	}

	log.Info().Msg("connected to access node")

	root, err := NewRoot(cli, *hex)
	if err != nil {
		log.Fatal().Err(err).Msg("could not load root user")
	}

	log.Info().Str("address", root.String()).Msg("root user loaded")

	cache := NewCache()

	iface, err := root.RunCode(
		ApplyTransforms(
			LoadBytes(contracts.FungibleToken()),
			DeployContract(),
		),
		AddAuthorizer(root.Address()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not deploy fungible token contract")
	}

	ifaceAddress, err := iface.Address()
	if err != nil {
		log.Fatal().Err(err).Msg("could not get fungible token contract address")
	}

	log.Info().Str("address", ifaceAddress.Hex()).Msg("fungible token contract deployed")

	token, err := root.RunCode(
		ApplyTransforms(
			LoadBytes(contracts.FlowToken(ifaceAddress.Hex())),
			ReplaceImport(FungibleTokenPlaceholder, ifaceAddress),
			ReplaceAmount(AllowedAmountPlaceholder, 184467440737),
			DeployContract(root.Pub()),
		),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not deploy flow token contract")
	}

	tokenAddress, err := token.Address()
	if err != nil {
		log.Fatal().Err(err).Msg("could not get flow token contract address")
	}

	log.Info().Str("address", tokenAddress.Hex()).Msg("flow token contract deployed")

	var users []*User
	for i := uint(0); i < *num; i++ {

		user, err := NewRandom(cli, root)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create random user")
		}

		log := log.With().Str("address", user.String()).Logger()

		log.Info().Msg("user generated")

		setup, err := user.RunCode(
			ApplyTransforms(
				LoadRemote(
					cache,
					SetupAccountTransaction,
				),
				ReplaceImport(FungibleTokenPlaceholder, ifaceAddress),
				ReplaceImport(FlowTokenPlaceholder, tokenAddress),
			),
			AddAuthorizer(user.Address()),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("could not submit setup account transaction")
		}

		err = setup.Error()
		if err != nil {
			log.Fatal().Err(err).Msg("could not set up account")

		}

		log.Info().Msg("account set up")

		amount := uint64(1000000)

		mint, err := user.RunCode(
			ApplyTransforms(
				LoadRemote(
					cache,
					MintTokensTransaction,
				),
				ReplaceImport(FungibleTokenPlaceholder, ifaceAddress),
				ReplaceImport(FlowTokenPlaceholder, tokenAddress),
				ReplaceRecipient(MintRecipientPlaceholder, user.Address()),
				ReplaceAmount(MintAmountPlaceholder, amount),
			),
			AddAuthorizer(tokenAddress),
			SignPayload(tokenAddress, 0, root.Signer()),
		)
		if err != nil {
			log.Fatal().Err(err).Msg("could not submit mint tokens transaction")
		}

		err = mint.Error()
		if err != nil {
			log.Fatal().Err(err).Msg("could not mint tokens")
		}

		log.Info().Uint64("amount", amount).Msg("tokens minted")

		users = append(users, user)

	}

	var mut sync.Mutex
	interval := time.Second / time.Duration(*tps)
	for {

		time.Sleep(interval)

		if len(users) < 2 {
			continue
		}

		index := rand.Intn(len(users))
		last := len(users) - 1

		mut.Lock()
		users[index], users[last] = users[last], users[index]
		mut.Unlock()

		sender := users[last]
		users = users[:last]
		receiver := users[rand.Intn(len(users))]

		log := log.With().Str("sender", sender.String()).Str("receiver", receiver.String()).Logger()

		go func() {
			defer func() {
				mut.Lock()
				users = append(users, sender)
				mut.Unlock()
			}()

			tx, err := sender.RunCode(
				ApplyTransforms(
					LoadRemote(
						cache,
						TransferTokensTransaction,
					),
					ReplaceImport(FungibleTokenPlaceholder, ifaceAddress),
					ReplaceImport(FlowTokenPlaceholder, tokenAddress),
					ReplaceRecipient(TransferRecipientPlaceholder, receiver.Address()),
					ReplaceAmount(TransferAmountPlaceholder, 1),
				),
				AddAuthorizer(sender.Address()),
			)
			if err != nil {
				log.Error().Err(err).Msg("could not submit transfer tokens transaction")
				return
			}

			err = tx.Error()
			if err != nil {
				log.Error().Err(err).Msg("could not transfer tokens")
				return
			}

			log.Info().Msg("tokens transferred")
		}()
	}
}
