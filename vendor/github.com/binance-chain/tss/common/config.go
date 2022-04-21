package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"reflect"
	"strings"
	"syscall"

	"github.com/mitchellh/mapstructure"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/viper"
)

var TssCfg TssConfig

// A new type we need for writing a custom flag parser
type addrList []multiaddr.Multiaddr

func (al *addrList) String() string {
	strs := make([]string, len(*al))
	for i, addr := range *al {
		strs[i] = addr.String()
	}
	return strings.Join(strs, ",")
}

func (al *addrList) Set(value string) error {
	addr, err := multiaddr.NewMultiaddr(value)
	if err != nil {
		return err
	}
	*al = append(*al, addr)
	return nil
}

type P2PConfig struct {
	ListenAddr string `mapstructure:"listen" json:"listen"`

	// client only config
	BootstrapPeers       addrList `mapstructure:"bootstraps" json:"bootstraps"`
	RelayPeers           addrList `mapstructure:"relays" json:"relays"`
	PeerAddrs            []string `mapstructure:"peer_addrs" json:"peer_addrs"` // used for some peer has known connectable ip:port so that connection to them doesn't require bootstrap and relay nodes. i.e. in a LAN environment, if ip ports are preallocated, BootstrapPeers and RelayPeers can be empty with all parties host port set
	ExpectedPeers        []string `mapstructure:"peers" json:"peers"`           // expected peer list, <moniker>@<TssClientId>
	NewPeerAddrs         []string `mapstructure:"new_peer_addrs" json:"-"`      // same with `PeerAddrs` but for new parties for regroup
	ExpectedNewPeers     []string `mapstructure:"new_peers" json:"-"`           // expected new peer list used for regroup, <moniker>@<TssClientId>, after regroup success, this field will replace ExpectedPeers
	DefaultBootstap      bool     `mapstructure:"default_bootstrap", json:"default_bootstrap"`
	BroadcastSanityCheck bool     `mapstructure:"broadcast_sanity_check" json:"-"`
}

// Argon2 parameters, setting should refer 9th section of https://github.com/P-H-C/phc-winner-argon2/blob/master/argon2-specs.pdf
type KDFConfig struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
	SaltLength  uint32 `mapstructure:"salt_length" json:"salt_length"`
	KeyLength   uint32 `mapstructure:"key_length" json:"key_length"`
	Salt        string // hex encoded salt
}

func DefaultKDFConfig() KDFConfig {
	return KDFConfig{
		65536,
		13,
		4,
		16,
		48,
		"",
	}
}

type TssConfig struct {
	P2PConfig `mapstructure:"p2p" json:"p2p"`
	KDFConfig `mapstructure:"kdf" json:"-"` // kdf config will be persistent together with cryptoJSON,
	// no need to keep it in config file

	Id            TssClientId
	Moniker       string
	Vault         string `mapstructure:"vault_name" json:"vault_name"` // subdir within home to indicate alias of different vaults (addresses)
	AddressPrefix string `mapstructure:"address_prefix" json:"-"`      //

	Threshold    int
	Parties      int
	NewThreshold int `mapstructure:"new_threshold" json:"-"`
	NewParties   int `mapstructure:"new_parties" json:"-"`

	LogLevel    string `mapstructure:"log_level" json:"log_level"`
	ProfileAddr string `mapstructure:"profile_addr" json:"profile_addr"`
	Password    string `json:"-"`
	Message     string `json:"-"` // string represented big.Int, will refactor later

	ChannelId       string `mapstructure:"channel_id" json:"-"`
	ChannelPassword string `mapstructure:"channel_password" json:"-"`

	IsOldCommittee bool          `mapstructure:"is_old" json:"-"`
	IsNewCommittee bool          `mapstructure:"is_new_member" json:"-"`
	BMode          BootstrapMode `json:"-"`

	Home string
}

func ReadConfigFromHome(v *viper.Viper, init bool, home, vault, passphrase string) error {
	cfg, err := LoadConfig(home, vault, passphrase)
	if e, ok := err.(*os.PathError); ok && e.Err == syscall.ENOENT {
		if !init {
			// Cannot find config.json. This is not an error for init command
			return fmt.Errorf("vault does not exist, please check your \"--home\" or \"--vault_name\" parameter, error: %v", err)
		}
	} else if err != nil {
		return fmt.Errorf("cannot use vault, error: %v", err)
	}
	marshaled, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	v.SetConfigType("json")
	err = v.MergeConfig(bytes.NewReader(marshaled))
	if err != nil {
		return err
	}

	var config TssConfig
	err = v.Unmarshal(&config, func(config *mapstructure.DecoderConfig) {
		config.DecodeHook = func(from, to reflect.Type, data interface{}) (interface{}, error) {
			if from.Kind() == reflect.Slice && from.Elem().Kind() == reflect.String && to == reflect.TypeOf(addrList{}) {
				var al addrList
				for _, value := range data.([]string) {
					addr, err := multiaddr.NewMultiaddr(value)
					if err != nil {
						return nil, err
					}
					al = append(al, addr)
				}
				return al, nil
			}
			if from.Kind() == reflect.Slice && from.Elem().Kind() == reflect.Interface && to == reflect.TypeOf(addrList{}) {
				var al addrList
				for _, value := range data.([]interface{}) {
					addr, err := multiaddr.NewMultiaddr(value.(string))
					if err != nil {
						return nil, err
					}
					al = append(al, addr)
				}
				return al, nil
			}
			return data, nil
		}
	})
	if err != nil {
		return err
	}
	// override kdfconfig with loaded kdf config rather than command line ones (because after init, kdf configs are not bound)
	// TODO: exclude KDFConfig from TssConfig
	if cfg != nil {
		config.KDFConfig = cfg.KDFConfig
	}

	// validate configs
	// TODO: reenable this when we want release bootstrap server supported version
	//if len(config.P2PConfig.BootstrapPeers) == 0 {
	//fmt.Println("!!!NOTICE!!! cannot find bootstraps servers in config")
	//if config.P2PConfig.DefaultBootstap {
	//	fmt.Println("!!!NOTICE!!! Would use libp2p's default bootstraps")
	//	config.P2PConfig.BootstrapPeers = dht.DefaultBootstrapPeers
	//}
	//}
	if config.KDFConfig.KeyLength != 48 {
		return fmt.Errorf("derived key length must be 48 bytes (32 bytes aes and 16 bytes MAC)")
	}

	if config.ProfileAddr != "" {
		go func() {
			http.ListenAndServe(config.ProfileAddr, nil)
		}()
	}

	TssCfg = config
	return nil
}
