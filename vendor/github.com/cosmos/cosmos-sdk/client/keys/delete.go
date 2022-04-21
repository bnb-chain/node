package keys

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	keys "github.com/cosmos/cosmos-sdk/crypto/keys"
	keyerror "github.com/cosmos/cosmos-sdk/crypto/keys/keyerror"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

const flagYes = "yes"

func deleteKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete the given key",
		RunE:  runDeleteCmd,
		Args:  cobra.ExactArgs(1),
	}
	cmd.Flags().BoolP(flagYes, "y", false,
		"Skip confirmation prompt when deleting offline or tss key references")
	return cmd
}

func runDeleteCmd(cmd *cobra.Command, args []string) error {
	name := args[0]

	kb, err := GetKeyBaseWithWritePerm()
	if err != nil {
		return err
	}

	info, err := kb.Get(name)
	if err != nil {
		return err
	}

	buf := client.BufferStdin()
	if info.GetType() == keys.TypeLedger ||
		info.GetType() == keys.TypeOffline ||
		info.GetType() == keys.TypeTss {
		if !viper.GetBool(flagYes) {
			if err := confirmDeletion(buf); err != nil {
				return err
			}
		}
		if err := kb.Delete(name, "yes"); err != nil {
			return err
		}
		fmt.Println("Public key reference deleted")
		return nil
	}

	oldpass, err := client.GetPassword(
		"DANGER - enter password to permanently delete key:", buf)
	if err != nil {
		return err
	}

	err = kb.Delete(name, oldpass)
	if err != nil {
		return err
	}
	fmt.Println("Password deleted forever (uh oh!)")
	return nil
}

////////////////////////
// REST

// delete key request REST body
type DeleteKeyBody struct {
	Password string `json:"password"`
}

// delete key REST handler
func DeleteKeyRequestHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	var kb keys.Keybase
	var m DeleteKeyBody

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&m)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	kb, err = GetKeyBaseWithWritePerm()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	err = kb.Delete(name, m.Password)
	if keyerror.IsErrKeyNotFound(err) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(err.Error()))
		return
	} else if keyerror.IsErrWrongPassword(err) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	} else if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func confirmDeletion(buf *bufio.Reader) error {
	answer, err := client.GetConfirmation("Key reference will be deleted. Continue?", buf)
	if err != nil {
		return err
	}
	if !answer {
		return errors.New("aborted")
	}
	return nil
}
