package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"sort"
)

func calcstats(csvFile string) {
	rows := readLog(csvFile)
	// check data is received before doing anything, else will crash due to no data in the csv file
	if len(rows) > 1 {
		log.Printf("Recorded %d data points!", len(rows))
		rows = calculate(rows)
	}
}

func readLog(name string) [][]string {

	f, err := os.Open(name)
	// Usually we would return the error to the caller and handle
	// all errors in function `main()`. However, this is just a
	// small command-line tool, and so we use `log.Fatal()`
	// instead, in order to write the error message to the
	// terminal and exit immediately.
	if err != nil {
		log.Fatalf("Cannot open '%s': %s\n", name, err.Error())
	}

	// After this point, the file has been successfully opened,
	// and we want to ensure that it gets closed when no longer
	// needed, so we add a deferred call to `f.Close()`.
	defer f.Close()

	// To read in the CSV data, we create a new CSV reader that
	// reads from the input file.
	//
	// The CSV reader is aware of the CSV data format. It
	// separates the input stream into rows and columns,
	// and returns a slice of slices of strings.
	r := csv.NewReader(f)

	// We can even adjust the reader to recognize a semicolon,
	// rather than a comma, as the column separator.
	r.Comma = ','

	// Read the whole file at once. (We don't expect large files.)
	rows, err := r.ReadAll()

	// Again, we check for any error,
	if err != nil {
		log.Fatalln("Cannot read CSV data:", err.Error())
	}

	// and finally we can return the rows.
	return rows
}

// calculate stats
func calculate(rows [][]string) [][]string {
	// Find row numbers based on column header names (row 0)
	speedRow := 0
	boostRow := 0

	for k, v := range rows[0] {
		if v == "Speed" {
			speedRow = k
		} else if v == "Boost" {
			boostRow = k
		} 
	}

	var s []float64 // array of speed values
	var b []float64 // array of boost values
	// fmt.Printf("%T\n", s) // print type

	for i := range rows {

		if i == 0 { // skip first row (header/column names)
			continue
		}

		// Add speed value from row to array
		speed, err := strconv.ParseFloat(rows[i][speedRow], 32) // convert speed string to int
		check(err)
		s = append(s, (speed * 2.237)) // convert to MPH

		// Add boost value from row to array
		boost, err := strconv.ParseFloat(rows[i][boostRow], 32) // convert boost string to int
		check(err)
		b = append(b, boost) // convert to PSI, not 100% sure what value this is natively?
	}

	var totalSpeed float64
	for _, value:= range s {
		totalSpeed += value // add all speed values together for getting average later
	}

	fmt.Printf("\nRace statistics:\n")

	// Get average speed
	fmt.Printf("Average speed: %.2f MPH \n", totalSpeed/float64(len(s)))  // truncate to 2 decimal places

	// Get top speed
	// fmt.Println(s)
	sort.Float64s(s)
	topSpeed := s[len(s)-1]
	fmt.Printf("Top speed: %.2f MPH \n", topSpeed)

	// Get peak boost
	sort.Float64s(b)
	topBoost := b[len(b)-1]
	fmt.Printf("Peak boost: %.2f PSI \n", topBoost)

	return rows
}

// `intToFloatString` takes an integer `n` and calculates the floating point value representing `n/100` as a string.
// func intToFloatString(n int) string {
// 	intgr := n / 100
// 	frac := n - intgr*100
// 	return fmt.Sprintf("%d.%d", intgr, frac)
// }

// func printSlice(s []int) {
// 	fmt.Printf("len=%d cap=%d %v\n", len(s), cap(s), s)
// }