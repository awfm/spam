package main

import (
	"flag"
	"os"
	"time"

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

	FungibleTokenContract = "https://raw.githubusercontent.com/onflow/flow-ft/master/contracts/FungibleToken.cdc"
	FlowTokenContract     = "https://raw.githubusercontent.com/onflow/flow-ft/master/contracts/FlowToken.cdc"

	CreateMinterTransaction   = "https://raw.githubusercontent.com/onflow/flow-ft/master/transactions/create_minter.cdc"
	SetupAccountTransaction   = "https://raw.githubusercontent.com/onflow/flow-ft/master/transactions/setup_account.cdc"
	MintTokensTransaction     = "https://raw.githubusercontent.com/onflow/flow-ft/master/transactions/mint_tokens.cdc"
	TransferTokensTransaction = "https://raw.githubusercontent.com/onflow/flow-ft/master/transactions/transfer_tokens.cdc"
)

func main() {

	rpc := flag.String("rpc", "127.0.0.1:3569", "RPC server address of the access node")
	hex := flag.String("hex", "", "hex-encoded private key for the service account")
	// tps := flag.Uint("tps", 10, "number of transaction per second to send")

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

	fungible, err := root.RunCode(
		LoadRemote(
			FungibleTokenContract,
			DeployContract(),
		),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not deploy fungible token contract")
	}

	fungibleAddress, err := fungible.Address()
	if err != nil {
		log.Fatal().Err(err).Msg("could not get fungible token contract address")
	}

	log.Info().Str("address", fungibleAddress.Short()).Msg("fungible token contract deployed")

	flow, err := root.RunCode(
		LoadRemote(
			FlowTokenContract,
			ReplaceImport(FungibleTokenPlaceholder, fungibleAddress),
			ReplaceAmount(AllowedAmountPlaceholder, 184467440737),
			DeployContract(root.Pub()),
		),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not deploy flow token contract")
	}

	flowAddress, err := flow.Address()
	if err != nil {
		log.Fatal().Err(err).Msg("could not get flow token contract address")
	}

	log.Info().Str("address", flowAddress.Short()).Msg("flow token contract deployed")

	user, err := NewRandom(cli, root)
	if err != nil {
		log.Fatal().Err(err).Msg("could not create random user")
	}

	log.Info().Str("address", user.String()).Msg("random user generated")

	setup, err := user.RunCode(
		LoadRemote(
			SetupAccountTransaction,
			ReplaceImport(FungibleTokenPlaceholder, fungibleAddress),
			ReplaceImport(FlowTokenPlaceholder, flowAddress),
		),
		AddAuthorizer(user.Address()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not run setup account transaction")
	}

	err = setup.Error()
	if err != nil {
		log.Fatal().Err(err).Msg("could not set up account")

	}
	log.Info().Msg("random user account set up")

	mint, err := user.RunCode(
		LoadRemote(
			MintTokensTransaction,
			ReplaceImport(FungibleTokenPlaceholder, fungibleAddress),
			ReplaceImport(FlowTokenPlaceholder, flowAddress),
			ReplaceRecipient(MintRecipientPlaceholder, user.Address()),
			ReplaceAmount(MintAmountPlaceholder, 1000000),
		),
		AddAuthorizer(flowAddress),
		SignPayload(flowAddress, 0, root.Signer()),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("could not run mint tokens transaction")
	}

	err = mint.Error()
	if err != nil {
		log.Fatal().Err(err).Msg("could not mint tokens")
	}

	log.Info().Msg("random user tokens minted")

	os.Exit(0)
}
