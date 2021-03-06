
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

func get(net *StellarNet, query string) []byte {
	if net.Horizon == "" {
		fmt.Fprintln(os.Stderr, "Missing or invalid horizon config file\n")
		return nil
	}
	resp, err := http.Get(net.Horizon + query)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	return body
}

type HorizonSigner struct {
	Key string
	Weight uint32
}
type HorizonAccountEntry struct {
	Sequence json.Number
	Thresholds struct {
		Low_threshold uint8
		Med_threshold uint8
		High_threshold uint8
	}
	Signers []HorizonSigner
}

func GetAccountEntry(net *StellarNet, acct string) *HorizonAccountEntry {
	if body := get(net, "accounts/" + acct); body != nil {
		var ae HorizonAccountEntry
		if err := json.Unmarshal(body, &ae); err != nil {
			return nil
		}
		return &ae
	}
	return nil
}

func GetLedgerHeader(net *StellarNet) *LedgerHeader {
	body := get(net, "ledgers?limit=1&order=desc")
	if body == nil {
		return nil
	}

	var lhx struct {
		Embedded struct {
			Records []struct {
				Header_xdr string
			}
		} `json:"_embedded"`
	}
	if err := json.Unmarshal(body, &lhx); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return nil
	} else if len(lhx.Embedded.Records) == 0 {
		fmt.Fprintln(os.Stderr, "Horizon returned no ledgers")
		return nil
	}

	ret := &LedgerHeader{}
	if err := txIn(ret, lhx.Embedded.Records[0].Header_xdr); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return nil
	}
	return ret
}

func PostTransaction(net *StellarNet,
	e *TransactionEnvelope) *TransactionResult {
	if net.Horizon == "" {
		fmt.Fprintln(os.Stderr, "Missing or invalid horizon config file\n")
		return nil
	}
	tx := txOut(e)
	resp, err := http.PostForm(net.Horizon + "/transactions",
		url.Values{"tx": {tx}})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return nil
	}
	defer resp.Body.Close()

	js := json.NewDecoder(resp.Body)
	var res struct {
		Result_xdr string
		Extras struct {
			Result_xdr string
		}
	}
	if err = js.Decode(&res); err != nil {
		fmt.Fprintf(os.Stderr, "PostTransaction: %s\n", err.Error())
		return nil
	}
	if res.Result_xdr == "" { res.Result_xdr = res.Extras.Result_xdr }

	var ret TransactionResult
	if err = txIn(&ret, res.Result_xdr); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid result_xdr\n")
		return nil
	}
	return &ret
}
