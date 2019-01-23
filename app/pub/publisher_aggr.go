package pub

type AggregatedMarketDataPublisher struct {
	publishers []MarketDataPublisher
}

func (publisher *AggregatedMarketDataPublisher) publish(msg AvroOrJsonMsg, tpe msgType, height int64, timestamp int64) {
	for _, pub := range publisher.publishers {
		pub.publish(msg, tpe, height, timestamp)
	}
}

func (publisher *AggregatedMarketDataPublisher) Stop() {
	for _, pub := range publisher.publishers {
		pub.Stop()
	}
}

func NewAggregatedMarketDataPublisher(publishers ...MarketDataPublisher) (publisher *AggregatedMarketDataPublisher) {
	publisher = &AggregatedMarketDataPublisher{
		publishers,
	}
	return
}
