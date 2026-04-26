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
