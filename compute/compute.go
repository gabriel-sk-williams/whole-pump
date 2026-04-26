package compute

import (
	"encoding/json"
	"fmt"
	"os"
	"whole-pump/model"
)

// go run main.go compute json/histories.json

func Run(args []string) {
	fmt.Printf("%s running compute \n", args)

	path := args[0]

	histories, err := loadHistories(path)
	checkFatal(err)

	fmt.Println(histories)
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

func checkFatal(err error) {
	if err != nil {
		panic(err)
	}
}
