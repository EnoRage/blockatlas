// +build observerbinance

package observerbinance

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/trustwallet/blockatlas/config"
	"github.com/trustwallet/blockatlas/storage"
	"net/http"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
)

func setMockTrue(mock *HookResult) {
	defer mock.Unlock()
	mock.Lock()
	mock.isHooked = true
}

func setMockFalse(mock *HookResult) {
	defer mock.Unlock()
	mock.Lock()
	mock.isHooked = false
}

func getMockRes(mock *HookResult) bool {
	defer mock.Unlock()
	mock.Lock()
	res := mock.isHooked
	return res
}

func prepareRedis(t *testing.T) *storage.Storage {
	cache := storage.New()
	confPath, err := filepath.Abs("config.yml")
	if err != nil {
		t.Fatal(err)
	}
	config.LoadConfig(confPath)
	host := viper.GetString("storage.redis")
	err = cache.Init(host)
	if err != nil {
		t.Fatal(err)
	}
	ok := cache.Redis.FlushAll()
	if !ok {
		t.Fatal(err)
	}
	return cache
}

func POST(url string, data []byte) {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+"test")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
}

type HookResult struct {
	sync.RWMutex
	isHooked bool
}

func runMockServer(mock *HookResult, block int) {
	setMockFalse(mock)
	r := gin.New()
	v1 := r.Group("/binance/api/v1")
	{
		v1.GET("/blocks", func(c *gin.Context) {
			mocked := `{"blockArray":[{"blockHeight":`
			block++
			mocked2 := `,"blockHash":"A8E13B2A39754B4E21679A6C41216501718FB1EE0470159F822BE8FF92AB352D","txNum":0}]}`
			c.String(200, mocked+strconv.Itoa(block)+mocked2)
		})
		v1.GET("/txs", func(c *gin.Context) {
			mocked := `{"txNums":1,"txArray":[{"blockHeight":65745252,"txType":"TRANSFER","code":0,"confirmBlocks":0,"data":"","fromAddr":"bnb10psf6hay75r0qs4qld3dfm8euy2edjp0kcplu8","orderId":"","timeStamp":1580680624155,"toAddr":"bnb155svs6sgxe55rnvs6ghprtqu0mh69kehphsppd","txAge":57,"txAsset":"BNB","txFee":0.00037500,"txHash":"832DEFBD306237F53DB71CFBD7097B138ABF8B7B7FD378ED8F2E54C604A656D5","value":100.09862500,"memo":"","hasChildren":0,"subTxsDto":{"totalNum":0,"subTxDtoList":null}}]}`
			bh := c.Query("blockHeight")
			if bh == "65745220" {
				c.String(200, mocked)
				return
			}
			c.String(200, `{"txNums":0,"txArray":[]}`)
		})
		v1.GET("/tx", func(c *gin.Context) {
			mocked := `{"blockHeight":65745252,"txType":"TRANSFER","code":0,"confirmBlocks":7805,"data":"","fromAddr":"bnb10psf6hay75r0qs4qld3dfm8euy2edjp0kcplu8","orderId":"","timeStamp":1580680624155,"toAddr":"bnb155svs6sgxe55rnvs6ghprtqu0mh69kehphsppd","txAge":3212,"txAsset":"BNB","txFee":0.00037500,"txHash":"832DEFBD306237F53DB71CFBD7097B138ABF8B7B7FD378ED8F2E54C604A656D5","value":100.09862500,"memo":"","hasChildren":0,"subTxsDto":{"totalNum":0,"subTxDtoList":null}}`
			c.String(200, mocked)
		})
	}

	r.POST("/hook", func(c *gin.Context) {
		setMockTrue(mock)
		c.String(200, "")
	})

	r.Run()
}
