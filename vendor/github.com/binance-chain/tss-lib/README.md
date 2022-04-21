# Multi-Party Threshold Signature Scheme
[![GoDoc][1]][2] [![MIT licensed][3]][4] [![Go Report Card][5]][6] [![Coverage Status][7]][8]

[1]: https://godoc.org/github.com/binance-chain/tss-lib?status.svg
[2]: https://godoc.org/github.com/binance-chain/tss-lib
[3]: https://img.shields.io/badge/license-MIT-blue.svg
[4]: LICENSE
[5]: https://goreportcard.com/badge/github.com/binance-chain/tss-lib
[6]: https://goreportcard.com/report/github.com/binance-chain/tss-lib
[7]: https://codecov.io/gh/binance-chain/tss-lib/branch/master/graph/badge.svg
[8]: https://codecov.io/gh/binance-chain/tss-lib

## ECDSA
This is an implementation of multi-party {t,n}-threshold ECDSA (elliptic curve digital signatures) based on GG18.

This library includes three protocols:

* Key Generation for creating secret shares with no trusted dealer ("keygen").
* Signing for using the secret shares to generate a signature ("signing").
* Dynamic Groups to change the group of participants while keeping the secret ("resharing").

ECDSA is used extensively for crypto-currencies such as Bitcoin, Ethereum (secp256k1 curve), NEO (NIST P-256 curve) and much more. 
This library can be used to create MultiSig and ThresholdSig crypto wallets.

## Usage
You should start by creating an instance of a `LocalParty` and giving it the initialization arguments that it needs.

The `LocalParty` that you use should be from the `keygen`, `signing` or `resharing` package, depending on what you want to do.

```go
// When using the keygen party, it is recommended to pre-compute the "safe primes" and Paillier secret beforehand because this can take some time.
// This code will generate those parameters using a concurrency limit equal to the number of available CPU cores.
preParams, err := keygen.GeneratePreParams(1 * time.Minute)
// ... handle err ...

// Create the LocalParty and start it:
// Note: The `id` and `moniker` are provided for convenience to allow you to track participants easier. The `id` is intended to be a unique string representation of `key` and `moniker` can be anything (even left blank).
thisParty := tss.NewPartyID(id, moniker, uniqueKey)
ctx := tss.NewPeerContext(tss.SortPartyIDs(allParties))
params := tss.NewParameters(p2pCtx, thisParty, len(allParties), threshold)
party := keygen.NewLocalParty(params, outCh, endCh, preParams) // Omit the last arg to compute the pre-params in round 1
go func() {
    err := party.Start()
    // handle err ...
}()
```

In this example, the `outCh` will receive outgoing messages from this party, and the `endCh` will receive a message when the protocol is complete.

During the protocol, you should provide the party with updates received from other parties over the network (implementing the network transport is left to you):

A `Party` has two thread-safe methods on it for receiving updates:
```go
// The main entry point when updating a party's state from the wire
UpdateFromBytes(wireBytes []byte, from *tss.PartyID, isBroadcast, isToOldCommittee bool) (ok bool, err *tss.Error)
// You may use this entry point to update a party's state when running locally or in tests
Update(msg tss.ParsedMessage) (ok bool, err *tss.Error)
```

And a `tss.Message` has the following two methods for converting messages to data for the wire:
```go
// Returns the encoded message bytes to send over the wire along with routing information
WireBytes() ([]byte, *tss.MessageRouting, error)
// Returns the protobuf wrapper message struct, used only in some exceptional scenarios (i.e. mobile apps)
WireMsg() *tss.MessageWrapper
```

In a typical use case, it is expected that a transport implementation will **consume** message bytes via the `out` channel of the local `Party`, send them to the destination(s) specified in the result of `msg.GetTo()`, and **pass** them to `UpdateFromBytes` on the receiving end.

This way, there is no need to deal with Marshal/Unmarshalling Protocol Buffers to implement a transport.

## Transport Considerations
When you build a transport, it should should offer a broadcast channel as well as point-to-point channels connecting every pair of parties.

Your transport should also employ suitable end-to-end encryption to ensure that a party can only read the messages intended for it.

Additionally, there should be a mechanism in your transport to allow for "reliable broadcasts", meaning players can broadcast a message to all other players such that it's guaranteed that every player receives the same message. There are several examples of algorithms online that do this by sharing and comparing hashes of received messages.

Timeouts and errors should be handled by the transport. The method `WaitingFor` may be called on a `Party` to get the set of other parties that it is still waiting for messages from. You may also get the set of culprit parties that caused an error from a `*tss.Error`.

## Security Audit
A full review of this library was carried out by Kudelski Security and their final report was made available in October, 2019. A copy of this report [`audit-binance-tss-lib-final-20191018.pdf`](https://github.com/binance-chain/tss-lib/releases/download/v1.0.0/audit-binance-tss-lib-final-20191018.pdf) may be found in the v1.0.0 release notes of this repository.

## Resources
GG18: https://eprint.iacr.org/2019/114.pdf

