package postgres

import (
	"context"
	"fmt"
	"github.com/jinzhu/gorm"
	"github.com/trustwallet/watchmarket/db/models"
	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmgorm"
	"strconv"
	"strings"
	"time"
)

const (
	rawBulkTickersInsert = `INSERT INTO tickers(updated_at,created_at,id,coin,coin_name,coin_type,token_id,image_url,change24h,currency,provider,value,last_updated,volume,market_cap,show_option) VALUES %s ON CONFLICT ON CONSTRAINT tickers_pkey DO UPDATE SET id = excluded.id, value = excluded.value, change24h = excluded.change24h, updated_at = excluded.updated_at, last_updated = excluded.last_updated, volume = excluded.volume, market_cap = excluded.market_cap`
)

func (i *Instance) AddTickers(tickers []models.Ticker, batchLimit uint, ctx context.Context) error {
	g := apmgorm.WithContext(ctx, i.Gorm)
	span, _ := apm.StartSpan(ctx, "AddTickers", "postgresql")
	defer span.End()
	batch := toTickersBatch(normalizeTickers(tickers), batchLimit)
	for _, b := range batch {
		err := bulkCreateTicker(g, b)
		if err != nil {
			return err
		}
	}
	return nil
}

func toTickersBatch(tickers []models.Ticker, sizeUint uint) [][]models.Ticker {
	size := int(sizeUint)
	resultLength := (len(tickers) + size - 1) / size
	result := make([][]models.Ticker, resultLength)
	lo, hi := 0, size
	for i := range result {
		if hi > len(tickers) {
			hi = len(tickers)
		}
		result[i] = tickers[lo:hi:hi]
		lo, hi = hi, hi+size
	}
	return result
}

func normalizeTickers(tickers []models.Ticker) []models.Ticker {
	tickersMap := make(map[string]models.Ticker)
	for _, ticker := range tickers {
		key := strconv.Itoa(int(ticker.Coin)) +
			ticker.CoinName + ticker.CoinType +
			ticker.TokenId + ticker.Currency +
			ticker.Provider
		if _, ok := tickersMap[key]; !ok {
			tickersMap[key] = ticker
		}
	}
	result := make([]models.Ticker, 0, len(tickersMap))
	for _, ticker := range tickersMap {
		result = append(result, ticker)
	}
	return result
}

func bulkCreateTicker(db *gorm.DB, dataList []models.Ticker) error {
	var (
		valueStrings []string
		valueArgs    []interface{}
	)

	for _, d := range dataList {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")

		valueArgs = append(valueArgs, time.Now())
		valueArgs = append(valueArgs, time.Now())
		valueArgs = append(valueArgs, d.ID)
		valueArgs = append(valueArgs, d.Coin)
		valueArgs = append(valueArgs, d.CoinName)
		valueArgs = append(valueArgs, d.CoinType)
		valueArgs = append(valueArgs, d.TokenId)
		valueArgs = append(valueArgs, d.ImageUrl)
		valueArgs = append(valueArgs, d.Change24h)
		valueArgs = append(valueArgs, d.Currency)
		valueArgs = append(valueArgs, d.Provider)
		valueArgs = append(valueArgs, d.Value)
		valueArgs = append(valueArgs, d.LastUpdated)
		valueArgs = append(valueArgs, d.Volume)
		valueArgs = append(valueArgs, d.MarketCap)
		valueArgs = append(valueArgs, d.ShowOption)
	}

	smt := fmt.Sprintf(rawBulkTickersInsert, strings.Join(valueStrings, ","))
	if err := db.Exec(smt, valueArgs...).Error; err != nil {
		return err
	}

	return nil
}

func (i *Instance) GetTickersByQueries(tickerQueries []models.TickerQuery, ctx context.Context) ([]models.Ticker, error) {
	db := apmgorm.WithContext(ctx, i.Gorm)
	var ticker []models.Ticker
	for _, tq := range tickerQueries {
		db = db.Or("coin = ? AND token_id = ?", tq.Coin, tq.TokenId)
	}
	if err := db.Find(&ticker).Error; err != nil {
		return nil, err
	}
	return ticker, nil
}

func (i *Instance) GetTickers(coin uint, tokenId string, ctx context.Context) ([]models.Ticker, error) {
	g := apmgorm.WithContext(ctx, i.Gorm)
	var ticker []models.Ticker
	if err := g.Where("coin = ? AND token_id = ?", coin, tokenId).
		Find(&ticker).Error; err != nil {
		return nil, err
	}
	return ticker, nil
}

func (i *Instance) GetAllTickers(ctx context.Context) ([]models.Ticker, error) {
	g := apmgorm.WithContext(ctx, i.Gorm)
	var tickers []models.Ticker
	if err := g.Find(&tickers).Error; err != nil {
		return nil, err
	}
	return tickers, nil
}
