package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"
	"log"
	"math"
	"flag"
	// "time"
	// "strconv"
	"os"
	"os/signal"
	"syscall"
	"strings"
	"io/ioutil"
	// "encoding/csv"	
	// "sort"
	"encoding/json"
)

const hostname = "0.0.0.0" // Address to listen on (0.0.0.0 = all interfaces)
const port = "9999" // UDP Port number to listen on
const service = hostname + ":" + port // Combined hostname+port

var jsonData string


type Telemetry struct {
	position int
	name string
	dataType string
	startOffset int
	endOffset int
}

// readForzaData processes recieved UDP packets
func readForzaData(conn *net.UDPConn, telemArray []Telemetry, csvFile string) {
	buffer := make([]byte, 1500)

	n, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		log.Fatal("Error reading UDP data:", err, addr)
	}
	// fmt.Println("UDP client:", addr)
	// fmt.Println("Data from UDP client :  ", string(buffer[:n]))  // Debug: Dump entire received buffer

	// TODO: Check length of received packet:
	// 324 = FH4
	// use this to switch formats? 
	// fmt.Println(len(string(buffer[:n])))

	// Create some maps to store the latest values for each data type
	s32map := make(map[string]uint32)
	u32map := make(map[string]uint32)
	f32map := make(map[string]float32)
	u16map := make(map[string]uint16)
	u8map := make(map[string]uint8)
	s8map := make(map[string]int8)

	// Use Telemetry array to map raw data against Forza's data format
	for _, T := range telemArray {
		data := buffer[:n][T.startOffset:T.endOffset] // Process data in chunks based on byte offsets
		switch T.dataType { // each data type needs to be converted / displayed differently
		case "s32":
			// fmt.Println("Name:", T.name, "Type:", T.dataType, "value:", binary.LittleEndian.Uint32(data)) 
			s32map[T.name] = binary.LittleEndian.Uint32(data)
		case "u32":
			// fmt.Println("Name:", T.name, "Type:", T.dataType, "value:", binary.LittleEndian.Uint32(data))
			u32map[T.name] = binary.LittleEndian.Uint32(data)
		case "f32":
			dataFloated := Float32frombytes(data) // convert raw data bytes into Float32
			f32map[T.name] = dataFloated
			// fmt.Println("Name:", T.name, "Type:", T.dataType, "value:", (dataFloated * 1))
		case "u16":
			u16map[T.name] = binary.LittleEndian.Uint16(data)
			// fmt.Println("Name:", T.name, "Type:", T.dataType, "value:", binary.LittleEndian.Uint16(data))
		case "u8":
			u8map[T.name] = uint8(data[0]) // convert to unsigned int8
		case "s8":
			s8map[T.name] = int8(data[0]) // convert to signed int8
		}
	}

	// Dont print / log / do anything if RPM is zero
	// This happens if the game is paused or you rewind
	// Bug with FH4 where it will continue to send data when in certain menus
	if f32map["CurrentEngineRpm"] == 0 {
		return
	}
	
	// Print received data to terminal:
	if isFlagPassed("d") == false {
		log.Printf(" RPM: %.0f \t Gear: %d \t BHP: %.0f \t Speed: %.0f \t Slip: %.0f", f32map["CurrentEngineRpm"], u8map["Gear"], (f32map["Power"] / 745.7), (f32map["Speed"] * 2.237), (f32map["TireCombinedSlipRearLeft"] + f32map["TireCombinedSlipRearRight"]))
		// log.Println("RPM:", f32map["CurrentEngineRpm"], "Gear:", u8map["Gear"], "BHP:", (f32map["Power"] / 745.7), "Speed:", (f32map["Speed"] * 2.237))
		// fmt.Println("BHP:", (f32map["Power"] / 745.7)) // Convert to BHP
		// fmt.Println("Torque:", (f32map["Torque"] * 0.74)) // Conver to LB-FT
	}

	// Write data to CSV file if enabled:
	if isFlagPassed("c") == true {
		file, err := os.OpenFile(csvFile, os.O_WRONLY|os.O_APPEND, 0644)
		check(err)
		defer file.Close()

		csvLine := ""

		for _, T := range telemArray {  // Construct CSV line
			switch T.dataType {
			case "s32":
				csvLine += "," + fmt.Sprint(s32map[T.name])
			case "u32":
				csvLine += "," + fmt.Sprint(u32map[T.name])
			case "f32":
				csvLine += "," + fmt.Sprint(f32map[T.name])
			case "u16":
				csvLine += "," + fmt.Sprint(u16map[T.name])
			case "u8":
				csvLine += "," + fmt.Sprint(u8map[T.name])
			case "s8":
				csvLine += "," + fmt.Sprint(s8map[T.name])
			case "hzn": // Forza Horizon 4 unknown values
				csvLine += ","
			}
		}
		csvLine += "\n"

		// log.Println(csvLine[1:])
		fmt.Fprintf(file, csvLine[1:])  // write new line to file
	} // end of if CSV enabled

	
	// Send data to JSON server if enabled:
	if isFlagPassed("j") == true {
		var jsonArray [][]byte 

		json1, _ := json.Marshal(s32map)
		jsonString := string(json1)
		jsonArray = append(jsonArray, json1)

		json2, _ := json.Marshal(u32map)
		jsonString2 := string(json2)
		jsonArray = append(jsonArray, json2)

		json3, _ := json.Marshal(f32map)
		jsonString3 := string(json3)
		jsonArray = append(jsonArray, json3)

		json4, _ := json.Marshal(u16map)
		jsonString4 := string(json4)
		jsonArray = append(jsonArray, json4)

		json5, _ := json.Marshal(u8map)
		jsonString5 := string(json5)
		jsonArray = append(jsonArray, json5)

		json6, _ := json.Marshal(s8map)
		jsonString6 := string(json6)
		jsonArray = append(jsonArray, json6)

		// Terrifying JSON hack
		// Probably a much better way to do this, one to look into
		jsonData = fmt.Sprintf("[%s, %s, %s, %s, %s, %s]", jsonString, jsonString2, jsonString3, jsonString4, jsonString5, jsonString6)
		// log.Println(jsonData)
	} // end of if jsonEnabled
}

func main() {
	// Parse flags
	csvFilePtr := flag.String("c", "", "Log data to given file in CSV format")
	horizonPTR := flag.Bool("z", false, "Enables Forza Horizon 4 support (Will default to Forza Motorsport if unset)")
	jsonPTR := flag.Bool("j", false, "Enables JSON server on port 8080")
	noTermPTR := flag.Bool("d", false, "Disables realtime terminal output if set")
	flag.Parse()
	csvFile := *csvFilePtr
	horizonMode := *horizonPTR
	jsonEnabled := *jsonPTR
	noTerm := *noTermPTR

	SetupCloseHandler(csvFile) // handle CTRL+C

	if noTerm {
		log.Println("Terminal data output disabled")
	}

	// Switch to Horizon format if needed
	var formatFile = "FM7_packetformat.dat" // Path to file containing Forzas data format
	if horizonMode {
		formatFile = "FH4_packetformat.dat"
		log.Println("Forza Horizon mode selected")
	} else {
		log.Println("Forza Motorsport mode selected")
	}

	// Load lines from packet format file
	lines, err := readLines(formatFile)
    if err != nil {
        log.Fatalf("Error reading format file: %s", err)
	}

	// Process format file into array of Telemetry structs
	startOffset := 0
	endOffset := 0
	dataLength := 0
	var telemArray []Telemetry

	log.Printf("Processing %s...", formatFile)
	for i, line := range lines {
		dataClean := strings.Split(line, ";") // remove comments after ; from data format file
		dataFormat := strings.Split(dataClean[0], " ") // array containing data type and name
		dataType := dataFormat[0]
		dataName := dataFormat[1]

		switch dataType {
		case "s32": // Signed 32bit int
			dataLength = 4 // Number of bytes
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset} // Create new Telemetry item / data point
			telemArray = append(telemArray, telemItem)
		case "u32": // Unsigned 32bit int
			dataLength = 4 
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset}
			telemArray = append(telemArray, telemItem)
		case "f32": // Floating point 32bit
			dataLength = 4 
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset}
			telemArray = append(telemArray, telemItem)
		case "u16": // Unsigned 16bit int
			dataLength = 2 
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset}
			telemArray = append(telemArray, telemItem)
		case "u8": // Unsigned 8bit int
			dataLength = 1 
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset}
			telemArray = append(telemArray, telemItem)
		case "s8": // Signed 8bit int
			dataLength = 1 
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset}
			telemArray = append(telemArray, telemItem)
		case "hzn": // Forza Horizon 4 unknown values
			dataLength = 12 
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset}
			telemArray = append(telemArray, telemItem)
		default:
			log.Fatalf("Error: Unknown data type in %s \n", formatFile)
		}
		//Debug format file processing:
		//log.Printf("Processed line %d: %s (%s) \t\t Byte offset: %d:%d \n", i, dataName, dataType, startOffset, endOffset)
    }
	log.Printf("Proccessed %d Telemetry types OK!", len(telemArray))
	
	// Prepare CSV file if requested
	if isFlagPassed("c") == true {
		log.Println("Logging data to", csvFile)
		
		csvHeader := ""
		for _, T := range telemArray {  // Construct CSV header/column names
			csvHeader += "," + T.name						
		}
		csvHeader = csvHeader + "\n"
		err := ioutil.WriteFile(csvFile, []byte(csvHeader)[1:], 0644)
		check(err)
		} else {
		log.Println("CSV Logging disabled")
	}

	// Start JSON server if requested
	if jsonEnabled {
		go serveJSON()
	}
	

	// Setup UDP listener
	udpAddr, err := net.ResolveUDPAddr("udp4", service)
	if err != nil {
			log.Fatal(err)
	}

	listener, err := net.ListenUDP("udp", udpAddr)
	check(err)
	defer listener.Close() // close after main ends - probably not really needed
	
	log.Printf("Forza data out server listening on %s, waiting for Forza data...\n", service)

	for { // main loop
		readForzaData(listener, telemArray, csvFile) // Also pass telemArray to UDP function - might be a better way instea do of passing each time?
	}
}

func init() {
	log.SetFlags(log.Lmicroseconds)
	log.Println("Starting Forza Data Tools")
}

// Helper functions

// Run on close (CTRL+C)
func SetupCloseHandler(csvFile string) {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
		<-c
		if isFlagPassed("c") == true { // Get stats if csv logging enabled
			calcstats(csvFile)
		}
		fmt.Println("")
        os.Exit(0)
    }()
}

// Quick error check helper
func check(e error) {
    if e != nil {
        log.Fatalln(e)
    }
}

// Check if flag was passed
func isFlagPassed(name string) bool {
    found := false
    flag.Visit(func(f *flag.Flag) {
        if f.Name == name {
            found = true
        }
    })
    return found
}

// Float32frombytes converts bytes into a float32
func Float32frombytes(bytes []byte) float32 {
    bits := binary.LittleEndian.Uint32(bytes)
    float := math.Float32frombits(bits)
    return float
}

// readLines reads a whole file into memory and returns a slice of its lines
func readLines(path string) ([]string, error) {
    file, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    var lines []string
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        lines = append(lines, scanner.Text())
    }
    return lines, scanner.Err()
}

// Replaced by Sprintf
// func FloatToString(input_num float32) string {
// 	// to convert a float number to a string
// 	to64 := float64(input_num)
//     return strconv.FormatFloat(to64, 'f', 4, 32)
// }