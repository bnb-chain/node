package pubsub

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/tendermint/tendermint/libs/pubsub"
)

const blockT = Topic("block")

type BlockCompleteEvent struct {
	txNum int
}

func (bc BlockCompleteEvent) GetTopic() Topic {
	return blockT
}

func TestSubscribe(t *testing.T) {

	server := startServer(t)

	sub, err := server.NewSubscriber("test_client", nil)
	require.Nil(t, err)

	_, err = server.NewSubscriber("test_client", nil)
	require.Equal(t, ErrDuplicateClientID, err)

	var getTxNum int
	err = sub.Subscribe(blockT, func(event Event) {
		switch event.(type) {
		case BlockCompleteEvent:
			time.Sleep(time.Second)
			bc := event.(BlockCompleteEvent)
			getTxNum = bc.txNum
		}
	})
	require.Nil(t, err)
	err = sub.Subscribe(blockT, func(event Event) {})
	require.Equal(t, pubsub.ErrAlreadySubscribed.Error(), err.Error())

	server.Publish(BlockCompleteEvent{txNum: 100})
	require.NotEqual(t, 100, getTxNum)
	sub.Wait()
	require.Equal(t, 100, getTxNum)

}

func TestUnsubscribe(t *testing.T) {
	server := startServer(t)

	clientId := ClientID("test_client")
	sub, err := server.NewSubscriber(clientId, nil)
	require.Nil(t, err)

	err = sub.Subscribe(blockT, func(event Event) {})
	require.Nil(t, err)

	require.True(t, server.HasSubscribed(clientId, blockT))

	err = sub.Unsubscribe(blockT)
	require.Nil(t, err)

	require.False(t, server.HasSubscribed(clientId, blockT))
}

func startServer(t *testing.T) *Server {
	pub := NewServer(nil)
	err := pub.Start()
	require.Nil(t, err)
	return pub
}
