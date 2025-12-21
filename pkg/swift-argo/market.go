package swiftargo

import (
	"time"

	"github.com/rxtech-lab/argo-trading/pkg/marketdata"
)	

type MarketDownloader struct {
	helper        MarketDownloaderHelper
	provider      string
	writer        string
	dataFolder    string
	polygonApiKey string
}

type MarketDownloaderHelper interface {
	OnDownloadProgress(current, total float64, message string)
}

func NewMarketDownloader(helper MarketDownloaderHelper, provider string, writer string, dataFolder string, polygonApiKey string) *MarketDownloader {
	return &MarketDownloader{
		helper:        helper,
		provider:      provider,
		writer:        writer,
		dataFolder:    dataFolder,
		polygonApiKey: polygonApiKey,
	}
}

func (m *MarketDownloader) Download(ticker string, from, to string, interval string) error {
	client, err := marketdata.NewClient(marketdata.ClientConfig{
		ProviderType:  marketdata.ProviderType(m.provider),
		WriterType:    marketdata.WriterType(m.writer),
		DataPath:      m.dataFolder,
		PolygonApiKey: m.polygonApiKey,
	}, func(current, total float64, message string) {
		m.helper.OnDownloadProgress(current, total, message)
	})
	if err != nil {
		return err
	}

	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		return err
	}

	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		return err
	}

	timespan := marketdata.Timespan(interval)

	err = client.Download(marketdata.DownloadParams{
		Ticker:     ticker,
		StartDate:  fromTime,
		EndDate:    toTime,
		Timespan:   timespan.Timespan(),
		Multiplier: timespan.Multiplier(),
	})
	if err != nil {
		return err
	}

	return nil
}
