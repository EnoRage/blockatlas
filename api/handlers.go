package api

import (
	"github.com/chenjiandongx/ginprom"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
	"github.com/trustwallet/blockatlas/pkg/ginutils"
	"github.com/trustwallet/blockatlas/pkg/metrics"
	"net/http"
)

// @Summary Get Transactions
// @ID tx_v1
// @Description Get transactions from the address
// @Accept json
// @Produce json
// @Tags Transactions
// @Param coin path string true "the coin name" default(tezos)
// @Param address path string true "the query address" default(tz1WCd2jm4uSt4vntk4vSuUWoZQGhLcDuR9q)
// @Failure 500 {object} ginutils.ApiError
// @Router /v1/{coin}/{address} [get]
func makeTxRouteV1(router gin.IRouter, api blockatlas.Platform) {
	makeTxRoute(router, api, "/:address")
}

// @Summary Get Transactions
// @ID tx_v2
// @Description Get transactions from the address
// @Accept json
// @Produce json
// @Tags Transactions
// @Param coin path string true "the coin name" default(tezos)
// @Param address path string true "the query address" default(tz1WCd2jm4uSt4vntk4vSuUWoZQGhLcDuR9q)
// @Success 200 {object} blockatlas.TxPage
// @Failure 500 {object} ginutils.ApiError
// @Router /v2/{coin}/transactions/{address} [get]
func makeTxRouteV2(router gin.IRouter, api blockatlas.Platform) {
	makeTxRoute(router, api, "/transactions/:address")
}

func makeTxRoute(router gin.IRouter, api blockatlas.Platform, path string) {
	var txAPI blockatlas.TxAPI
	var tokenTxAPI blockatlas.TokenTxAPI
	txAPI, _ = api.(blockatlas.TxAPI)
	tokenTxAPI, _ = api.(blockatlas.TokenTxAPI)

	if txAPI == nil && tokenTxAPI == nil {
		return
	}

	router.GET(path, func(c *gin.Context) {
		address := c.Param("address")
		if address == "" {
			emptyPage(c)
			return
		}
		token := c.Query("token")

		var txs []blockatlas.Tx
		var err error
		switch {
		case token == "" && txAPI != nil:
			txs, err = txAPI.GetTxsByAddress(address)
		case token != "" && tokenTxAPI != nil:
			txs, err = tokenTxAPI.GetTokenTxsByAddress(address, token)
		default:
			emptyPage(c)
			return
		}

		if err != nil {
			errResp := ginutils.ErrorResponse(c)
			switch {
			case err == blockatlas.ErrInvalidAddr:
				errResp.Params(http.StatusBadRequest, "Invalid address")
			case err == blockatlas.ErrNotFound:
				errResp.Params(http.StatusNotFound, "No such address")
			case err == blockatlas.ErrSourceConn:
				errResp.Params(http.StatusServiceUnavailable, "Lost connection to blockchain")
			}
			errResp.Render()
			return
		}

		page := make(blockatlas.TxPage, 0)
		for _, tx := range txs {
			if tx.Direction != "" {
				goto AddTx
			}
			tx.Direction = blockatlas.DirectionOutgoing
			if tx.To == address {
				tx.Direction = blockatlas.DirectionIncoming
				if tx.From == address {
					tx.Direction = blockatlas.DirectionSelf
				}
			}
		AddTx:
			page = append(page, tx)
		}
		if len(page) > blockatlas.TxPerPage {
			page = page[0:blockatlas.TxPerPage]
		}
		page.Sort()
		ginutils.RenderSuccess(c, &page)
	})
}

// @Summary Get Collections
// @ID collections_v2
// @Description Get all collections from the address
// @Accept json
// @Produce json
// @Tags Collections
// @Param coin path string true "the coin name" default(ethereum)
// @Param address path string true "the query address" default(0x5574Cd97432cEd0D7Caf58ac3c4fEDB2061C98fB)
// @Success 200 {object} blockatlas.CollectionPage
// @Failure 500 {object} ginutils.ApiError
// @Router /v2/{coin}/collections/{address} [get]
//TODO: remove once most of the clients will be updated (deadline: March 17th)
func oldMakeCollectionsRoute(router gin.IRouter, api blockatlas.Platform) {
	var collectionAPI blockatlas.CollectionAPI
	collectionAPI, _ = api.(blockatlas.CollectionAPI)

	if collectionAPI == nil {
		return
	}

	router.GET("/collections/:owner", func(c *gin.Context) {
		collections, err := collectionAPI.OldGetCollections(c.Param("owner"))
		if err != nil {
			ginutils.ErrorResponse(c).Message(err.Error()).Render()
			return
		}

		ginutils.RenderSuccess(c, collections)
	})
}

// @Summary Get Collections
// @ID collections_v3
// @Description Get all collections from the address
// @Accept json
// @Produce json
// @Tags Collections
// @Param coin path string true "the coin name" default(ethereum)
// @Param address path string true "the query address" default(0x5574Cd97432cEd0D7Caf58ac3c4fEDB2061C98fB)
// @Success 200 {object} blockatlas.CollectionPage
// @Failure 500 {object} ginutils.ApiError
// @Router /v3/{coin}/collections/{address} [get]
func makeCollectionsRoute(router gin.IRouter, api blockatlas.Platform) {
	var collectionAPI blockatlas.CollectionAPI
	collectionAPI, _ = api.(blockatlas.CollectionAPI)

	if collectionAPI == nil {
		return
	}

	router.GET("/collections/:owner", func(c *gin.Context) {
		collections, err := collectionAPI.GetCollections(c.Param("owner"))
		if err != nil {
			ginutils.ErrorResponse(c).Message(err.Error()).Render()
			return
		}

		ginutils.RenderSuccess(c, collections)
	})
}

// @Summary Get Collection
// @ID collection_v2
// @Description Get a collection from the address
// @Accept json
// @Produce json
// @Tags Collections
// @Param coin path string true "the coin name" default(ethereum)
// @Param owner path string true "the query address" default(0x0875BCab22dE3d02402bc38aEe4104e1239374a7)
// @Param collection_id path string true "the query collection" default(0x06012c8cf97bead5deae237070f9587f8e7a266d)
// @Success 200 {object} blockatlas.CollectionPage
// @Failure 500 {object} ginutils.ApiError
// @Router /v2/{coin}/collections/{owner}/collection/{collection_id} [get]
//TODO: remove once most of the clients will be updated (deadline: March 17th)
func oldMakeCollectionRoute(router gin.IRouter, api blockatlas.Platform) {
	var collectionAPI blockatlas.CollectionAPI
	collectionAPI, _ = api.(blockatlas.CollectionAPI)

	if collectionAPI == nil {
		return
	}

	router.GET("/collections/:owner/collection/:collection_id", func(c *gin.Context) {
		collectibles, err := collectionAPI.OldGetCollectibles(c.Param("owner"), c.Param("collection_id"))
		if err != nil {
			ginutils.ErrorResponse(c).Message(err.Error()).Render()
			return
		}

		ginutils.RenderSuccess(c, collectibles)
	})
}

// @Summary Get Collection
// @ID collection_v3
// @Description Get a collection from the address
// @Accept json
// @Produce json
// @Tags Collections
// @Param coin path string true "the coin name" default(ethereum)
// @Param owner path string true "the query address" default(0x0875BCab22dE3d02402bc38aEe4104e1239374a7)
// @Param collection_id path string true "the query collection" default(0x06012c8cf97bead5deae237070f9587f8e7a266d)
// @Success 200 {object} blockatlas.CollectionPage
// @Failure 500 {object} ginutils.ApiError
// @Router /v3/{coin}/collections/{owner}/collection/{collection_id} [get]
func makeCollectionRoute(router gin.IRouter, api blockatlas.Platform) {
	var collectionAPI blockatlas.CollectionAPI
	collectionAPI, _ = api.(blockatlas.CollectionAPI)

	if collectionAPI == nil {
		return
	}

	router.GET("/collections/:owner/collection/:collection_id", func(c *gin.Context) {
		collectibles, err := collectionAPI.GetCollectibles(c.Param("owner"), c.Param("collection_id"))
		if err != nil {
			ginutils.ErrorResponse(c).Message(err.Error()).Render()
			return
		}

		ginutils.RenderSuccess(c, collectibles)
	})
}

// @Summary Get Tokens
// @ID tokens
// @Description Get tokens from the address
// @Accept json
// @Produce json
// @Tags Transactions
// @Param coin path string true "the coin name" default(ethereum)
// @Param address path string true "the query address" default(0x5574Cd97432cEd0D7Caf58ac3c4fEDB2061C98fB)
// @Success 200 {object} blockatlas.CollectionPage
// @Failure 500 {object} ginutils.ApiError
// @Router /v2/{coin}/tokens/{address} [get]
func makeTokenRoute(router gin.IRouter, api blockatlas.Platform) {
	var tokenAPI blockatlas.TokenAPI
	tokenAPI, _ = api.(blockatlas.TokenAPI)

	if tokenAPI == nil {
		return
	}

	router.GET("/tokens/:address", func(c *gin.Context) {
		address := c.Param("address")
		if address == "" {
			emptyPage(c)
			return
		}

		tl, err := tokenAPI.GetTokenListByAddress(address)
		if err != nil {
			ginutils.ErrorResponse(c).Message(err.Error()).Render()
			return
		}

		ginutils.RenderSuccess(c, blockatlas.DocsResponse{Docs: tl})
	})
}

func MakeMetricsRoute(router gin.IRouter) {
	router.Use(metrics.PromMiddleware())
	m := router.Group("/metrics")
	m.Use(ginutils.TokenAuthMiddleware(viper.GetString("metrics.api_token")))
	m.GET("/", ginprom.PromHandler(promhttp.Handler()))
}

func emptyPage(c *gin.Context) {
	var page blockatlas.TxPage
	ginutils.RenderSuccess(c, &page)
}
