package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
	"whole-pump/model"
)

//
// Helius structs
//

type TokenAccount struct {
	Address string `json:"address"`
	Mint    string `json:"mint"`
	Owner   string `json:"owner"`
	Amount  int64  `json:"amount"`
}

type HeliusResponse struct {
	Result struct {
		Total         int            `json:"total"`
		Limit         int            `json:"limit"`
		Cursor        string         `json:"cursor"`
		TokenAccounts []TokenAccount `json:"token_accounts"`
	} `json:"result"`
}

type TokenBalanceChange struct {
	UserAccount    string `json:"userAccount"`
	TokenAccount   string `json:"tokenAccount"`
	Mint           string `json:"mint"`
	RawTokenAmount struct {
		TokenAmount string `json:"tokenAmount"`
		Decimals    int    `json:"decimals"`
	} `json:"rawTokenAmount"`
}

type TokenTransfer struct {
	FromTokenAccount string  `json:"fromTokenAccount"`
	ToTokenAccount   string  `json:"toTokenAccount"`
	FromUserAccount  string  `json:"fromUserAccount"`
	ToUserAccount    string  `json:"toUserAccount"`
	TokenAmount      float64 `json:"tokenAmount"`
	Mint             string  `json:"mint"`
}

type NativeTransfer struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Amount          int64  `json:"amount"` // lamports
}

type AccountData struct {
	Account             string               `json:"account"`
	NativeBalanceChange int64                `json:"nativeBalanceChange"` // lamports, negative = SOL left wallet
	TokenBalanceChanges []TokenBalanceChange `json:"tokenBalanceChanges"`
}

type HeliusTransaction struct {
	Signature       string           `json:"signature"`
	Timestamp       int64            `json:"timestamp"`
	Slot            int64            `json:"slot"`
	Type            string           `json:"type"`
	Source          string           `json:"source"`
	TokenTransfers  []TokenTransfer  `json:"tokenTransfers"`
	NativeTransfers []NativeTransfer `json:"nativeTransfers"`
	AccountData     []AccountData    `json:"accountData"`
}

// Jupiter: JUP6LkbZbjS1jKKwapdHNy74zcZ3tLUZoi5QNyVTaV4
// OKX Router: ARu4n5mFdZogZAravu7CcizaojWnS6oqka37gdLT5SZn
// Meteora Pool Authority: HLnpSz9h2S4hiLQ43rnSD9XkcUThA7B8hQMKmDaiTLcC
// Fomo wallet: AgmLJB...zn51

// go run main.go snapshot wallet <address>

const PumpAMM = "B56BWXyJPZABa79JPV3AmghFieK8zLaLV1iaUmq5PvKd"
const WrappedSOL = "So11111111111111111111111111111111111111112"
const AnnouncementTimestamp = 1776976115
const TokenOrigin = 1772006401 // Feb 25

func Run(args []string) {
	apiKey := os.Getenv("HELIUS_API_KEY")
	mint := os.Getenv("TOKEN_ADDRESS")

	switch args[0] {
	case "eligible":
		eligible, ineligible, err := getEligible()
		checkFatal(err)
		writeWalletsJSON(eligible, "json/eligible.json")
		writeWalletsJSON(ineligible, "json/ineligible.json")
	case "holders":
		all, err := fetchCurrentHolders(apiKey, mint)
		checkFatal(err)
		writeHoldersJSON(all, "json/holders.json")
	case "affected":
		all, err := fetchAffectedWallets(apiKey, mint)
		checkFatal(err)
		writeWalletsJSON(all, "json/affected.json")
	case "history":
		fileName := args[1]
		all, err := fetchWalletHistories(apiKey, mint, fileName)
		checkFatal(err)
		writeWalletHistoriesJSON(all, fmt.Sprintf("json/%s_histories.json", fileName))
	case "wallet":
		if len(args) < 2 {
			log.Fatal("Usage: go run main.go snapshot wallet <address>")
		}
		wallet := args[1]
		walletTxs, err := fetchWalletTransactions(apiKey, mint, wallet)
		checkFatal(err)

		swapHistory := buildWalletHistoryFromTransactions(walletTxs, wallet, mint)

		singleHistory := make(map[string]model.WalletHistory)
		singleHistory[wallet] = swapHistory

		writeWalletHistoriesJSON(singleHistory, "json/single_history.json")

	default:
		log.Fatalf("Unknown command: %s", os.Args[1])
	}
}

func fetchWalletHistories(apiKey, mint, fileName string) (map[string]model.WalletHistory, error) {
	fmt.Println("API Key:", apiKey)
	fmt.Println("Token Address:", mint)

	data, err := os.ReadFile(fmt.Sprintf("json/%s.json", fileName))
	if err != nil {
		return nil, err
	}

	var wallets []string
	if err := json.Unmarshal(data, &wallets); err != nil {
		return nil, err
	}

	walletHistories := make(map[string]model.WalletHistory)

	for _, wallet := range wallets {
		swapTxs, err := fetchWalletTransactions(apiKey, mint, wallet)
		check(err)

		swapHistory := buildWalletHistoryFromTransactions(swapTxs, wallet, mint)
		if len(swapHistory.Buys) == 0 && len(swapHistory.Sells) == 0 {
			continue
		}

		walletHistories[wallet] = swapHistory
		time.Sleep(200 * time.Millisecond)
	}

	return walletHistories, nil
}

// go run main.go wallet <address>
// go run main.go snapshot wallet 4UK5njEDyNKFA36b4b67VAE6gC4ne8f7tJRiL5WDAXyc
func fetchWalletTransactions(apiKey, mint, wallet string) ([]HeliusTransaction, error) {
	var all []HeliusTransaction
	before := ""
	total := 0

	for {
		url := fmt.Sprintf("https://api.helius.xyz/v0/addresses/%s/transactions?api-key=%s&limit=100", wallet, apiKey)
		if before != "" {
			url += "&before=" + before
		}

		resp, err := http.Get(url)
		checkFatal(err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		checkFatal(err)

		var txs []HeliusTransaction
		if err := json.Unmarshal(body, &txs); err != nil {
			return nil, err
		}

		total += len(txs)
		if total > 500 {
			fmt.Printf("Skipping wallet %s: exceeded 500 transactions\n", wallet)
			return nil, nil
		}

		// filter for our mint only
		for _, tx := range txs {
			if tx.Timestamp < TokenOrigin {
				return all, nil // No relevant transactions before origin
			}

			for _, transfer := range tx.TokenTransfers {
				if transfer.Mint == mint {
					all = append(all, tx)
					break
				}
			}
		}

		//walletHistory := buildWalletHistoryFromTransactions(txs, wallet, mint)
		//printWalletHistory(walletHistory)

		if len(txs) < 100 {
			break
		}

		// Passes signature of the last transaction in each page
		// as the `before` parameter on the next request, stepping
		// further back in time until we reach the crash timestamp.
		before = txs[len(txs)-1].Signature
		time.Sleep(400 * time.Millisecond) // to avoid rate limit
	}

	return all, nil
}

func buildWalletHistoryFromTransactions(txs []HeliusTransaction, wallet, mint string) model.WalletHistory {
	var history model.WalletHistory

	fmt.Println("WALLET", wallet)
	for _, tx := range txs {
		var solIn, solOut, tokenIn, tokenOut float64

		for _, transfer := range tx.TokenTransfers {
			// SOL leaving the wallet (buy side)
			if transfer.Mint == WrappedSOL && transfer.FromUserAccount == wallet {
				solOut += transfer.TokenAmount
			}
			// SOL arriving at the wallet (sell side)
			if transfer.Mint == WrappedSOL && transfer.ToUserAccount == wallet {
				solIn += transfer.TokenAmount
			}
			// tokens arriving at the wallet
			if transfer.Mint == mint && transfer.ToUserAccount == wallet {
				tokenIn += transfer.TokenAmount
			}
			// tokens leaving the wallet
			if transfer.Mint == mint && transfer.FromUserAccount == wallet {
				tokenOut += transfer.TokenAmount
			}

			// if tokenIn > 0 || tokenOut > 0 {
			//		printTransaction(tx)
			// }
		}

		for _, acc := range tx.AccountData {
			fmt.Printf("Account: %s | NativeBalanceChange: %d\n", acc.Account, acc.NativeBalanceChange)
			for _, tbc := range acc.TokenBalanceChanges {
				fmt.Printf("  Token: %s | Amount: %s | User: %s\n", tbc.Mint, tbc.RawTokenAmount.TokenAmount, tbc.UserAccount)
			}
		}

		// --- Native SOL transfers (catches aggregator routes that never wrap SOL) ---
		for _, native := range tx.NativeTransfers {

			//fmt.Println("TRANSACTION")
			//fmt.Println(native)

			if native.FromUserAccount == wallet {
				solOut += float64(native.Amount) / 1e9
			}
			if native.ToUserAccount == wallet {
				solIn += float64(native.Amount) / 1e9
			}
		}

		// Native transfers are noisy — they include fees, rent, etc.
		// Deduplicate against wSOL amounts already counted, and subtract
		// the tx fee so you're not double-counting lamport noise.
		// A cleaner alternative (see note below) is to use accountData diffs.

		if tokenIn > 0 {
			history.Buys = append(history.Buys, model.WalletEvent{
				Source:      tx.Source,
				Slot:        tx.Slot,
				Timestamp:   tx.Timestamp,
				SOLAmount:   solSpentByWallet(tx, wallet),
				TokenAmount: tokenIn,
			})
		}
		if tokenOut > 0 {
			history.Sells = append(history.Sells, model.WalletEvent{
				Source:      tx.Source,
				Slot:        tx.Slot,
				Timestamp:   tx.Timestamp,
				SOLAmount:   solReceivedByWallet(tx, wallet),
				TokenAmount: tokenOut,
			})
		}
	}

	return history
}

//
// Get Lists
//

func getEligible() ([]string, []string, error) {
	var eligible, ineligible []string

	claimData, err := os.ReadFile("json/claimed.json")
	if err != nil {
		return nil, nil, err
	}

	var claimedWallets []string
	if err := json.Unmarshal(claimData, &claimedWallets); err != nil {
		return nil, nil, err
	}

	affectedData, err := os.ReadFile("json/affected.json")
	if err != nil {
		return nil, nil, err
	}

	var affectedWallets []string
	if err := json.Unmarshal(affectedData, &affectedWallets); err != nil {
		return nil, nil, err
	}

	// Build a set from affectedWallets for O(1) lookups
	affectedSet := make(map[string]struct{}, len(affectedWallets))
	for _, wallet := range affectedWallets {
		affectedSet[wallet] = struct{}{}
	}

	for _, wallet := range claimedWallets {
		if wallet == "" { // in claimed.json for organizational purposes
			continue
		}
		if _, found := affectedSet[wallet]; found {
			eligible = append(eligible, wallet)
		} else {
			ineligible = append(ineligible, wallet)
		}
	}

	return eligible, ineligible, nil
}

func fetchCurrentHolders(apiKey, mint string) ([]TokenAccount, error) {
	url := fmt.Sprintf("https://mainnet.helius-rpc.com/?api-key=%s", apiKey)

	var all []TokenAccount
	cursor := ""

	for {
		params := map[string]any{
			"mint":  mint,
			"limit": 1000,
		}
		if cursor != "" {
			params["cursor"] = cursor
		}

		body, _ := json.Marshal(map[string]any{
			"jsonrpc": "2.0",
			"id":      "1",
			"method":  "getTokenAccounts",
			"params":  params,
		})

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var result HeliusResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		all = append(all, result.Result.TokenAccounts...)

		if len(result.Result.TokenAccounts) < 1000 || result.Result.Cursor == "" {
			break
		}
		cursor = result.Result.Cursor
	}

	for _, account := range all {
		fmt.Println(account.Address, account.Mint, account.Owner, account.Amount)
	}

	return all, nil
}

func fetchAffectedWallets(apiKey, mint string) ([]string, error) {
	url := fmt.Sprintf("https://api.helius.xyz/v0/addresses/%s/transactions?api-key=%s&type=SWAP&limit=100", PumpAMM, apiKey)

	wallets := make(map[string]struct{})
	before := "" // i.e. before Signature

	for {
		reqURL := url
		if before != "" {
			reqURL += "&before=" + before
		}

		resp, err := http.Get(reqURL)
		checkFatal(err)

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		checkFatal(err)

		var txs []HeliusTransaction
		if err := json.Unmarshal(body, &txs); err != nil {
			return nil, err
		}

		for _, tx := range txs {
			if tx.Timestamp < AnnouncementTimestamp {
				fmt.Println("Timestamp reached:", tx.Timestamp)
				break
			}
			for _, transfer := range tx.TokenTransfers {
				if transfer.Mint == mint && transfer.TokenAmount > 0 {
					if transfer.FromUserAccount != "" && transfer.FromUserAccount != PumpAMM {
						wallets[transfer.FromUserAccount] = struct{}{}
					}
					if transfer.ToUserAccount != "" && transfer.ToUserAccount != PumpAMM {
						wallets[transfer.ToUserAccount] = struct{}{}
					}
				}
			}
		}

		if len(txs) == 0 || txs[len(txs)-1].Timestamp < AnnouncementTimestamp {
			break
		}

		// Passes signature of the last transaction in each page
		// as the `before` parameter on the next request, stepping
		// further back in time until we reach the crash timestamp.
		before = txs[len(txs)-1].Signature
		time.Sleep(200 * time.Millisecond)
	}

	result := make([]string, 0, len(wallets))
	for wallet := range wallets {
		result = append(result, wallet)
	}

	return result, nil
}

func solSpentByWallet(tx HeliusTransaction, wallet string) float64 {
	for _, acc := range tx.AccountData {
		if acc.Account == wallet {
			// nativeBalanceChange is negative when SOL left the wallet
			// already accounts for fees, rent, everything
			if acc.NativeBalanceChange < 0 {
				return float64(-acc.NativeBalanceChange) / 1e9
			}
		}
	}
	return 0
}

func solReceivedByWallet(tx HeliusTransaction, wallet string) float64 {
	for _, acc := range tx.AccountData {
		if acc.Account == wallet {
			if acc.NativeBalanceChange > 0 {
				return float64(acc.NativeBalanceChange) / 1e9
			}
		}
	}
	return 0
}

func printWalletHistory(wh model.WalletHistory) {
	for _, buy := range wh.Buys {
		fmt.Println(buy)
	}

	for _, sell := range wh.Sells {
		fmt.Println(sell)
	}
}

func printTransaction(tx HeliusTransaction) {
	fmt.Println(tx.Source)
	fmt.Println(tx.Signature)
	fmt.Println(tx.Type)
	for _, xfer := range tx.TokenTransfers {
		fmt.Println("From Token Account:", xfer.FromTokenAccount)
		fmt.Println("To Token Account:", xfer.ToTokenAccount)
		fmt.Println("From User Account:", xfer.FromUserAccount)
		fmt.Println("To User Account:", xfer.ToUserAccount)
		fmt.Println("Token Amount:", xfer.TokenAmount)
		fmt.Println("Mint:", xfer.Mint)
		fmt.Println()
	}
}
