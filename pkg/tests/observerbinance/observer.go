// +build observerbinance

package observerbinance

import (
	"context"
	"github.com/spf13/cast"
	"github.com/trustwallet/blockatlas/observer"
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
	"github.com/trustwallet/blockatlas/pkg/logger"
	"github.com/trustwallet/blockatlas/storage"
	"sync"
	"time"
)

func runObserver(cache storage.Storage, platform blockatlas.Platform) {
	BlockAPIs := Init(platform)
	if len(BlockAPIs) == 0 {
		logger.Fatal("No APIs to observe")
	}

	minInterval := cast.ToDuration("250ms")
	backlogTime := cast.ToDuration("3h")

	var wg sync.WaitGroup
	wg.Add(len(BlockAPIs))

	for _, api := range BlockAPIs {
		coin := api.Coin()
		blockTime := time.Duration(coin.BlockTime) * time.Millisecond
		pollInterval := blockTime / 4
		if pollInterval < minInterval {
			pollInterval = minInterval
		}

		// Stream incoming blocks
		var backlogCount int
		if coin.BlockTime == 0 {
			backlogCount = 50
			logger.Warn("Unknown block time", logger.Params{"coin": coin.ID})
		} else {
			backlogCount = int(backlogTime / blockTime)
		}

		stream := observer.Stream{
			BlockAPI:     api,
			Tracker:      &cache,
			PollInterval: pollInterval,
			BacklogCount: backlogCount,
		}
		blocks := stream.Execute(context.Background())

		// Check for transaction events
		obs := observer.Observer{
			Storage: &cache,
			Coin:    coin.ID,
		}
		events := obs.Execute(blocks)

		// Dispatch events
		dispatcher := observer.Dispatcher{}
		go func() {
			dispatcher.Run(events)
			wg.Done()
		}()

		logger.Info("Observing", logger.Params{
			"coin":     coin,
			"interval": pollInterval,
			"backlog":  backlogCount,
		})
	}
	wg.Wait()
}
