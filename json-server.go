package main

import (
        // "encoding/json"
        "fmt"
        "log"
        "net/http"
        "strings"
)

const jsonServerPort = ":8080" // Port to serve JSON api on


func responder(w http.ResponseWriter, r *http.Request) {

        switch r.Method {
        case "GET":
                enableCors(&w)
                
                // Check if request is from a browser (Accept header contains text/html)
                acceptHeader := r.Header.Get("Accept")
                if strings.Contains(acceptHeader, "text/html") {
                        // Serve auto-refreshing HTML page for browsers
                        w.Header().Set("Content-Type", "text/html; charset=utf-8")
                        w.Write([]byte(getHTMLPage()))
                } else {
                        // Serve raw JSON for API requests
                        w.Header().Set("Content-Type", "application/json")
                        w.Write([]byte(jsonData))
                }
        default:
                w.WriteHeader(http.StatusMethodNotAllowed)
                fmt.Fprintf(w, "Not supported.")
        }
}

func getHTMLPage() string {
        return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Forza Telemetry Data (Real-time)</title>
    <style>
        body {
            font-family: 'Courier New', monospace;
            background-color: #1e1e1e;
            color: #d4d4d4;
            padding: 20px;
            margin: 0;
        }
        h1 {
            color: #4ec9b0;
            margin-bottom: 10px;
        }
        .info {
            color: #6a9955;
            margin-bottom: 20px;
            font-size: 14px;
        }
        #json-data {
            background-color: #252526;
            border: 1px solid #3e3e42;
            border-radius: 4px;
            padding: 15px;
            overflow: auto;
            max-height: 80vh;
        }
        pre {
            margin: 0;
            white-space: pre-wrap;
            word-wrap: break-word;
        }
        .status {
            position: fixed;
            top: 10px;
            right: 10px;
            padding: 8px 16px;
            border-radius: 4px;
            font-size: 12px;
            font-weight: bold;
        }
        .status.connected {
            background-color: #4ec9b0;
            color: #1e1e1e;
        }
        .status.disconnected {
            background-color: #f48771;
            color: #1e1e1e;
        }
    </style>
</head>
<body>
    <div id="status" class="status connected">● LIVE</div>
    <h1>Forza Telemetry Data</h1>
    <div class="info">Auto-refreshing every 100ms | <a href="/forza" style="color: #4ec9b0;">Raw JSON endpoint</a></div>
    <div id="json-data">
        <pre id="data-content">Waiting for telemetry data...</pre>
    </div>
    
    <script>
        let updateInterval;
        let failCount = 0;
        const maxFails = 5;
        
        function updateData() {
            fetch('/forza', {
                headers: {
                    'Accept': 'application/json'
                }
            })
            .then(response => response.text())
            .then(data => {
                // Try to parse and pretty-print JSON
                try {
                    const jsonObj = JSON.parse(data);
                    document.getElementById('data-content').textContent = JSON.stringify(jsonObj, null, 2);
                    document.getElementById('status').className = 'status connected';
                    document.getElementById('status').textContent = '● LIVE';
                    failCount = 0;
                } catch (e) {
                    document.getElementById('data-content').textContent = data;
                }
            })
            .catch(error => {
                console.error('Error fetching data:', error);
                failCount++;
                if (failCount >= maxFails) {
                    document.getElementById('status').className = 'status disconnected';
                    document.getElementById('status').textContent = '● DISCONNECTED';
                    document.getElementById('data-content').textContent = 'Error: Unable to fetch telemetry data.\n\nPlease check that:\n1. The Forza game is running\n2. Data Out is enabled in game settings\n3. The correct IP and port (9999) are configured';
                }
            });
        }
        
        // Start updating immediately and then every 100ms
        updateData();
        updateInterval = setInterval(updateData, 100);
    </script>
</body>
</html>`
}

func serveJSON() {
        http.HandleFunc("/forza", responder)

        log.Printf("JSON Telemetry Server started at http://localhost%s", jsonServerPort)
        log.Fatal(http.ListenAndServe(jsonServerPort, nil))
}

func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}