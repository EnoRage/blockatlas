package cmd

import (
	"github.com/spf13/cobra"
	"github.com/trustwallet/blockatlas/marketdata"
)

var syncCmd = &cobra.Command{
	Use:   "sync-markets",
	Short: "Sync all markets prices and rates",
	Args:  cobra.NoArgs,
	Run:   syncMarketData,
}

func syncMarketData(cmd *cobra.Command, args []string) {
	marketdata.InitRates(Storage)
	marketdata.InitMarkets(Storage)
	<-make(chan bool)
}

func init() {
	rootCmd.AddCommand(syncCmd)
}