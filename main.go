package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"math"
	"net"

	// "time"
	// "strconv"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"

	// "encoding/csv"
	// "sort"
	"encoding/json"
)

const hostname = "0.0.0.0"            // Address to listen on (0.0.0.0 = all interfaces)
const port = "9999"                   // UDP Port number to listen on
const service = hostname + ":" + port // Combined hostname+port

var jsonData string // Stores the JSON data to be sent out via the web server if enabled

// Telemetry struct represents a piece of telemetry as defined in the Forza data format (see the .dat files)
type Telemetry struct {
	position    int
	name        string
	dataType    string
	startOffset int
	endOffset   int
}

// readForzaData processes recieved UDP packets
func readForzaData(conn *net.UDPConn, telemArray []Telemetry, csvFile string) {
	buffer := make([]byte, 1500)

	n, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		log.Fatal("Error reading UDP data:", err, addr)
	}

	if isFlagPassed("d") == true { // Print extra connection info if debugMode set
		log.Println("UDP client connected:", addr)
		// fmt.Printf("Raw Data from UDP client:\n%s", string(buffer[:n])) // Debug: Dump entire received buffer
	}

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
	for i, T := range telemArray {
		data := buffer[:n][T.startOffset:T.endOffset] // Process received data in chunks based on byte offsets

		if isFlagPassed("d") == true { // if debugMode, print received data in each chunk
			log.Printf("Data chunk %d: %v (%s) (%s)", i, data, T.name, T.dataType)
		}

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
	// There is a bug with FH4 where it will continue to send data when in certain menus
	if f32map["CurrentEngineRpm"] == 0 {
		return
	}

	// Print received data to terminal (if not in quiet mode):
	if isFlagPassed("q") == false {
		// Convert slip values to ints as the precision of a float means a neutral state is rarely reported
		totalSlipRear := int(f32map["TireCombinedSlipRearLeft"] + f32map["TireCombinedSlipRearRight"])
		totalSlipFront := int(f32map["TireCombinedSlipFrontLeft"] + f32map["TireCombinedSlipFrontRight"])
		carAttitude := CheckAttitude(totalSlipFront, totalSlipRear)

		log.Printf("RPM: %.0f \t Gear: %d \t BHP: %.0f \t Speed: %.0f \t Total slip: %.0f \t Attitude: %s", f32map["CurrentEngineRpm"], u8map["Gear"], (f32map["Power"] / 745.7), (f32map["Speed"] * 2.237), (f32map["TireCombinedSlipRearLeft"] + f32map["TireCombinedSlipRearRight"]), carAttitude)

		// Testing traction control sensors:
		// log.Printf("TireSlipRatioFrontLeft: %.0f TireSlipRatioFrontRight %.0f", f32map["TireSlipRatioFrontLeft"], f32map["TireSlipRatioFrontRight"])
		// log.Printf("TireSlipAngleFrontLeft: %.0f TireSlipAngleFrontRight %.0f", f32map["TireSlipAngleFrontLeft"], f32map["TireSlipAngleFrontRight"])
		// log.Printf("TireCombinedSlipFrontLeft: %.0f TireCombinedSlipFrontRight %.0f", f32map["TireCombinedSlipFrontLeft"], f32map["TireCombinedSlipFrontRight"])

		// log.Printf("TireSlipRatioRearLeft: %.0f TireSlipRatioRearRight %.0f", f32map["TireSlipRatioRearLeft"], f32map["TireSlipRatioRearRight"])
		// log.Printf("TireSlipAngleRearLeft: %.0f TireSlipAngleRearRight %.0f", f32map["TireSlipAngleRearLeft"], f32map["TireSlipAngleRearRight"])
		// log.Printf("TireCombinedSlipRearLeft: %.0f TireCombinedSlipRearRight %.0f", f32map["TireCombinedSlipRearLeft"], f32map["TireCombinedSlipRearRight"])

		// Testing other sensors
		// log.Printf("AccelerationX: %.0f", f32map["AccelerationX"])
		// log.Printf("AccelerationZ: %.0f", f32map["AccelerationZ"])
		// log.Printf("DistanceTraveled: %.0f", f32map["DistanceTraveled"])

		// "Traction control" if slip higher than threshold and not understeering
		if (totalSlipRear+totalSlipFront) > 2 && carAttitude == "Oversteer" { // Basic traction control detection testing where we allow slip up to a certain amount
			log.Printf("TRACTION LOST!")
		}
	}

	// Write data to CSV file if enabled:
	if isFlagPassed("c") == true {
		file, err := os.OpenFile(csvFile, os.O_WRONLY|os.O_APPEND, 0644)
		check(err)
		defer file.Close()

		csvLine := ""

		for _, T := range telemArray { // Construct CSV line
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
		fmt.Fprintf(file, csvLine[1:]) // write new line to file
	} // end of if CSV enabled

	// Send data to JSON server if enabled:
	if isFlagPassed("j") == true {
		var jsonArray [][]byte

		s32json, _ := json.Marshal(s32map)
		jsonArray = append(jsonArray, s32json)

		u32json, _ := json.Marshal(u32map)
		jsonArray = append(jsonArray, u32json)

		f32json, _ := json.Marshal(f32map)
		jsonArray = append(jsonArray, f32json)

		u16json, _ := json.Marshal(u16map)
		jsonArray = append(jsonArray, u16json)

		u8json, _ := json.Marshal(u8map)
		jsonArray = append(jsonArray, u8json)

		s8json, _ := json.Marshal(s8map)
		jsonArray = append(jsonArray, s8json)

		var jd []string
		for i, j := range jsonArray { // concatenate json objects
			if i == 0 {
				jd = append(jd, string(j))
			} else {
				jd = append(jd, ", "+string(j))
			}

		}

		jsonData = fmt.Sprintf("%s", jd)

	} // end of if jsonEnabled
}

func main() {
	// Parse flags
	csvFilePtr := flag.String("c", "", "Log data to given file in CSV format")
	horizonPTR := flag.Bool("z", false, "Enables Forza Horizon 4 support (Will default to Forza Motorsport if unset)")
	jsonPTR := flag.Bool("j", false, "Enables JSON HTTP server on port 8080")
	noTermPTR := flag.Bool("q", false, "Disables realtime terminal output if set")
	debugModePTR := flag.Bool("d", false, "Enables extra debug information if set")
	flag.Parse()
	csvFile := *csvFilePtr
	horizonMode := *horizonPTR
	jsonEnabled := *jsonPTR
	noTerm := *noTermPTR
	debugMode := *debugModePTR

	SetupCloseHandler(csvFile) // handle CTRL+C

	if debugMode {
		log.Println("Debug mode enabled")
	}

	if noTerm {
		log.Println("Realtime terminal data output disabled")
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
		dataClean := strings.Split(line, ";")          // remove comments after ; from data format file
		dataFormat := strings.Split(dataClean[0], " ") // array containing data type and name
		dataType := dataFormat[0]
		dataName := dataFormat[1]

		switch dataType {
		case "s32": // Signed 32bit int
			dataLength = 4 // Number of bytes
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset} // Create new Telemetry item / data point
			telemArray = append(telemArray, telemItem)                            // Add Telemetry item to main telemetry array
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
		case "hzn": // Forza Horizon 4 unknown values (12 bytes of.. something)
			dataLength = 12
			endOffset = endOffset + dataLength
			startOffset = endOffset - dataLength
			telemItem := Telemetry{i, dataName, dataType, startOffset, endOffset}
			telemArray = append(telemArray, telemItem)
		default:
			log.Fatalf("Error: Unknown data type in %s \n", formatFile)
		}
		//Debug format file processing:
		if debugMode {
			log.Printf("Processed %s line %d: %s (%s),  Byte offset: %d:%d \n", formatFile, i, dataName, dataType, startOffset, endOffset)
		}
	}

	if debugMode { // Print completed telemetry array
		log.Printf("Logging entire telemArray: \n%v", telemArray)
	}

	log.Printf("Proccessed %d Telemetry types OK!", len(telemArray))

	// Prepare CSV file if requested
	if isFlagPassed("c") == true {
		log.Println("Logging data to", csvFile)

		csvHeader := ""
		for _, T := range telemArray { // Construct CSV header/column names
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

	// log.Printf("Forza data out server listening on %s, waiting for Forza data...\n", service)
	log.Printf("Forza data out server listening on %s:%s, waiting for Forza data...\n", GetOutboundIP(), port)

	for { // main loop
		readForzaData(listener, telemArray, csvFile) // Also pass telemArray to UDP function - might be a better way instea do of passing each time?
	}
}

func init() {
	log.SetFlags(log.Lmicroseconds)
	log.Println("Started Forza Data Tools")
}

// Helper functions

// SetupCloseHandler performs some clean up on exit (CTRL+C)
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

// GetOutboundIP finds preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "1.2.3.4:4321") // Destination does not need to exist, using this to see which is the primary network interface
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

// CheckAttitude looks for balance of the car
func CheckAttitude(totalSlipFront int, totalSlipRear int) string {
	// Check attitude of car by comparing front and rear slip levels
	// If front slip > rear slip, means car is understeering
	if totalSlipRear > totalSlipFront {
		// log.Printf("Car is oversteering")
		return "Oversteer"
	} else if totalSlipFront > totalSlipRear {
		// log.Printf("Car is understeering")
		return "Understeer"
	} else {
		// log.Printf("Car balance is neutral")
		return "Neutral"
	}

}
