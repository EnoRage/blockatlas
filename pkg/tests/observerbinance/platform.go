// +build observerbinance

package observerbinance

import (
	"context"
	"fmt"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/trustwallet/blockatlas/api"
	"github.com/trustwallet/blockatlas/pkg/blockatlas"
	"github.com/trustwallet/blockatlas/pkg/ginutils"
	"github.com/trustwallet/blockatlas/pkg/logger"
	"github.com/trustwallet/blockatlas/storage"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Init(specificPlatform blockatlas.Platform) map[string]blockatlas.BlockAPI {
	BlockAPIs := make(map[string]blockatlas.BlockAPI)

	platformList := make([]blockatlas.Platform, 0)
	platformList = append(platformList, specificPlatform)
	for _, platform := range platformList {
		handle := platform.Coin().Handle
		apiURL := fmt.Sprintf("%s.api", handle)

		if !viper.IsSet(apiURL) {
			continue
		}
		if viper.GetString(apiURL) == "" {
			continue
		}

		p := logger.Params{
			"platform": handle,
			"coin":     platform.Coin(),
		}

		err := platform.Init()
		if err != nil {
			logger.Error("Failed to initialize API", err, p)
		}

		if blockAPI, ok := platform.(blockatlas.BlockAPI); ok {
			BlockAPIs[handle] = blockAPI
		}
	}
	return BlockAPIs
}

func runPlatform(cache storage.Storage, platform blockatlas.Platform) {
	var (
		port,
		sg gin.HandlerFunc
	)

	sg = sentrygin.New(sentrygin.Options{})

	logger.InitLogger()
	Init(platform)

	engine := gin.New()

	engine.Use(ginutils.CheckReverseProxy, sg)
	engine.Use(ginutils.CORSMiddleware())
	engine.OPTIONS("/*path", ginutils.CORSMiddleware())

	engine.GET("/", api.GetRoot)
	engine.GET("/status", func(c *gin.Context) {
		ginutils.RenderSuccess(c, map[string]interface{}{
			"status": true,
		})
	})

	api.MakeMetricsRoute(engine)
	api.LoadPlatforms(engine)

	logger.Info("Loading observer API")
	observerAPI := engine.Group("/observer/v1")
	api.SetupObserverAPI(observerAPI, &cache)

	signalForExit := make(chan os.Signal, 1)

	signal.Notify(signalForExit,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	srv := &http.Server{
		Addr:    ":8420",
		Handler: engine,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			logger.Fatal("Application failed", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	defer func() {
		if err := srv.Shutdown(ctx); err != nil {
			logger.Fatal("Server Shutdown: ", err)
		}
	}()

	logger.Info("Running application", logger.Params{"bind": port})

	stop := <-signalForExit
	logger.Info("Stop signal Received", stop)
	logger.Info("Waiting for all jobs to stop")
}
