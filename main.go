package main

import (
	"flag"
	"os"
	"time"

	"github.com/onflow/flow-go-sdk/client"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
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

	root, err := NewRoot(cli, *hex)
	if err != nil {
		log.Fatal().Err(err).Msg("could not create root user")
	}

	for i := 0; i < 16; i++ {
		_, err := NewRandom(cli, root)
		if err != nil {
			log.Fatal().Err(err).Msg("could not create user account")
		}
	}

	os.Exit(0)
}
