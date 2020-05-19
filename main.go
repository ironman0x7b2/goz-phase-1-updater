package main

import (
	"flag"
	"github.com/avast/retry-go"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/07-tendermint/types"
	"github.com/cosmos/gaia/app"
	"github.com/iqlusioninc/relayer/cmd"
	"github.com/iqlusioninc/relayer/relayer"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"
)

var (
	srcChainID string
	dstChainID string
	pathName   string
	duration   time.Duration
)

func init() {
	flag.StringVar(&srcChainID, "src", "ibc0", "Source chain ID")
	flag.StringVar(&dstChainID, "dst", "ibc1", "Destination chain ID")
	flag.StringVar(&pathName, "path", "ibc0ibc1", "Path name")
	flag.DurationVar(&duration, "duration", 2*time.Minute, "Duration between consecutive updates")
	flag.Parse()
}

func main() {
	log.SetFlags(log.Lshortfile | log.LUTC | log.LstdFlags | log.Lmicroseconds)
	log.Printf("Source -> %s | Destination -> %s | Path -> %s | Duration -> %s\n", srcChainID, dstChainID, pathName, duration)

	appCodec, cdc := app.MakeCodecs()

	home, err := os.UserHomeDir()
	if err != nil {
		log.Panicln(err)
	}

	configDirectory := path.Join(home, ".relayer")

	file, err := ioutil.ReadFile(path.Join(configDirectory, "config/config.yaml"))
	if err != nil {
		log.Panicln(err)
	}

	var config cmd.Config
	if err := yaml.Unmarshal(file, &config); err != nil {
		log.Panicln(err)
	}

	timeout, err := time.ParseDuration(config.Global.Timeout)
	if err != nil {
		log.Panicln(err)
	}

	for _, chain := range config.Chains {
		if err := chain.Init(configDirectory, appCodec, cdc, timeout, true); err != nil {
			log.Panicln(err)
		}
	}

	src, err := config.Chains.Get(srcChainID)
	if err != nil {
		log.Panicln(err)
	}

	dst, err := config.Chains.Get(dstChainID)
	if err != nil {
		log.Panicln(err)
	}

	path, err := config.Paths.Get(pathName)
	if err != nil {
		log.Panicln(err)
	}

	go routine(path.Src, src, dst)
	go routine(path.Dst, dst, src)

	select {}
}

func routine(end *relayer.PathEnd, src, dst *relayer.Chain) {
	if err := src.AddPath(end.ClientID, end.ConnectionID, end.ChannelID, end.PortID, end.Order); err != nil {
		log.Panicln(err)
	}

	for {
		state, err := src.QueryClientState()
		if err != nil {
			log.Printf("Source -> %s | Destination -> %s | Error -> %s\n", src.ChainID, dst.ChainID, err)
			continue
		}

		timestamp := state.ClientState.(types.ClientState).GetLatestTimestamp()
		next := timestamp.Add(duration)

		log.Printf("Source -> %s | Destination -> %s | Timestamp -> %s | Next -> %s\n", src.ChainID, dst.ChainID, timestamp, next)

		if next.After(time.Now()) {
			select {
			case <-time.After(next.Sub(time.Now())):
			}
		}

		log.Printf("-------- | Source -> %s | Destination -> %s | --------\n", src.ChainID, dst.ChainID)

		header, err := dst.UpdateLiteWithHeader()
		if err != nil {
			log.Printf("Source -> %s | Destination -> %s | Error -> %s\n", src.ChainID, dst.ChainID, err)
			continue
		}

		msg := src.PathEnd.UpdateClient(header, src.MustGetAddress())

		txBytes, err := src.BuildAndSignTx([]sdk.Msg{msg})
		if err != nil {
			log.Printf("Source -> %s | Destination -> %s | Error -> %s\n", src.ChainID, dst.ChainID, err)
			continue
		}

		_ = retry.Do(func() error {
			res, err := src.BroadcastTxCommit(txBytes)
			if err != nil {
				log.Printf("Source -> %s | Destination -> %s | Error -> %s\n", src.ChainID, dst.ChainID, err)
				return err
			}

			if res.Code != 19 && res.Code != 20 && res.Code != 21 {
				log.Printf("Source -> %s | Destination -> %s | TxHash -> %s | "+
					"Code -> %d\n", src.ChainID, dst.ChainID, res.TxHash, res.Code)
				return nil
			}

			res, err = src.QueryTx(res.TxHash)
			if err != nil {
				log.Printf("Source -> %s | Destination -> %s | Error -> %s\n", src.ChainID, dst.ChainID, err)
				return err
			}

			log.Printf("Source -> %s | Destination -> %s | TxHash -> %s | "+
				"Code -> %d\n", src.ChainID, dst.ChainID, res.TxHash, res.Code)
			return nil
		}, retry.Attempts(100), retry.DelayType(retry.FixedDelay))
	}
}
