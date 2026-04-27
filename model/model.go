package model

//
// Shared structs
//

type WalletEvent struct {
	Source      string
	Slot        int64
	Timestamp   int64
	SOLAmount   float64
	TokenAmount float64
}

type WalletHistory struct {
	Buys  []WalletEvent
	Sells []WalletEvent
}

type Position struct {
	Wallet        string
	Transactions  int
	SOL           float64
	Token         float64
	PercentSupply float64
}
