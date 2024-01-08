package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	redisAddr      = "localhost:6379"
	sampleDataPath = "sample3"
)

type Indexes struct {
	Name        map[string][]string `json:"name"`
	Description map[string][]string `json:"description"`
	Price       map[string][]string `json:"price"`
	Categories  map[string][]string `json:"categories"`
}

type Product struct {
	ID          string   `json:"id"`
	Status      string   `json:"status"` // 'available' or 'unavailable'
	Name        string   `json:"name"`
	Price       float64  `json:"price"`
	Categories  []string `json:"categories"`
	Description string   `json:"description"`
}

func sourceData(saveData func(p Product) (bool, error)) ([]Product, error) {
	var products []Product

	file, err := os.Open(sampleDataPath)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	fileScanner := bufio.NewScanner(file)

	fileScanner.Split(bufio.ScanLines)

	for fileScanner.Scan() {
		p := Product{}
		byt := fileScanner.Bytes()

		err := json.Unmarshal(byt, &p)
		if err != nil {
			log.Fatal(err)
		}

		ok, err := saveData(p)
		if !ok {
			log.Fatal(err)
		}

		products = append(products, p)
	}

	return products, nil
}

func writeResult(indexObj *Indexes) {
	f, err := os.Create("indexes")
	if err != nil {
		log.Println(err)
	}
	defer f.Close()

	jsonData, err := json.MarshalIndent(indexObj, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Write the JSON data to the file
	_, err = f.Write(jsonData)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

}

func mountPriceIndex(products []Product, indexObj *Indexes) {
	minRange := 0
	maxRange := 99

	for _, p := range products {
		for p.Price > float64(maxRange) {
			minRange += 100
			maxRange += 100
		}
		priceRange := fmt.Sprintf("%d-%d", minRange, maxRange)
		indexObj.Price[priceRange] = append(indexObj.Price[priceRange], p.ID)

		minRange = 0
		maxRange = 99
	}
}

func mountCategoriesIndex(products []Product, indexObj *Indexes) {
	for _, p := range products {
		for _, c := range p.Categories {
			indexObj.Categories[c] = append(indexObj.Categories[c], p.ID)
		}
	}
}

func parseName(name string) []string {
	return strings.Split(name, " ")
}

func mountNameIndex(products []Product, indexObj *Indexes) {
	for _, p := range products {
		for _, s := range parseName(p.Name) {
			indexObj.Name[s] = append(indexObj.Name[s], p.ID)
		}
	}
}

func parseDescription(description string) []string {
	return strings.Split(description, " ")
}

func mountDescriptionIndex(products []Product, indexObj *Indexes) {
	for _, p := range products {
		for _, s := range parseDescription(p.Description) {
			indexObj.Description[s] = append(indexObj.Description[s], p.ID)
		}
	}
}

func main() {
	db := NewDB(redisAddr)
	defer db.close()

	products, err := sourceData(db.saveProduct)
	if err != nil {
		log.Fatal(err)
	}

	var indexes = Indexes{
		Name:        make(map[string][]string),
		Description: make(map[string][]string),
		Price:       make(map[string][]string),
		Categories:  make(map[string][]string),
	}

	mountNameIndex(products, &indexes)
	mountDescriptionIndex(products, &indexes)
	mountPriceIndex(products, &indexes)
	mountCategoriesIndex(products, &indexes)

	writeResult(&indexes)
}
