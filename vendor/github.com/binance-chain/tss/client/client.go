package client

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path"
	"strconv"
	"time"

	lib "github.com/binance-chain/tss-lib/common"
	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/ecdsa/resharing"
	"github.com/binance-chain/tss-lib/ecdsa/signing"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/btcsuite/btcd/btcec"
	"github.com/gogo/protobuf/proto"
	"github.com/ipfs/go-log"

	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
)

var Logger = log.Logger("tss")

type ClientMode uint8

const (
	KeygenMode ClientMode = iota
	SignMode
	RegroupMode
)

func (m ClientMode) String() string {
	switch m {
	case KeygenMode:
		return "keygen"
	case SignMode:
		return "sign"
	case RegroupMode:
		return "regroup"
	default:
		common.Panic(fmt.Errorf("unknown mode"))
		return "" // won't reach here as Panic would kill process
	}
}

type TssClient struct {
	config      *common.TssConfig
	localParty  tss.Party
	transporter common.Transporter

	params        *tss.Parameters
	regroupParams *tss.ReSharingParameters
	idToPartyIds  map[string]*tss.PartyID
	key           *keygen.LocalPartySaveData
	signature     []byte

	saveCh chan keygen.LocalPartySaveData
	signCh chan signing.SignatureData
	sendCh chan tss.Message

	mode ClientMode
}

func NewTssClient(config *common.TssConfig, mode ClientMode, mock bool) *TssClient {
	id := string(config.Id)
	idToPartyIds := make(map[string]*tss.PartyID)
	key := lib.SHA512_256([]byte(id)) // TODO: discuss should we really need pass p2p nodeid pubkey into NewPartyID? (what if in memory implementation)
	partyID := tss.NewPartyID(id, config.Moniker, new(big.Int).SetBytes(key))
	idToPartyIds[id] = partyID
	unsortedPartyIds := make(tss.UnSortedPartyIDs, 0, config.Parties)
	if mode == RegroupMode {
		if common.TssCfg.IsOldCommittee {
			unsortedPartyIds = append(unsortedPartyIds, partyID)
		}
	} else {
		unsortedPartyIds = append(unsortedPartyIds, partyID)
	}
	unsortedNewPartyIds := make(tss.UnSortedPartyIDs, 0, config.NewParties)

	signers := make(map[string]int, 0) // used by sign and regroup mode for filtering correct shares from LocalPartySaveData, including self
	if mode != KeygenMode {
		if mode == SignMode {
			common.TssCfg.BMode = common.SignMode
		}
		if mode == RegroupMode {
			common.TssCfg.BMode = common.RegroupMode
		}
		bootstrapper := common.NewBootstrapper(0, &common.TssCfg)
		t := p2p.NewP2PTransporter(config.Home, config.Vault, config.Id.String(), bootstrapper, nil, nil, signers, &config.P2PConfig)
		t.Shutdown()
		bootstrapper.Peers.Range(func(_, value interface{}) bool {
			if pi, ok := value.(common.PeerInfo); ok {
				if mode == SignMode || (mode == RegroupMode && pi.IsOld) {
					signers[pi.Moniker] = 0
				}

				if mode == RegroupMode {
					if pi.IsNew {
						// we don't know whether pi's info has been updated during raw tcp bootstrapping
						config.ExpectedNewPeers = appendIfNotExist(config.ExpectedNewPeers, fmt.Sprintf("%s@%s", pi.Moniker, pi.Id))
						config.NewPeerAddrs = appendIfNotExist(config.NewPeerAddrs, pi.RemoteAddr)
					}
				}
			}
			return true
		})
		if mode == SignMode || (mode == RegroupMode && common.TssCfg.IsOldCommittee) {
			signers[config.Moniker] = 0
		}

		if len(signers) < config.Threshold+1 {
			common.Panic(fmt.Errorf("no enough signers (%d) to meet requirement: %d", len(signers), config.Threshold+1))
		}
		updatePeerOriginalIndexes(config, bootstrapper, partyID, signers)
	}

	if !mock {
		for _, peer := range config.P2PConfig.ExpectedPeers {
			id := string(p2p.GetClientIdFromExpectedPeers(peer))
			moniker := p2p.GetMonikerFromExpectedPeers(peer)
			key := lib.SHA512_256([]byte(id))
			if mode == SignMode || mode == RegroupMode {
				if _, ok := signers[moniker]; !ok {
					continue
				}
			}
			partyId := tss.NewPartyID(
				id,
				moniker,
				new(big.Int).SetBytes(key))
			idToPartyIds[id] = partyId
			unsortedPartyIds = append(unsortedPartyIds, partyId)
		}
		if mode == RegroupMode {
			for _, peer := range config.P2PConfig.ExpectedNewPeers {
				id := string(p2p.GetClientIdFromExpectedPeers(peer))
				moniker := p2p.GetMonikerFromExpectedPeers(peer)
				key := lib.SHA512_256([]byte(id))
				if moniker != config.Moniker {
					partyId := tss.NewPartyID(
						id,
						moniker,
						new(big.Int).SetBytes(key))
					idToPartyIds[id] = partyId
					unsortedNewPartyIds = append(unsortedNewPartyIds, partyId)
				}
			}
			if !config.IsOldCommittee {
				unsortedNewPartyIds = append(unsortedNewPartyIds, partyID)
			}
		}
	} else {
		for i := 0; i < config.Parties; i++ {
			id, _ := strconv.Atoi(string(config.Id))
			if i != id {
				id := strconv.Itoa(i)
				key := lib.SHA512_256([]byte(id))
				unsortedPartyIds = append(unsortedPartyIds, tss.NewPartyID(id, id, new(big.Int).SetBytes(key)))
			}
		}
	}
	sortedIds := tss.SortPartyIDs(unsortedPartyIds)
	p2pCtx := tss.NewPeerContext(sortedIds)
	saveCh := make(chan keygen.LocalPartySaveData)
	signCh := make(chan signing.SignatureData)
	sendCh := make(chan tss.Message, len(sortedIds)*10*2) // max signing messages 10 times hash confirmation messages
	c := TssClient{
		config:       config,
		idToPartyIds: idToPartyIds,

		saveCh: saveCh,
		signCh: signCh,
		sendCh: sendCh,

		mode: mode,
	}

	var localParty tss.Party
	if mode == KeygenMode {
		params := tss.NewParameters(p2pCtx, partyID, config.Parties, config.Threshold)
		localParty = keygen.NewLocalParty(params, sendCh, saveCh)
		c.localParty = localParty
		Logger.Infof("[%s] initialized localParty: %s", config.Moniker, localParty)
	} else if mode == SignMode {
		key := loadSavedKeyForSign(config, sortedIds, signers)
		pubKey := btcec.PublicKey(ecdsa.PublicKey{tss.EC(), key.ECDSAPub.X(), key.ECDSAPub.Y()})
		Logger.Infof("[%s] public key: %X\n", config.Moniker, pubKey.SerializeCompressed())
		address, err := GetAddress(ecdsa.PublicKey{tss.EC(), key.ECDSAPub.X(), key.ECDSAPub.Y()}, config.AddressPrefix)
		if err != nil {
			panic(err)
		}
		Logger.Debugf("[%s] address is: %s\n", config.Moniker, address)
		params := tss.NewParameters(p2pCtx, partyID, config.Parties, config.Threshold)
		c.key = &key
		c.params = params
	} else if mode == RegroupMode {
		sortedNewIds := tss.SortPartyIDs(unsortedNewPartyIds)
		newP2pCtx := tss.NewPeerContext(sortedNewIds)
		params := tss.NewReSharingParameters(
			p2pCtx,
			newP2pCtx,
			partyID,
			config.Parties,
			config.Threshold,
			config.NewParties,
			config.NewThreshold)
		c.regroupParams = params

		if _, ok := signers[common.TssCfg.Moniker]; ok {
			key := loadSavedKeyForRegroup(config, sortedIds, signers)
			c.key = &key
			localParty = resharing.NewLocalParty(params, key, sendCh, saveCh)
		} else {
			// TODO do this better!
			save := newEmptySaveData()
			localParty = resharing.NewLocalParty(params, save, sendCh, saveCh)
		}
		c.localParty = localParty
	}

	if mock {
		c.transporter = p2p.GetMemTransporter(config.Id)
	} else {
		// will block until peers are connected
		c.transporter = p2p.NewP2PTransporter(
			config.Home,
			config.Vault,
			config.Id.String(),
			nil,
			c.params,
			c.regroupParams,
			signers,
			&config.P2PConfig)
	}

	return &c
}

func (client *TssClient) Start() {
	switch client.mode {
	case SignMode:
		message, ok := big.NewInt(0).SetString(client.config.Message, 10)
		if !ok {
			common.Panic(fmt.Errorf("message to be sign: %s is not a valid big.Int", client.config.Message))
		}
		client.signImpl(message)
		time.Sleep(5 * time.Second)
	default:
		if err := client.localParty.Start(); err != nil {
			common.Panic(err)
		}
		done := make(chan bool)
		go client.sendMessageRoutine(client.sendCh)
		go client.saveDataRoutine(client.saveCh, done)
		//go c.sendDummyMessageRoutine()
		go client.handleMessageRoutine()
		<-done
	}
}

func (client *TssClient) handleMessageRoutine() {
	for msg := range client.transporter.ReceiveCh() {
		var messageWrapper tss.MessageWrapper
		if err := proto.Unmarshal(msg.MessageWrapperBytes, &messageWrapper); err != nil {
			common.Panic(fmt.Errorf("[%s] error updating local party state: %v", client.config.Moniker, err))
		}
		any, _ := proto.Marshal(messageWrapper.Message)
		ok, err := client.localParty.UpdateFromBytes(
			any,
			client.idToPartyIds[messageWrapper.From.Id],
			messageWrapper.IsBroadcast,
			messageWrapper.IsToOldCommittee)
		if err != nil {
			Logger.Errorf("[%s] error updating local party state: %v", client.config.Moniker, err)
		} else if !ok {
			Logger.Warningf("[%s] Update still waiting for round to finish", client.config.Moniker)
		} else {
			Logger.Debugf("[%s] update success", client.config.Moniker)
		}
	}
}

func (client *TssClient) sendMessageRoutine(sendCh <-chan tss.Message) {
	for msg := range sendCh {
		dest := msg.GetTo()
		if dest == nil || len(dest) > 1 {
			err := client.transporter.Broadcast(msg)
			if err != nil {
				Logger.Errorf("failed to broadcast message: %v", err)
			}
		} else {
			payload, err := proto.Marshal(msg.WireMsg())
			if err != nil {
				Logger.Error("failed to protobuf marshal the message wrapper: %v", err)
			}
			payload = append([]byte{p2p.MessagePrefix}, payload...)
			if err = client.transporter.Send(payload, common.TssClientId(dest[0].Id)); err != nil {
				Logger.Errorf("failed to send message: %v", err)
			}
		}
	}
}

func (client *TssClient) saveDataRoutine(saveCh <-chan keygen.LocalPartySaveData, done chan<- bool) {
	for msg := range saveCh {
		// Used for debugging signature verification failed issue, never uncomment in production!
		//plainJson, err := json.Marshal(msg)
		//ioutil.WriteFile(path.Join(client.config.Home, "plain.json"), plainJson, 0400)

		if client.mode == RegroupMode {
			if !common.TssCfg.IsNewCommittee {
				// wait for round_3 messages sent success before close old
				// TODO: introduce a send callback to waiting here
				time.Sleep(5 * time.Second)
				if done != nil {
					done <- true
					close(done)
				}
				break
			}
		}

		Logger.Infof("[%s] received save data", client.config.Moniker)
		address, err := GetAddress(ecdsa.PublicKey{tss.EC(), msg.ECDSAPub.X(), msg.ECDSAPub.Y()}, client.config.AddressPrefix)
		if err != nil {
			Logger.Errorf("[%s] failed to generate address from public key :%v", client.config.Moniker, err)
		} else {
			Logger.Infof("[%s] bech32 address is: %s", client.config.Moniker, address)
		}

		wPriv, err := os.OpenFile(path.Join(client.config.Home, client.config.Vault, "sk.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			common.Panic(err)
		}
		defer wPriv.Close() // defer within loop is fine here as for one party there would be only one element from saveCh
		wPub, err := os.OpenFile(path.Join(client.config.Home, client.config.Vault, "pk.json"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			common.Panic(err)
		}
		defer wPub.Close() // defer within loop is fine here as for one party there would be only one element from saveCh
		err = common.Save(&msg, client.transporter.NodeKey(), client.config.KDFConfig, client.config.Password, wPriv, wPub)
		if err != nil {
			common.Panic(err)
		}

		if done != nil {
			done <- true
			close(done)
		}
		break
	}
}

func (client *TssClient) saveSignatureRoutine(signCh <-chan signing.SignatureData, done chan<- bool) {
	for signature := range signCh {
		client.signature = signature.Signature
		if done != nil {
			done <- true
			close(done)
		}
		break
	}
}

// assign original keygen index to signers (old parties in regroup)
func updatePeerOriginalIndexes(config *common.TssConfig, bootstrapper *common.Bootstrapper, partyID *tss.PartyID, signers map[string]int) {
	allPartyIds := make(tss.UnSortedPartyIDs, 0, config.Parties) // all parties, used for calculating party's index during keygen
	allPartyIds = append(allPartyIds, partyID)
	for _, peer := range config.P2PConfig.ExpectedPeers {
		id := string(p2p.GetClientIdFromExpectedPeers(peer))
		moniker := p2p.GetMonikerFromExpectedPeers(peer)
		key := lib.SHA512_256([]byte(id))
		allPartyIds = append(allPartyIds,
			tss.NewPartyID(
				id,
				moniker,
				new(big.Int).SetBytes(key)))
	}

	sortedIds := tss.SortPartyIDs(allPartyIds)
	for _, id := range sortedIds {
		if _, ok := signers[id.Moniker]; ok {
			signers[id.Moniker] = id.Index
		}
	}
}
