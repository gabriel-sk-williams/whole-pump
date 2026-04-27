# whole-pump
Open-source scripts for Pump.fun token analysis

Prerequisites
- Go installed
- Helius API key
- Solana token mint address

### Configuration
The tool is configured via environment variables:

HELIUS_API_KEY=Your Helius API key
TOKEN_ADDRESS=The mint address of the token to query

### Usage
`go run main.go snapshot <command> [arguments]`

### Commands
holders — Fetch and display all current holders of the token.
`go run main.go snapshot holders`

affected — Fetch all wallets that have interacted with the token and write them to json/affected.json.
`go run main.go snapshot affected`

history — Fetch full transaction histories for all affected wallets and write them to json/histories.json.
`go run main.go snapshot history`

wallet <address> — Fetch transaction history for a single wallet address.
`go run main.go snapshot wallet <address>`

### Example
```
go run main.go snapshot holders
go run main.go snapshot wallet 9xQeWvG816bUx9EPjHmaT23yvVM2ZWbrrpZb9PusVFin
```


