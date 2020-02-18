package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/ProxeusApp/node-crypto-forex-rates/service"

	"github.com/ProxeusApp/proxeus-core/externalnode"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

const serviceID = "node-crypto-forex-rates"
const defaultServiceName = "Crypto to Fiat Forex Rates"
const defaultJWTSecret = "my secret"
const defaultServiceUrl = "127.0.0.1"
const defaultServicePort = "8011"
const defaultAuthkey = "auth"
const defaultProxeusUrl = "http://127.0.0.1:1323"

var (
	tokens             []string
	cryptoPriceService service.PriceService
)

type configData struct {
	FiatCurrency string
}

var configPage *template.Template

type configuration struct {
	proxeusUrl string
	serviceUrl string
	jwtsecret  string
	authtoken  string
}

var Config configuration

func main() {
	proxeusUrl := os.Getenv("PROXEUS_INSTANCE_URL")
	if len(proxeusUrl) == 0 {
		proxeusUrl = defaultProxeusUrl
	}
	serviceUrl := os.Getenv("SERVICE_URL")
	if len(serviceUrl) == 0 {
		serviceUrl = defaultServiceUrl
	}
	servicePort := os.Getenv("SERVICE_PORT")
	if len(servicePort) == 0 {
		servicePort = defaultServicePort
	}
	jwtsecret := os.Getenv("SERVICE_SECRET")
	if len(jwtsecret) == 0 {
		jwtsecret = defaultJWTSecret
	}
	serviceName := os.Getenv("SERVICE_NAME")
	if len(serviceName) == 0 {
		serviceName = defaultServiceName
	}
	Config = configuration{proxeusUrl: proxeusUrl, serviceUrl: serviceUrl, jwtsecret: jwtsecret, authtoken: defaultAuthkey}
	fmt.Println()
	fmt.Println("#######################################################")
	fmt.Println("# STARTING NODE - " + serviceName)
	fmt.Println("# listing on " + serviceUrl)
	fmt.Println("# connecting to " + proxeusUrl)
	fmt.Println("#######################################################")
	fmt.Println()

	tokens = []string{
		"ETH",
		"XES",
		"MKR",
	}

	cryptoPriceService = service.NewCryptoComparePriceService("API_KEY",
		"https://min-api.cryptocompare.com")

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.GET("/health", externalnode.Health)
	{
		g := e.Group("/node/:id")
		conf := middleware.DefaultJWTConfig
		conf.SigningKey = []byte(jwtsecret)
		conf.TokenLookup = "query:" + defaultAuthkey
		g.Use(middleware.JWTWithConfig(conf))

		g.GET("/config", config)
		g.POST("/config", setConfig)
		g.POST("/next", next)
		g.POST("/remove", externalnode.Nop)
		g.POST("/close", externalnode.Nop)
	}

	//External Node Specific Initialization
	parseTemplates()

	//Common External Node registration
	externalnode.Register(proxeusUrl, serviceName, serviceUrl+":"+servicePort, jwtsecret, "Converts Crypto to Firat currencies")
	err := e.Start("localhost:" + servicePort)
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
	fiatCurrency := getConfig(c).FiatCurrency

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

func config(c echo.Context) error {
	id, err := externalnode.NodeID(c)
	if err != nil {
		return err
	}
	conf := getConfig(c)
	var buf bytes.Buffer
	err = configPage.Execute(&buf, map[string]string{
		"Id":           id,
		"AuthToken":    c.QueryParam(Config.authtoken),
		"FiatCurrency": conf.FiatCurrency,
	})
	if err != nil {
		return err
	}
	return c.Stream(http.StatusOK, "text/html", &buf)
}

func setConfig(c echo.Context) error {
	conf := &configData{
		FiatCurrency: strings.TrimSpace(c.FormValue("FiatCurrency")),
	}
	if conf.FiatCurrency == "" {
		return c.String(http.StatusBadRequest, "empty currency")
	}

	err := externalnode.SetStoredConfig(c, Config.proxeusUrl, conf)
	if err != nil {
		return err
	}
	return config(c)
}

func getConfig(c echo.Context) *configData {
	jsonBody, err := externalnode.GetStoredConfig(c, Config.proxeusUrl)
	if err != nil {
		return &configData{
			FiatCurrency: "USD",
		}
	}

	config := configData{}
	if err := json.Unmarshal(jsonBody, &config); err != nil {
		fmt.Println(err)
		return nil
	}
	return &config
}

func parseTemplates() {
	var err error
	configPage, err = template.New("").Parse(configHTML)
	if err != nil {
		panic(err.Error())
	}
}

const configHTML = `
<!DOCTYPE html>
<html>
<body>
<form action="/node/{{.Id}}/config?auth={{.AuthToken}}" method="post">
Convert to Fiat currency: <input type="text" size="2" name="FiatCurrency" value="{{.FiatCurrency}}">
<input type="submit" value="Submit">
</form>
</body>
</html>
`
