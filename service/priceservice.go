package service

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"errors"
)

type PriceService interface {
	GetHistoricalPriceInFor(to, from string, date time.Time) (float64, error)
	GetPriceInFor(toCurrency string, fromCurrency string) (float64, error)
}

func NewCryptoComparePriceService(apiKey, baseUrl string) PriceService {
	return &cryptoComparePriceService{
		apiKey:  apiKey,
		baseUrl: baseUrl,
	}
}

// CryptoCompare's HTTP Implementation
type cryptoComparePriceService struct {
	apiKey  string
	baseUrl string
}

func (me *cryptoComparePriceService) GetPriceInFor(to string, from string) (float64, error) {
	var value = 0.0
	url := me.baseUrl + "/data/price?fsym=" + from + "&tsyms=" + to + "&api_key=" + me.apiKey
	response, err := me.handleResponse(url)
	if err != nil {
		return value, err
	}
	value, ok := response[to].(float64)
	if !ok {
		return value, errors.New("can't convert response (" + fmt.Sprintf("%v", response) + ") " + err.Error())
	}
	return value, nil
}

func (me *cryptoComparePriceService) GetHistoricalPriceInFor(to, from string, date time.Time) (float64, error) {
	var value = 0.0
	u, err := url.Parse(me.baseUrl + "/data/pricehistorical")
	q := u.Query()
	q.Set("fsym", from)
	q.Set("tsyms", to)
	q.Set("ts", strconv.FormatInt(date.Unix(), 10))
	q.Set("api_key", me.apiKey)
	u.RawQuery = q.Encode()

	response, err := me.handleResponse(u.String())
	if err != nil {
		return value, err
	}

	res := response[from].(map[string]interface{})
	value, ok := res[to].(float64)
	if !ok {
		return value, errors.New("can't convert response (" + fmt.Sprintf("%v", response) + ") " + err.Error())
	}
	return value, nil
}

func (me *cryptoComparePriceService) handleResponse(url string) (map[string]interface{}, error) {
	var response map[string]interface{}

	resp, err := http.Get(url)
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return response, errors.New("server returned an unexpected answer: " + resp.Status)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(body, &response)
	return response, err
}
