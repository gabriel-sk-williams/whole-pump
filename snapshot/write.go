package snapshot

import (
	"encoding/json"
	"os"
	"whole-pump/model"
)

func writeWalletsJSON(wallets []string, filename string) error {
	data, err := json.MarshalIndent(wallets, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func writeWalletHistoriesJSON(histories map[string]model.WalletHistory, filename string) error {
	data, err := json.MarshalIndent(histories, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}
