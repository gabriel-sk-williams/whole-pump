package compute

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"whole-pump/model"
)

// go run main.go compute json/histories.json
const CompromiseEvent int64 = 1776932732
const AnnouncementTimestamp int64 = 1776976115

func Run(args []string) {
	fmt.Printf("%s running compute \n", args)

	path := args[0]

	histories, err := loadHistories(path)
	checkFatal(err)

	// printNumberOfTransactions(histories)

	computeLosses(histories)
}

func loadHistories(path string) (map[string]model.WalletHistory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var histories map[string]model.WalletHistory
	if err := json.Unmarshal(data, &histories); err != nil {
		return nil, fmt.Errorf("parsing json: %w", err)
	}

	return histories, nil
}

func printNumberOfTransactions(histories map[string]model.WalletHistory) {
	for wallet, history := range histories {
		numberOfBuys, numberOfSells, total := computeNumberOfTransactions(history)
		fmt.Printf("%s txs: %d + %d = %d  \n", truncate(wallet), numberOfBuys, numberOfSells, total)
	}
}

func computeNumberOfTransactions(wh model.WalletHistory) (int, int, int) {
	numberOfBuys := len(wh.Buys)
	numberOfSells := len(wh.Sells)
	total := numberOfBuys + numberOfSells
	return numberOfBuys, numberOfSells, total
}

func computeLosses(histories map[string]model.WalletHistory) {
	var positionsBeforeCrash []model.Position

	for wallet, history := range histories {
		_, _, total := computeNumberOfTransactions(history)
		solPosition, tokenPosition := computePositionBeforeTimestamp(history, AnnouncementTimestamp) //time.Now().Unix()
		percentSupply := tokenPosition / 1000000000.0 * 100

		position := model.Position{Wallet: wallet, Transactions: total, SOL: solPosition, Token: tokenPosition, PercentSupply: percentSupply}
		positionsBeforeCrash = append(positionsBeforeCrash, position)
	}

	slices.SortFunc(positionsBeforeCrash, func(a, b model.Position) int {
		return cmp.Compare(a.SOL, b.SOL)
	})

	for _, p := range positionsBeforeCrash {
		fmt.Printf("%s (%d txs) \n    SOL: %f\n    Tokens: %f  (%f%%) \n", truncate(p.Wallet), p.Transactions, p.SOL, p.Token, p.PercentSupply)
	}
}

func computePositionBeforeTimestamp(wh model.WalletHistory, t int64) (float64, float64) {
	var solSent, solReceived float64
	var tokensBought, tokensSold float64

	for _, buy := range wh.Buys {
		if buy.Timestamp < t && buy.SOLAmount > 0 {
			solSent += buy.SOLAmount
			tokensBought += buy.TokenAmount
		}
	}

	for _, sell := range wh.Sells {
		if sell.Timestamp < t && sell.SOLAmount > 0 {
			solReceived += sell.SOLAmount
			tokensSold += sell.TokenAmount
		}
	}

	solPosition := solReceived - solSent
	tokenPosition := tokensBought - tokensSold

	return solPosition, tokenPosition
}

func truncate(walletAddress string) string {
	front := walletAddress[:4]
	back := walletAddress[40:]

	return fmt.Sprintf("%s...%s", front, back)
}

func checkFatal(err error) {
	if err != nil {
		panic(err)
	}
}
