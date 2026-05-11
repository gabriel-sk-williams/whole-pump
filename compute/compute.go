package compute

import (
	"cmp"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"time"
	"whole-pump/model"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// go run main.go compute json/histories.json
const CompromiseEvent int64 = 1776932732
const AnnouncementTimestamp int64 = 1776976115

func Run(args []string) {
	fmt.Printf("%s running compute \n", args)

	path := args[0]

	histories, err := loadHistories(fmt.Sprintf("%s_histories_enriched.json", path))
	checkFatal(err)

	fmt.Println("lenymo", len(histories))

	totalPositions := computeTotalPositions(histories)
	checkFatal(err)

	computeLosses(totalPositions)
}

func loadHistories(path string) (map[string]model.WalletHistory, error) {
	data, err := os.ReadFile(fmt.Sprintf("json/%s", path))
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var histories map[string]model.WalletHistory
	if err := json.Unmarshal(data, &histories); err != nil {
		return nil, fmt.Errorf("parsing json: %w", err)
	}

	return histories, nil
}

func computeNumberOfTransactions(wh model.WalletHistory, t int64) (int, int, int) {
	var numberOfBuys, numberOfSells, total int

	for _, buy := range wh.Buys {
		if buy.Timestamp < t {
			numberOfBuys += 1
			total += 1
		}
	}

	for _, sell := range wh.Sells {
		if sell.Timestamp < t {
			numberOfSells += 1
			total += 1
		}
	}

	return numberOfBuys, numberOfSells, total
}

func computeTotalPositions(histories map[string]model.WalletHistory) map[string]model.TotalPosition {
	totalPositions := make(map[string]model.TotalPosition)
	now := time.Now().Unix()

	for wallet, history := range histories {
		_, _, totalBefore := computeNumberOfTransactions(history, AnnouncementTimestamp)
		solBefore, tokenBefore := computePositionBeforeTimestamp(history, AnnouncementTimestamp)
		percentSupplyBefore := percentSupply(tokenBefore)

		_, _, totalAfter := computeNumberOfTransactions(history, now)
		solNow, tokenNow := computePositionBeforeTimestamp(history, time.Now().Unix())
		percentSupplyNow := percentSupply(tokenNow)

		positionBefore := model.Position{Transactions: totalBefore, SOL: solBefore, Token: tokenBefore, PercentSupply: percentSupplyBefore}
		positionNow := model.Position{Transactions: totalAfter, SOL: solNow, Token: tokenNow, PercentSupply: percentSupplyNow}

		netTransfer := computeNetTransfers(history)

		totalPositions[wallet] = model.TotalPosition{PositionBefore: positionBefore, PositionNow: positionNow, NetTransfer: netTransfer}
	}

	return totalPositions
}

func computeNetTransfers(history model.WalletHistory) float64 {
	var netReceived, netSent float64

	for _, rt := range history.Received {
		netReceived += rt.TokenAmount
	}

	for _, st := range history.Sent {
		netSent += st.TokenAmount
	}

	return netReceived - netSent
}

func computeLosses(positions map[string]model.TotalPosition) {
	var allLosses []model.ComputedLoss
	var winners []model.ComputedLoss

	var totalLosses float64
	for wallet, tp := range positions {
		if tp.PositionNow.SOL < 0 {
			computedLoss := model.ComputedLoss{Wallet: wallet, Loss: tp.PositionNow.SOL, NetTransfer: tp.NetTransfer}
			totalLosses += computedLoss.Loss
			allLosses = append(allLosses, computedLoss)
		} else {
			computedLoss := model.ComputedLoss{Wallet: wallet, Loss: tp.PositionNow.SOL, NetTransfer: tp.NetTransfer}
			winners = append(winners, computedLoss)
		}
	}

	slices.SortFunc(allLosses, func(a, b model.ComputedLoss) int {
		return cmp.Compare(a.Loss, b.Loss)
	})

	totalPool := 100_000_000.0
	fmt.Println("Total Loss: ", totalLosses)
	fmt.Printf("Total Tokens: %f \n", totalPool)
	for _, cl := range allLosses {
		position := positions[cl.Wallet]
		tokensBefore := position.PositionBefore.Token
		percentageLoss := cl.Loss / totalLosses * 100
		recommendedTokens := recommendTokensAirdrop(percentageLoss, tokensBefore, totalPool)
		displayName := displayName(cl.Wallet)
		fmt.Printf("%s\n", displayName)
		fmt.Printf("   Realized Loss: %f (%f%%)\n", cl.Loss, percentageLoss)
		fmt.Printf("   %s Tokens (from %s)\n", formatWithCommas(recommendedTokens), formatWithCommas(tokensBefore))
	}

	// for _, cl := range allLosses {
	// 	position := positions[cl.Wallet]
	// 	tokensBefore := position.PositionBefore.Token
	// 	percentageLoss := cl.Loss / totalLosses * 100
	// 	displayName := obfuscateName(cl.Wallet)
	// 	fmt.Printf("%s\n", displayName)
	// 	fmt.Printf("   Realized Loss: %f SOL (%f%% of cohort)\n", cl.Loss, percentageLoss)
	// 	fmt.Printf("   Tokens held at announcement: %s \n", formatWithCommas(tokensBefore))
	// }

	fmt.Println()
	var totalTokensHeldAtAnnouncement float64
	for _, cl := range allLosses {
		position := positions[cl.Wallet]
		tokensBefore := position.PositionBefore.Token
		totalTokensHeldAtAnnouncement += tokensBefore
		percentageLoss := cl.Loss / totalLosses * 100
		recommendedTokens := recommendTokensAirdrop(percentageLoss, tokensBefore, totalPool)
		displayName := cl.Wallet
		fmt.Printf("| %s | | %d | %f | %d | | |\n", displayName, formatWithoutCommas(recommendedTokens), cl.Loss, formatWithoutCommas(cl.NetTransfer))
	}

	fmt.Println()
	for _, winner := range winners {
		position := positions[winner.Wallet]
		tokensBefore := position.PositionBefore.Token
		fmt.Printf("%s %f %d \n", winner.Wallet, winner.Loss, formatWithoutCommas(winner.NetTransfer))
		fmt.Printf("   Tokens held at announcement: %s \n", formatWithCommas(tokensBefore))
	}

	fmt.Println()
	fmt.Println()
	fmt.Println("     Total Held:", formatWithCommas(totalTokensHeldAtAnnouncement))
	fmt.Println()
	fmt.Println()
}

func recommendTokensAirdrop(percentageLoss float64, tokensBefore float64, totalPool float64) float64 {
	recommendedTokens := percentageLoss * totalPool / 100.0
	return min(recommendedTokens, tokensBefore)
}

func percentSupply(amount float64) float64 {
	return amount / 1000000000.0 * 100
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

func displayName(contact string) string {
	if len(contact) > 40 {
		return truncate(contact)
	} else {
		return contact
	}
}

func truncate(walletAddress string) string {
	front := walletAddress[:4]
	back := walletAddress[40:]

	return fmt.Sprintf("%s...%s", front, back)
}

func obfuscateName(contact string) string {
	if len(contact) > 40 {
		return obfuscate(contact)
	} else {
		return contact
	}
}

func obfuscate(walletAddress string) string {
	front := walletAddress[4:8]
	back := walletAddress[36:40]

	return fmt.Sprintf("%s...%s", front, back)
}

func formatWithCommas(n float64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", int64(n))
}

func formatWithoutCommas(n float64) int64 {
	return int64(n)
}

func checkFatal(err error) {
	if err != nil {
		panic(err)
	}
}
