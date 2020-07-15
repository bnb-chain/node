package pub

import (
	metricsPkg "github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
)

// Metrics contains metrics exposed by this package.
type Metrics struct {
	// Height of last published message
	PublicationHeight metricsPkg.Gauge

	// Size of publication queue
	PublicationQueueSize metricsPkg.Gauge

	// Time between publish this and the last block.
	// Should be (approximate) blocking + abci + publication time
	PublicationBlockIntervalMs metricsPkg.Gauge

	// Time used to collect block information
	CollectBlockTimeMs metricsPkg.Gauge

	// Time used to collect orderbook information
	CollectOrderBookTimeMs metricsPkg.Gauge

	// Time used to publish everything in a block
	// Should be (approximate) sum of folllowing Times
	PublishTotalTimeMs metricsPkg.Gauge
	// Time used to publish order & trade
	PublishTradeAndOrderTimeMs metricsPkg.Gauge
	// Time used to publish orderbook
	PublishOrderbookTimeMs metricsPkg.Gauge
	// Time used to publish accounts
	PublishAccountTimeMs metricsPkg.Gauge
	// Time used to publish blockfee
	PublishBlockfeeTimeMs metricsPkg.Gauge
	// Time	used to publish transfer
	PublishTransfersTimeMs metricsPkg.Gauge
	// Time	used to publish block
	PublishBlockTimeMs metricsPkg.Gauge
	// Time	used to publish sideProposal
	PublishSideProposalTimeMs metricsPkg.Gauge

	// num of trade
	NumTrade metricsPkg.Gauge
	// num of order
	NumOrder metricsPkg.Gauge
	// num of orderbook levels
	NumOrderBook metricsPkg.Gauge
	// num of account balance changes
	NumAccounts metricsPkg.Gauge
	// num of transfer
	NumTransfers metricsPkg.Gauge
}

// PrometheusMetrics returns Metrics build using Prometheus client library.
func PrometheusMetrics() *Metrics {
	return &Metrics{
		PublicationHeight: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "height",
			Help:      "Height of last published messages",
		}, []string{}),
		PublicationQueueSize: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "queue_size",
			Help:      "Size of publication queue",
		}, []string{}),
		PublicationBlockIntervalMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "block_interval",
			Help:      "How often we publish a block (ms)",
		}, []string{}),
		CollectBlockTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "collect_block",
			Help:      "Time to collect block info",
		}, []string{}),
		CollectOrderBookTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "collect_orderbook",
			Help:      "Time to collect order book info",
		}, []string{}),
		PublishTotalTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "total_pub_time",
			Help:      "Time to publish everything within a block (ms)",
		}, []string{}),
		PublishTradeAndOrderTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "tradeorder_pub_time",
			Help:      "Time to publish trade and order (ms)",
		}, []string{}),
		PublishOrderbookTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "orderbook_pub_time",
			Help:      "Time to publish orderbook (ms)",
		}, []string{}),
		PublishAccountTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "account_pub_time",
			Help:      "Time to publish account (ms)",
		}, []string{}),
		PublishBlockfeeTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "blockfee_pub_time",
			Help:      "Time to publish block fee (ms)",
		}, []string{}),
		PublishTransfersTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "transfers_pub_time",
			Help:      "Time to publish transfers (ms)",
		}, []string{}),
		PublishSideProposalTimeMs: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "side_proposal_pub_time",
			Help:      "Time to publish sideProposal (ms)",
		}, []string{}),

		NumTrade: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "num_trade",
			Help:      "Number of trades published",
		}, []string{}),
		NumOrder: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "num_order",
			Help:      "Number of orders published",
		}, []string{}),
		NumOrderBook: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "num_orderbook",
			Help:      "Number of price levels in orderbook we published",
		}, []string{}),
		NumAccounts: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "num_account",
			Help:      "Number of accounts we published",
		}, []string{}),
		NumTransfers: prometheus.NewGaugeFrom(stdprometheus.GaugeOpts{
			Subsystem: "publication",
			Name:      "num_transfer",
			Help:      "Number of transfer we published",
		}, []string{}),
	}
}
