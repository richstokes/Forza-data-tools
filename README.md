# Forza data tools
Building some tools for playing with the UDP data out feature from the Forza Motorsport 7 / Forza Horizon 4 games. Built with [golang](https://golang.org/dl/).  



## Prerequisites

Before you can build and run this application, you'll need to install Go (also known as Golang).

### What is Go?
Go is a programming language developed by Google. This project is written in Go, so you'll need the Go compiler installed on your computer to build and run it.

### Installing Go
1. **Download Go**: Visit [https://golang.org/dl/](https://golang.org/dl/) and download the installer for your operating system (Windows, macOS, or Linux)
2. **Install Go**: Run the installer and follow the installation instructions for your platform
3. **Verify Installation**: Open a terminal (or Command Prompt on Windows) and run:
   ```bash
   go version
   ```
   You should see output like `go version go1.21.3 ...` or similar

### Minimum Requirements
- **Go version 1.21.3 or higher** is required for this project
- If your version is older, download and install the latest version from the link above

### Learn More About Go
- [Official Go Documentation](https://golang.org/doc/)
- [A Tour of Go](https://tour.golang.org/) - Interactive introduction to Go
- [How to Write Go Code](https://golang.org/doc/code.html) - Official guide

&nbsp;

## Getting Started

Follow these steps to get the Forza data tools up and running:

### 1. Get the Code
Clone this repository or download it as a ZIP file:
```bash
git clone https://github.com/richstokes/Forza-data-tools.git
```

Or download the ZIP file from GitHub and extract it to a folder on your computer.

### 2. Navigate to the Project Directory
Open a terminal (or Command Prompt on Windows) and navigate to the project folder:
```bash
cd Forza-data-tools
```

### 3. Build the Application
Compile the application using the Go compiler:
```bash
go build -o fdt
```

**What this does**: The `go build` command compiles all the Go source code files into a single executable program. The `-o fdt` flag tells Go to name the output file `fdt` (or `fdt.exe` on Windows).

**Note**: This project has no external dependencies, so there's no need to run `go mod download` or similar commands. The build should complete in a few seconds.

### 4. Configure Forza Game Settings
From your Forza game HUD options:
1. Enable the "Data Out" feature
2. Set it to use the IP address of your computer
3. Set the port to **9999**
4. For Forza Motorsport 7, select the **"car dash"** format

### 5. Run the Application
Run the compiled application (see [Command line options](#command-line-options) below for configuration):
```bash
./fdt
```

On Windows:
```cmd
fdt.exe
```

### Troubleshooting

**"go: command not found" or "'go' is not recognized"**
- Go is not installed or not in your system PATH
- Solution: Make sure you've installed Go and restarted your terminal/command prompt after installation

**Build fails with "package ... is not in GOROOT"**
- Your Go version might be too old
- Solution: Update to Go 1.21.3 or higher

**"bind: address already in use" when running with `-j` flag**
- Port 8080 is already being used by another application
- Solution: Close the other application, or modify the `jsonServerPort` constant in the `json-server.go` file to use a different port

**No data appearing in the terminal**
- Check that Forza's "Data Out" feature is enabled and configured correctly
- Verify the IP address matches your computer's local IP
- Ensure port 9999 is not blocked by your firewall

&nbsp;


## Features
- Realtime telemetry output to terminal  
- Telemetry data logging to csv file  
- Serve Forza Telemetry data as JSON over HTTP
- Display race statistics from race/drive (when logging to CSV)  



(Feel free to open an issue if you have any suggestions/feature requests)
&nbsp;

## Run
### Command line options
Specify a CSV file to log to: `-c log.csv` (File will be overwritten if it exists)    
Enable support for Forza Horizon: `-z`    
Enable JSON server: `-j`   
Disable realtime terminal output: `-q`   
Enable debug information: `-d`

&nbsp;

##### Example (Forza Horizon)
`fdt -z -j -c log.csv`  
`fdt -z`  

##### Example (Forza Motorsport)
`fdt -c -j log.csv`  

&nbsp;

### JSON Data
If the `-j` flag is provided, JSON data will be available at: http://localhost:8080/forza. Could be used to make a web dashboard interface or something similar. JSON Format is an array of objects containing the various Forza data types.  

You can see a sample of the kind of data that will be returned [here](https://github.com/richstokes/Forza-data-tools/blob/master/dash/sample.json).  

There is a basic example JavaScript dashboard (with rev limiter function) in the `/dash` directory.  

&nbsp; 

## Further reading
- Forza data out format: https://forums.forzamotorsport.net/turn10_postsm926839_Forza-Motorsport-7--Data-Out--feature-details.aspx#post_926839

- Forza Horizon 4 has some mystery data in the packet, waiting on info from the developers: https://forums.forzamotorsport.net/turn10_postsm1086012_Data-Output.aspx#post_1086012