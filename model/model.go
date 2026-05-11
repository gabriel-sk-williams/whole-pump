package model

//
// Shared structs
//

type Holder struct {
	Wallet string
	Tokens float64
}

type WalletEvent struct {
	Source      string
	Type        string
	Timestamp   int64
	SOLAmount   float64
	TokenAmount float64
}

type WalletHistory struct {
	Buys     []WalletEvent
	Sells    []WalletEvent
	Received []Transfer
	Sent     []Transfer
}

type Transfer struct {
	Address     string
	TokenAmount float64
}

type Position struct {
	Transactions  int
	SOL           float64
	Token         float64
	PercentSupply float64
}

type TotalPosition struct {
	PositionBefore Position
	PositionNow    Position
	NetTransfer    float64
}

type ComputedLoss struct {
	Wallet      string
	Loss        float64
	NetTransfer float64
}

type MultiWallet struct {
	Contact string
	Wallets []string
}
