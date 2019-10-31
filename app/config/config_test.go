package config

import (
	"fmt"
	"testing"

	"github.com/Shopify/sarama"
)

func TestKafkaVersion(t *testing.T) {
	pubCfg := defaultPublicationConfig()
	version, err := sarama.ParseKafkaVersion(pubCfg.KafkaVersion)
	if err != nil {
		t.Error(err)
	}
	if version != sarama.MaxVersion {
		t.Error(fmt.Errorf("default publisher setting is not compatible with current kafka setting"))
	}
}
