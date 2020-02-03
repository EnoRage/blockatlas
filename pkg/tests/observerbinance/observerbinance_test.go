// +build observerbinance

package observerbinance

import (
	"github.com/stretchr/testify/assert"
	"github.com/trustwallet/blockatlas/platform/binance"
	"testing"
	"time"
)

func TestObserverBinance(t *testing.T) {
	cache := prepareRedis(t)
	bp := binance.Platform{}
	mockBinance := HookResult{}
	go runMockServer(&mockBinance, 65745234)
	go runPlatform(*cache, &bp)
	time.Sleep(time.Second / 10)
	POST("http://localhost:8420/observer/v1/webhook/register",
		[]byte(`{"subscriptions":{"714":["bnb155svs6sgxe55rnvs6ghprtqu0mh69kehphsppd"]},"webhook":"http://localhost:8080/hook"}`))
	go runObserver(*cache, &bp)
	time.Sleep(time.Second * 3)
	if !getMockRes(&mockBinance) {
		t.Fatal("FAIL: timeout for hook. Was not hooked")
	}
	assert.Equal(t, getMockRes(&mockBinance), true)
}
