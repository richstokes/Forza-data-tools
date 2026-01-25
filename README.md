# Forza data tools
Building some tools for playing with the UDP data out feature from the Forza Motorsport 7 / Forza Horizon 4 games. Built with [golang](https://golang.org/dl/).  




## Features
- Realtime telemetry output to terminal  
- Telemetry data logging to csv file  
- Serve Forza Telemetry data as JSON over HTTP
- Display race statistics from race/drive (when logging to CSV)  



(Feel free to open an issue if you have any suggestions/feature requests)
&nbsp;

## Prerequisites
Before building this application, you need to have Go installed on your system.

1. **Install Go**: Download and install Go from [https://golang.org/dl/](https://golang.org/dl/)
   - For Windows: Download the `.msi` installer and run it
   - For macOS: Download the `.pkg` installer or use `brew install go`
   - For Linux: Download the archive and follow the installation instructions

2. **Verify Go installation**: Open a terminal/command prompt and run:
   ```
   go version
   ```
   You should see output showing your Go version (e.g., `go version go1.21.3`)

&nbsp;

## Build
To compile the application:

1. **Clone or download this repository**
2. **Open a terminal/command prompt** and navigate to the project directory
3. **Build the application** with:
   ```
   go build -o fdt
   ```
   On Windows, you might want to use:
   ```
   go build -o fdt.exe
   ```

This will create an executable file named `fdt` (or `fdt.exe` on Windows) in the current directory.

### Troubleshooting Build Issues

If you encounter errors while building:

- **"go: command not found"**: Go is not installed or not in your system PATH. Install Go and make sure it's added to your PATH.
- **Module-related errors**: Run `go mod download` to ensure all dependencies are available (though this project uses only the Go standard library).
- **Permission errors**: Make sure you have write permissions in the directory where you're building.

&nbsp;

## Game Setup
After building the application, you need to configure your Forza game:

1. From your game HUD options, enable the **data out** feature
2. Set it to use the **IP address of your computer**
3. Set the port to **9999**
4. For Forza Motorsport 7, select the **"car dash"** format

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