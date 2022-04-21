package goodman

import (
	"bufio"
	"net"
	"testing"
)

func TestSendingServerMessages(t *testing.T) {
	runner := DummyRunner{}
	server := NewServer([]Runner{&runner})

	go func() {
		err := server.Run()
		if err != nil {
			t.Fatalf("Dredd hooks server failed to start with error %s", err.Error())
		}
	}()

	messages := []struct {
		Payload []byte
	}{
		{
			Payload: []byte("{\"uuid\":\"1234-abcd\",\"event\":\"beforeEach\",\"data\":{\"skip\":false}}\n"),
		},
		{
			Payload: []byte("{\"uuid\":\"2234-abcd\",\"event\":\"beforeEachValidation\",\"data\":{\"skip\":true}}\n"),
		},
		{
			Payload: []byte("{\"uuid\":\"2234-abcd\",\"event\":\"afterEach\",\"data\":{\"skip\":false}}\n"),
		},
		{
			Payload: []byte("{\"uuid\":\"2234-abcd\",\"event\":\"beforeAll\",\"data\":[{\"skip\":true}]}\n"),
		},
		{
			Payload: []byte("{\"uuid\":\"2234-abcd\",\"event\":\"afterAll\",\"data\":[{\"skip\":false}]}\n"),
		},
	}

	var (
		conn net.Conn
		err  error
	)

	for {
		// If server does not fail to start client should be able to connect.
		// This is to prevent test from failing due to syncronization between
		// server starting in go routine and client trying to connect.  This
		// is a test, should probably pass a channel to server.Run instead.
		conn, err = net.Dial("tcp", "localhost:61321")
		if err == nil {
			break
		}
	}

	if err != nil {
		t.Fatalf("Client connection to dredd hooks server failed")
	}

	for _, v := range messages {

		_, err := conn.Write(v.Payload)

		if err != nil {
			t.Errorf("Sending message %s failed with error %s", string(v.Payload), err.Error())
		}

		body, err := bufio.NewReader(conn).ReadString(byte('\n'))
		if body != string(v.Payload) {
			t.Errorf("Body of %s does not match the payload of %s", body, string(v.Payload))
		}
	}
}
