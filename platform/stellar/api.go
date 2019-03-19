package stellar

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/stellar/go/clients/horizon"
	"github.com/stellar/go/xdr"
	"net/http"
	"strconv"
	"sync"
	"time"
	"trustwallet.com/blockatlas/models"
	"trustwallet.com/blockatlas/util"
)

var client = horizon.Client{
	HTTP: http.DefaultClient,
}

func Setup(router gin.IRouter) {
	router.Use(util.RequireConfig("stellar.api"))
	router.Use(func(c *gin.Context) {
		client.URL = viper.GetString("stellar.api")
		c.Next()
	})
	router.GET("/:address", getTransactions)
}

func getTransactions(c *gin.Context) {
	acc, err := client.LoadAccount(c.Param("address"))
	if apiError(c, err) {
		return
	}

	ctxt, _ := context.WithTimeout(context.Background(), time.Second)

	var txMut sync.Mutex
	var txs []models.LegacyTx

	err = client.StreamTransactions(ctxt, acc.ID, nil, func(tx horizon.Transaction) {
		txMut.Lock()
		defer txMut.Unlock()

		if tx.ResultXdr == "" {
			return
		}
		if !tx.Successful {
			return
		}

		var envelope xdr.TransactionEnvelope
		err = xdr.SafeUnmarshalBase64(tx.EnvelopeXdr, &envelope)
		if err != nil {
			return
		}

		for _, op := range envelope.Tx.Operations {
			payment := op.Body.PaymentOp
			if payment == nil {
				continue
			}
			if payment.Asset.Type != xdr.AssetTypeAssetTypeNative {
				continue
			}
			legacy := models.LegacyTx{
				Id:          tx.Hash,
				BlockNumber: uint64(tx.Ledger),
				Timestamp:   strconv.FormatInt(tx.LedgerCloseTime.Unix(), 10),
				From:        tx.Account,
				To:          payment.Destination.Address(),
				Value:       strconv.FormatInt(int64(payment.Amount), 10),
				GasPrice:    strconv.FormatInt(int64(tx.FeePaid), 10),
				Coin:        148,
			}
			legacy.Init()
			txs = append(txs, legacy)
		}
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Wait for transaction stream to finish
	<-ctxt.Done()

	txMut.Lock()
	defer txMut.Unlock()
	c.JSON(http.StatusOK, models.Response {
		Total: len(txs),
		Docs:  txs,
	})
}

func apiError(c *gin.Context, err error) bool {
	if hErr, ok := err.(*horizon.Error); ok {
		if hErr.Problem.Type == "https://stellar.org/horizon-errors/bad_request" {
			c.String(http.StatusBadRequest, "Bad request!")
			return true
		} else {
			c.String(http.StatusBadRequest, hErr.Problem.Type)
			return true
		}
	}
	if err != nil {
		logrus.WithError(err).Warning("Stellar API request failed")
		c.String(http.StatusBadGateway, "Stellar API request failed")
		return true
	}
	return false
}
