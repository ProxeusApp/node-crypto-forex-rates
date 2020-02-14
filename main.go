package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"

	"github.com/ProxeusApp/node-crypto-forex-rates/service"

	"github.com/ProxeusApp/proxeus-core/externalnode"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

const serviceName = "node-crypto-forex-rates"
const jwtSecret = "my secret"
const serviceUrl = "127.0.0.1:8011"
const authKey = "auth"

var (
	tokens             []string
	fiatCurrency       string
	cryptoPriceService service.PriceService
)

func main() {
	fmt.Println()
	fmt.Println("#######################################################")
	fmt.Println("# STARTING NODE - " + serviceName + " #")
	fmt.Println("# on " + serviceUrl + " #")
	fmt.Println("#######################################################")
	fmt.Println()

	tokens = []string{
		"ETH",
		"XES",
		"MKR",
	}
	fiatCurrency = "USD"

	cryptoPriceService = service.NewCryptoComparePriceService("API_KEY",
		"https://min-api.cryptocompare.com")

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.GET("/health", externalnode.Health)
	{
		g := e.Group("/node/:id")
		conf := middleware.DefaultJWTConfig
		conf.SigningKey = []byte(jwtSecret)
		conf.TokenLookup = "query:" + authKey
		g.Use(middleware.JWTWithConfig(conf))

		g.GET("/config", externalnode.Nop)
		g.POST("/config", externalnode.Nop)
		g.POST("/next", next)
		g.POST("/remove", externalnode.Nop)
		g.POST("/close", externalnode.Nop)
	}

	//Common External Node registration
	externalnode.Register(serviceName, serviceUrl, jwtSecret, "Converts currencies")
	err := e.Start(serviceUrl)
	if err != nil {
		log.Printf("[%s][run] err: %s", serviceName, err.Error())
	}
}

var errAssetNotFoundInRequest = errors.New("asset not found in request")

func next(c echo.Context) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	for _, asset := range tokens {
		if response[asset] == nil {
			log.Printf("asset %s not found. skipping, won't get price. err: %s", asset, errAssetNotFoundInRequest.Error())
			continue
		}
		ratio, err := cryptoPriceService.GetPriceInFor(asset, fiatCurrency)
		if err != nil {
			return err
		}
		valInt, ok := big.NewFloat(0).SetString(response[asset].(string))
		if !ok {
			return fmt.Errorf("could not convert value for asset: %s", asset)
		}
		response[fmt.Sprintf("%s_%s", fiatCurrency, asset)] = valInt.Mul(valInt, big.NewFloat(ratio)).String()
	}

	return c.JSON(http.StatusOK, response)
}
