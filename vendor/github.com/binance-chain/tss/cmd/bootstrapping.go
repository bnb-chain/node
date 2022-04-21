package cmd

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/bgentry/speakeasy"
	"github.com/gogo/protobuf/proto"
	"github.com/spf13/cobra"

	"github.com/binance-chain/tss/client"
	"github.com/binance-chain/tss/common"
	"github.com/binance-chain/tss/p2p"
	"github.com/binance-chain/tss/ssdp"
)

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}

var bootstrapCmd = &cobra.Command{
	Use:    "bootstrap",
	Short:  "bootstrapping for network configuration",
	Long:   "bootstrapping for network configuration. Will try connect to configured address and get peer's id and moniker",
	Hidden: true, // This command would be used as a step of other commands rather than a standalone one
	Run: func(cmd *cobra.Command, args []string) {
		src, err := common.ConvertMultiAddrStrToNormalAddr(common.TssCfg.ListenAddr)
		if err != nil {
			common.Panic(err)
		}
		listenAddrs := getListenAddrs(common.TssCfg.ListenAddr)
		client.Logger.Debugf("This node is listening on: %v", listenAddrs)

		setChannelId()
		setChannelPasswd()
		client.Logger.Info("waiting peers startup...")
		numOfPeers := common.TssCfg.Parties - 1
		if common.TssCfg.BMode == common.PreRegroupMode {
			numOfPeers = common.TssCfg.Threshold + common.TssCfg.NewParties
		}

		bootstrapper := common.NewBootstrapper(numOfPeers, &common.TssCfg)

		listener, err := net.Listen("tcp", src)
		client.Logger.Infof("listening on %s", src)
		if err != nil {
			common.Panic(err)
		}
		defer func() {
			err = listener.Close()
			if err != nil {
				client.Logger.Error(err)
			}
			client.Logger.Info("closed ssdp listener")
		}()

		done := make(chan bool)
		go acceptConnRoutine(listener, bootstrapper, done)

		peerAddrs := findPeerAddrsViaSsdp(numOfPeers, listenAddrs)
		client.Logger.Debugf("Found peers via ssdp: %v", peerAddrs)

		go func() {
			for _, peerAddr := range peerAddrs {
				go func(peerAddr string) {
					dest, err := common.ConvertMultiAddrStrToNormalAddr(peerAddr)
					if err != nil {
						common.Panic(fmt.Errorf("failed to convert peer multiAddr to addr: %v", err))
					}
					conn, err := net.Dial("tcp", dest)
					for conn == nil {
						if err != nil {
							if !strings.Contains(err.Error(), "connection refused") {
								common.Panic(err)
							}
						}
						time.Sleep(time.Second)
						conn, err = net.Dial("tcp", dest)
					}
					defer conn.Close()
					handleConnection(conn, bootstrapper)
				}(peerAddr)
			}

			checkReceivedPeerInfos(bootstrapper, done)
		}()

		<-done
		err = updateConfigWithPeerInfos(bootstrapper)
		if err != nil {
			common.Panic(err)
		}
	},
}

func setChannelId() {
	if common.TssCfg.ChannelId != "" {
		return
	}

	reader := bufio.NewReader(os.Stdin)
	channelId, err := common.GetString("please set channel id of this session", reader)
	if err != nil {
		common.Panic(err)
	}
	if len(channelId) != 11 {
		common.Panic(fmt.Errorf("channelId format is invalid"))
	}
	common.TssCfg.ChannelId = channelId
}

func setChannelPasswd() {
	if pw := common.TssCfg.ChannelPassword; pw != "" {
		checkComplexityOfPassword(pw)
		return
	}

	if p, err := speakeasy.Ask("> please input password (AGREED offline with peers) of this session:"); err == nil {
		if p == "" {
			common.Panic(fmt.Errorf("channel password should not be empty"))
		}
		checkComplexityOfPassword(p)
		common.TssCfg.ChannelPassword = p
	} else {
		common.Panic(err)
	}
}

func findPeerAddrsViaSsdp(n int, listenAddrs string) []string {
	if common.TssCfg.BMode == common.KeygenMode && len(common.TssCfg.PeerAddrs) == n {
		return common.TssCfg.PeerAddrs
	}
	if common.TssCfg.BMode == common.PreRegroupMode && len(common.TssCfg.NewPeerAddrs) == n {
		return common.TssCfg.NewPeerAddrs
	}

	existingMonikers := make(map[string]struct{})
	for _, peer := range common.TssCfg.ExpectedPeers {
		moniker := p2p.GetMonikerFromExpectedPeers(peer)
		existingMonikers[moniker] = struct{}{}
	}
	ssdpSrv := ssdp.NewSsdpService(common.TssCfg.Moniker, common.TssCfg.Vault, listenAddrs, n, existingMonikers)
	ssdpSrv.CollectPeerAddrs()
	var peerAddrs []string
	ssdpSrv.PeerAddrs.Range(func(_, value interface{}) bool {
		if peerAddr, ok := value.(string); ok {
			peerAddrs = append(peerAddrs, peerAddr)
		}
		return true
	})
	return peerAddrs
}

func acceptConnRoutine(listener net.Listener, bootstrapper *common.Bootstrapper, done <-chan bool) {
	for {
		select {
		case <-done:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					client.Logger.Errorf("Some connection error: %s\n", err)
				}
				continue
			} else {
				client.Logger.Debugf("%s connected to us!\n", conn.RemoteAddr().String())
			}

			handleConnection(conn, bootstrapper)
		}
	}
}

func handleConnection(conn net.Conn, b *common.Bootstrapper) {
	client.Logger.Debugf("handling connection from %s", conn.RemoteAddr().String())

	sendBootstrapMessage(conn, b.Msg)
	readBootstrapMessage(conn, b)
}

func sendBootstrapMessage(conn net.Conn, msg *common.BootstrapMessage) {
	// TODO: support ipv6
	realIp := strings.SplitN(conn.LocalAddr().String(), ":", 2)
	msgForConnect := common.BootstrapMessage{
		ChannelId: msg.ChannelId,
		PeerInfo:  msg.PeerInfo,
		Addr:      common.ReplaceIpInAddr(msg.Addr, realIp[0]),
	}

	payload, err := proto.Marshal(&msgForConnect)
	if err != nil {
		common.SkipTcpClosePanic(fmt.Errorf("bootstrap message cannot be marshaled to protobuf payload: %v", err))
		return
	}
	messageLength := int32(len(payload))
	err = binary.Write(conn, binary.BigEndian, &messageLength)
	if err != nil {
		common.SkipTcpClosePanic(fmt.Errorf("failed to write bootstrap message length: %v", err))
		return
	}
	n, err := conn.Write(payload)
	if int32(n) != messageLength || err != nil {
		common.SkipTcpClosePanic(fmt.Errorf("failed to write bootstrap message: %v", err))
		return
	}
	client.Logger.Debugf("sent bootstrap msg: %v to %s", msgForConnect, conn.RemoteAddr().String())
}

func readBootstrapMessage(conn net.Conn, b *common.Bootstrapper) {
	var messageLength int32
	err := binary.Read(conn, binary.BigEndian, &messageLength)
	if err != nil {
		common.SkipTcpClosePanic(fmt.Errorf("failed to read bootstrap message length: %v", err))
		return
	}
	payload := make([]byte, messageLength)
	n, err := conn.Read(payload)

	if int32(n) != messageLength {
		return
	}
	if err != nil {
		common.SkipTcpClosePanic(fmt.Errorf("failed to read bootstrap message: %v", err))
		return
	}
	var peerMsg common.BootstrapMessage
	err = proto.Unmarshal(payload, &peerMsg)
	if err != nil {
		common.SkipTcpClosePanic(fmt.Errorf("failed to unmarshal bootstrap message: %v", err))
		return
	}
	if err := b.HandleBootstrapMsg(peerMsg); err != nil {
		// peer's channel id or channel password is not correct, we can wait them fix
		client.Logger.Error(err)
	}
}

func checkReceivedPeerInfos(bootstrapper *common.Bootstrapper, done chan<- bool) {
	for {
		if bootstrapper.IsFinished() {
			done <- true
			close(done)
			break
		} else {
			time.Sleep(time.Second)
		}
	}
}

func updateConfigWithPeerInfos(bootstrapper *common.Bootstrapper) error {
	peerAddrs := make([]string, 0)
	expectedPeers := make([]string, 0)

	newPeerAddrs := make([]string, 0)
	expectedNewPeers := make([]string, 0)

	var err error
	bootstrapper.Peers.Range(func(id, value interface{}) bool {
		if pi, ok := value.(common.PeerInfo); ok {
			if common.TssCfg.BMode != common.PreRegroupMode || (common.TssCfg.BMode == common.PreRegroupMode && pi.IsOld) {
				peerAddrs = append(peerAddrs, pi.RemoteAddr)
				expectedPeers = append(expectedPeers, fmt.Sprintf("%s@%s", pi.Moniker, pi.Id))
			} else {
				newPeerAddrs = append(newPeerAddrs, pi.RemoteAddr)
				expectedNewPeers = append(expectedNewPeers, fmt.Sprintf("%s@%s", pi.Moniker, pi.Id))
			}
			return true
		} else {
			err = fmt.Errorf("failed to parse peerInfo from received messages")
			return false
		}
	})

	if err != nil {
		return err
	}

	common.TssCfg.PeerAddrs, common.TssCfg.ExpectedPeers = mergeAndUpdate(
		common.TssCfg.PeerAddrs,
		common.TssCfg.ExpectedPeers,
		peerAddrs,
		expectedPeers)
	common.TssCfg.NewPeerAddrs, common.TssCfg.ExpectedNewPeers = mergeAndUpdate(
		common.TssCfg.NewPeerAddrs,
		common.TssCfg.ExpectedNewPeers,
		newPeerAddrs,
		expectedNewPeers)

	return nil
}

func mergeAndUpdate(peerAddrs, expectedPeers, updatedPeerAddrs, updatedPeers []string) ([]string, []string) {
	mergedPeers := make(map[string]string) // expected peer -> peer addr
	for i, peer := range expectedPeers {
		mergedPeers[peer] = peerAddrs[i]
	}
	for i, peer := range updatedPeers {
		// update addr if already exists
		mergedPeers[peer] = updatedPeerAddrs[i]
	}

	updatedPeerAddrs = make([]string, 0)
	updatedPeers = make([]string, 0)
	for peer, addr := range mergedPeers {
		updatedPeers = append(updatedPeers, peer)
		updatedPeerAddrs = append(updatedPeerAddrs, addr)
	}

	return updatedPeerAddrs, updatedPeers
}
