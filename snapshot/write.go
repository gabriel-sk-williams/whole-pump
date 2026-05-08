package snapshot

import (
	"encoding/json"
	"math"
	"os"
	"whole-pump/model"
)

func NormalizeTokenAmount(amount int64) float64 {
	return float64(amount) / math.Pow(10, 6)
}

func writeTokenAccountsToHoldersJSON(accounts []TokenAccount, filename string) error {

	var holders []model.Holder
	for _, account := range accounts {
		holder := model.Holder{Wallet: account.Owner, Tokens: NormalizeTokenAmount(account.Amount)}
		holders = append(holders, holder)
	}

	data, err := json.MarshalIndent(holders, "", "  ")
	check(err)

	return os.WriteFile(filename, data, 0644)
}

func writeHoldersJSON(holders []model.Holder, filename string) error {
	data, err := json.MarshalIndent(holders, "", "  ")
	check(err)

	return os.WriteFile(filename, data, 0644)
}

func writeWalletsJSON(wallets []string, filename string) error {
	data, err := json.MarshalIndent(wallets, "", "  ")
	check(err)

	return os.WriteFile(filename, data, 0644)
}

func writeWalletHistoriesJSON(histories map[string]model.WalletHistory, filename string) error {
	data, err := json.MarshalIndent(histories, "", "  ")
	check(err)

	return os.WriteFile(filename, data, 0644)
}
