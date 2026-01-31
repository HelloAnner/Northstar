package main

import (
	"fmt"
	"os"
	"path/filepath"

	"northstar/internal/importer"
	"northstar/internal/store"
)

func main() {
	dataDir := os.Getenv("NS_DATA_DIR")
	if dataDir == "" {
		panic("NS_DATA_DIR is required")
	}

	dbPath := filepath.Join(dataDir, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		panic(err)
	}
	defer func() { _ = st.Close() }()

	coord := importer.NewCoordinator(st)
	filePath := filepath.Join("prd", "12月月报（预估）_补全企业名称社会代码_20260129.xlsx")
	ch := coord.Import(importer.ImportOptions{
		FilePath:         filePath,
		OriginalFilename: filepath.Base(filePath),
		ClearExisting:    true,
		UpdateConfigYM:   true,
		CalculateFields:  true,
	})

	for evt := range ch {
		if evt.Type == "error" {
			panic(evt.Message)
		}
	}

	fmt.Println("import done", dbPath)
}
