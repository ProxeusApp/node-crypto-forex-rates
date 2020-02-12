package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/ProxeusApp/node-crypto-forex-rates/service"

	external "github.com/ProxeusApp/proxeus-core/externalnode"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

const serviceName = "node-crypto-forex-rates"
const jwtSecret = "my secret"
const serviceUrl = "127.0.0.1:8011"
const authKey = "auth"

type configData struct {
	FromCurrency string
	ToCurrency   string
}

var configPage *template.Template

func main() {
	fmt.Println()
	fmt.Println("#######################################################")
	fmt.Println("# STARTING NODE - " + serviceName + " #")
	fmt.Println("# on " + serviceUrl + " #")
	fmt.Println("#######################################################")
	fmt.Println()
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.GET("/health", external.Health)
	{
		g := e.Group("/node/:id")
		conf := middleware.DefaultJWTConfig
		conf.SigningKey = []byte(jwtSecret)
		conf.TokenLookup = "query:" + authKey
		g.Use(middleware.JWTWithConfig(conf))

		g.GET("/config", config)
		g.POST("/config", setConfig)
		g.POST("/next", next)
		g.POST("/remove", external.Nop)
		g.POST("/close", external.Nop)
	}

	//External Node Specific Initialization
	parseTemplates()

	//Common External Node registration
	external.Register(serviceName, serviceUrl, jwtSecret, "Converts currencies")
	e.Start(serviceUrl)
}

func next(c echo.Context) error {
	body, err := ioutil.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}
	conf := getConfig(c)

	ratio, err := convert(conf.FromCurrency, conf.ToCurrency)
	if err != nil {
		return err
	}

	re, err := regexp.Compile("[0-9]+ " + conf.FromCurrency)
	if err != nil {
		return err
	}

	var replaceErr error
	replaced := re.ReplaceAllFunc(body, func(r []byte) []byte {
		if replaceErr != nil {
			return r
		}
		d := strings.Split(string(r), " ")
		val := d[0]
		valInt, err := strconv.Atoi(val)
		if err != nil {
			replaceErr = err
			return r
		}
		return []byte(fmt.Sprintf("%.3f %s", ratio*float64(valInt), conf.ToCurrency))
	})
	if replaceErr != nil {
		return replaceErr
	}
	return c.String(http.StatusOK, string(replaced))
}

func convert(from, to string) (float64, error) {
	s := service.NewCryptoComparePriceService("API_KEY",
		"https://min-api.cryptocompare.com")
	return s.GetPriceInFor(to, from)
}

func config(c echo.Context) error {
	id, err := external.NodeID(c)
	if err != nil {
		return err
	}
	conf := getConfig(c)
	var buf bytes.Buffer
	err = configPage.Execute(&buf, map[string]string{
		"Id":           id,
		"AuthToken":    c.QueryParam(authKey),
		"FromCurrency": conf.FromCurrency,
		"ToCurrency":   conf.ToCurrency,
	})
	if err != nil {
		return err
	}
	return c.Stream(http.StatusOK, "text/html", &buf)
}

func setConfig(c echo.Context) error {
	conf := &configData{
		FromCurrency: strings.TrimSpace(c.FormValue("FromCurrency")),
		ToCurrency:   strings.TrimSpace(c.FormValue("ToCurrency")),
	}
	if conf.FromCurrency == "" || conf.ToCurrency == "" {
		return c.String(http.StatusBadRequest, "empty currency")
	}

	err := external.SetStoredConfig(c, conf)
	if err != nil {
		return err
	}
	return config(c)
}

func getConfig(c echo.Context) *configData {
	jsonBody, err := external.GetStoredConfig(c)
	if err != nil {
		return &configData{
			FromCurrency: "CHF",
			ToCurrency:   "XES",
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
Convert fom currency: <input type="text" size="2" name="FromCurrency" value="{{.FromCurrency}}">
to currency: <input type="text" size="2" name="ToCurrency" value="{{.ToCurrency}}"><br/>
<input type="submit" value="Submit">
</form>
</body>
</html>
`
