package main

import (
        // "encoding/json"
        "fmt"
        "log"
        "net/http"
        "strings"
)

const jsonServerPort = ":8080" // Port to serve JSON api on


// responder handles HTTP requests to the /forza endpoint.
// For browser requests (Accept: text/html), it serves an interactive dashboard.
// For API requests, it returns raw JSON telemetry data.
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

// getHTMLPage returns the HTML content for the interactive telemetry dashboard,
// including gauges for RPM, speed, pedal inputs, boost, tire temps, and lap times.
func getHTMLPage() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Forza Telemetry Dashboard</title>
    <style>
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
            color: #fff;
            min-height: 100vh;
            padding: 20px;
        }
        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 24px;
            padding-bottom: 16px;
            border-bottom: 1px solid rgba(255,255,255,0.1);
        }
        h1 {
            font-size: 24px;
            font-weight: 600;
            background: linear-gradient(90deg, #00d4ff, #7b2cbf);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }
        .status {
            padding: 6px 14px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 600;
            display: flex;
            align-items: center;
            gap: 6px;
        }
        .status.connected { background: rgba(0, 255, 136, 0.2); color: #00ff88; }
        .status.disconnected { background: rgba(255, 71, 87, 0.2); color: #ff4757; }
        .status::before { content: '●'; }
        
        .dashboard { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; }
        
        .card {
            background: rgba(255,255,255,0.05);
            border-radius: 16px;
            padding: 24px;
            backdrop-filter: blur(10px);
            border: 1px solid rgba(255,255,255,0.1);
        }
        .card-title {
            font-size: 12px;
            text-transform: uppercase;
            letter-spacing: 1px;
            color: rgba(255,255,255,0.5);
            margin-bottom: 16px;
        }
        
        /* RPM Gauge */
        .rpm-container { text-align: center; }
        .rpm-gauge {
            position: relative;
            width: 220px;
            height: 110px;
            margin: 0 auto 16px;
        }
        .rpm-arc {
            fill: none;
            stroke-width: 12;
            stroke-linecap: round;
        }
        .rpm-bg { stroke: rgba(255,255,255,0.1); }
        .rpm-fill { stroke: url(#rpmGradient); transition: stroke-dashoffset 0.1s ease; }
        .rpm-value {
            font-size: 48px;
            font-weight: 700;
            line-height: 1;
        }
        .rpm-label { font-size: 14px; color: rgba(255,255,255,0.5); }
        
        /* Gear Display */
        .gear-display {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 20px;
        }
        .gear-number {
            font-size: 96px;
            font-weight: 800;
            line-height: 1;
            background: linear-gradient(180deg, #fff 0%, #888 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }
        .gear-label { font-size: 14px; color: rgba(255,255,255,0.5); }
        
        /* Speed Display */
        .speed-container { text-align: center; }
        .speed-value {
            font-size: 72px;
            font-weight: 700;
            line-height: 1;
        }
        .speed-unit { font-size: 18px; color: rgba(255,255,255,0.5); margin-left: 8px; }
        
        /* Linear Gauges */
        .linear-gauge { margin-bottom: 16px; }
        .linear-gauge:last-child { margin-bottom: 0; }
        .gauge-header {
            display: flex;
            justify-content: space-between;
            margin-bottom: 8px;
            font-size: 13px;
        }
        .gauge-label { color: rgba(255,255,255,0.7); }
        .gauge-value { font-weight: 600; }
        .gauge-track {
            height: 8px;
            background: rgba(255,255,255,0.1);
            border-radius: 4px;
            overflow: hidden;
        }
        .gauge-fill {
            height: 100%;
            border-radius: 4px;
            transition: width 0.1s ease;
        }
        .gauge-throttle { background: linear-gradient(90deg, #00d4ff, #00ff88); }
        .gauge-brake { background: linear-gradient(90deg, #ff6b6b, #ff4757); }
        .gauge-clutch { background: linear-gradient(90deg, #ffd93d, #ff9f43); }
        
        /* Tire Temps */
        .tire-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 12px;
        }
        .tire {
            background: rgba(0,0,0,0.3);
            border-radius: 8px;
            padding: 12px;
            text-align: center;
        }
        .tire-temp {
            font-size: 24px;
            font-weight: 700;
        }
        .tire-label {
            font-size: 10px;
            color: rgba(255,255,255,0.5);
            text-transform: uppercase;
        }
        .tire.cold .tire-temp { color: #74b9ff; }
        .tire.optimal .tire-temp { color: #00ff88; }
        .tire.hot .tire-temp { color: #ff9f43; }
        .tire.overheat .tire-temp { color: #ff4757; }
        
        /* Boost Gauge */
        .boost-container { text-align: center; }
        .boost-value {
            font-size: 48px;
            font-weight: 700;
        }
        .boost-value.vacuum { color: #74b9ff; }
        .boost-value.boost { color: #00ff88; }
        .boost-bar {
            height: 12px;
            background: rgba(255,255,255,0.1);
            border-radius: 6px;
            margin-top: 12px;
            position: relative;
            overflow: hidden;
        }
        .boost-bar-fill {
            position: absolute;
            height: 100%;
            border-radius: 6px;
            transition: all 0.1s ease;
        }
        .boost-bar-vacuum {
            right: 50%;
            background: linear-gradient(270deg, #74b9ff, #0984e3);
        }
        .boost-bar-positive {
            left: 50%;
            background: linear-gradient(90deg, #00ff88, #00b894);
        }
        .boost-bar-center {
            position: absolute;
            left: 50%;
            top: -4px;
            bottom: -4px;
            width: 2px;
            background: rgba(255,255,255,0.3);
            transform: translateX(-50%);
        }
        
        /* Car Info */
        .info-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 12px;
        }
        .info-item {
            background: rgba(0,0,0,0.2);
            border-radius: 8px;
            padding: 12px;
        }
        .info-label {
            font-size: 10px;
            color: rgba(255,255,255,0.5);
            text-transform: uppercase;
            margin-bottom: 4px;
        }
        .info-value { font-size: 18px; font-weight: 600; }
        
        /* Waiting State */
        .waiting {
            text-align: center;
            padding: 60px 20px;
            color: rgba(255,255,255,0.5);
        }
        .waiting-icon {
            font-size: 48px;
            margin-bottom: 16px;
            animation: pulse 2s infinite;
        }
        @keyframes pulse {
            0%, 100% { opacity: 0.5; }
            50% { opacity: 1; }
        }
        
        .raw-toggle {
            margin-top: 20px;
            text-align: center;
        }
        .raw-toggle a {
            color: rgba(255,255,255,0.5);
            font-size: 12px;
            text-decoration: none;
        }
        .raw-toggle a:hover { color: #00d4ff; }
    </style>
</head>
<body>
    <div class="header">
        <h1>Forza Telemetry</h1>
        <div id="status" class="status connected">LIVE</div>
    </div>
    
    <div id="waiting" class="waiting">
        <div class="waiting-icon">🏎️</div>
        <div>Waiting for telemetry data...</div>
        <div style="margin-top: 8px; font-size: 12px;">Make sure Forza is running with Data Out enabled</div>
    </div>
    
    <div id="dashboard" class="dashboard" style="display: none;">
        <!-- RPM Gauge -->
        <div class="card">
            <div class="card-title">Engine RPM</div>
            <div class="rpm-container">
                <div class="rpm-gauge">
                    <svg viewBox="0 0 220 110" width="220" height="110">
                        <defs>
                            <linearGradient id="rpmGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                                <stop offset="0%" stop-color="#00d4ff"/>
                                <stop offset="70%" stop-color="#00ff88"/>
                                <stop offset="100%" stop-color="#ff4757"/>
                            </linearGradient>
                        </defs>
                        <path class="rpm-arc rpm-bg" d="M 20 100 A 90 90 0 0 1 200 100"/>
                        <path id="rpmFill" class="rpm-arc rpm-fill" d="M 20 100 A 90 90 0 0 1 200 100" stroke-dasharray="283" stroke-dashoffset="283"/>
                    </svg>
                </div>
                <div class="rpm-value"><span id="rpmValue">0</span></div>
                <div class="rpm-label">RPM</div>
            </div>
        </div>
        
        <!-- Gear & Speed -->
        <div class="card">
            <div class="card-title">Gear & Speed</div>
            <div class="gear-display">
                <div>
                    <div id="gearValue" class="gear-number">N</div>
                    <div class="gear-label">GEAR</div>
                </div>
                <div class="speed-container">
                    <span id="speedValue" class="speed-value">0</span>
                    <span class="speed-unit">MPH</span>
                    <div class="gear-label" style="margin-top: 4px;">SPEED</div>
                </div>
            </div>
        </div>
        
        <!-- Pedals -->
        <div class="card">
            <div class="card-title">Pedal Inputs</div>
            <div class="linear-gauge">
                <div class="gauge-header">
                    <span class="gauge-label">Throttle</span>
                    <span id="throttleValue" class="gauge-value">0%</span>
                </div>
                <div class="gauge-track">
                    <div id="throttleFill" class="gauge-fill gauge-throttle" style="width: 0%"></div>
                </div>
            </div>
            <div class="linear-gauge">
                <div class="gauge-header">
                    <span class="gauge-label">Brake</span>
                    <span id="brakeValue" class="gauge-value">0%</span>
                </div>
                <div class="gauge-track">
                    <div id="brakeFill" class="gauge-fill gauge-brake" style="width: 0%"></div>
                </div>
            </div>
            <div class="linear-gauge">
                <div class="gauge-header">
                    <span class="gauge-label">Clutch</span>
                    <span id="clutchValue" class="gauge-value">0%</span>
                </div>
                <div class="gauge-track">
                    <div id="clutchFill" class="gauge-fill gauge-clutch" style="width: 0%"></div>
                </div>
            </div>
        </div>
        
        <!-- Boost -->
        <div class="card">
            <div class="card-title">Boost / Vacuum</div>
            <div class="boost-container">
                <div id="boostValue" class="boost-value vacuum">0.0</div>
                <div class="rpm-label">PSI</div>
                <div class="boost-bar">
                    <div class="boost-bar-center"></div>
                    <div id="boostVacuum" class="boost-bar-fill boost-bar-vacuum" style="width: 0%"></div>
                    <div id="boostPositive" class="boost-bar-fill boost-bar-positive" style="width: 0%"></div>
                </div>
            </div>
        </div>
        
        <!-- Tire Temps -->
        <div class="card">
            <div class="card-title">Tire Temperature (°F)</div>
            <div class="tire-grid">
                <div id="tireFL" class="tire optimal">
                    <div class="tire-label">Front Left</div>
                    <div id="tireFLTemp" class="tire-temp">--</div>
                </div>
                <div id="tireFR" class="tire optimal">
                    <div class="tire-label">Front Right</div>
                    <div id="tireFRTemp" class="tire-temp">--</div>
                </div>
                <div id="tireRL" class="tire optimal">
                    <div class="tire-label">Rear Left</div>
                    <div id="tireRLTemp" class="tire-temp">--</div>
                </div>
                <div id="tireRR" class="tire optimal">
                    <div class="tire-label">Rear Right</div>
                    <div id="tireRRTemp" class="tire-temp">--</div>
                </div>
            </div>
        </div>
        
        <!-- Car Info -->
        <div class="card">
            <div class="card-title">Session Info</div>
            <div class="info-grid">
                <div class="info-item">
                    <div class="info-label">Position</div>
                    <div id="positionValue" class="info-value">--</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Lap</div>
                    <div id="lapValue" class="info-value">--</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Best Lap</div>
                    <div id="bestLapValue" class="info-value">--</div>
                </div>
                <div class="info-item">
                    <div class="info-label">Last Lap</div>
                    <div id="lastLapValue" class="info-value">--</div>
                </div>
            </div>
        </div>
    </div>
    
    <div class="raw-toggle">
        <a href="/forza" target="_blank">View Raw JSON →</a>
    </div>
    
    <script>
        let failCount = 0;
        const maxFails = 5;
        let maxRpm = 8000;
        
        function formatLapTime(seconds) {
            if (!seconds || seconds <= 0) return '--';
            const mins = Math.floor(seconds / 60);
            const secs = (seconds % 60).toFixed(3);
            return mins + ':' + secs.padStart(6, '0');
        }
        
        function getTireClass(tempF) {
            if (tempF < 150) return 'cold';
            if (tempF < 220) return 'optimal';
            if (tempF < 280) return 'hot';
            return 'overheat';
        }
        
        function celsiusToFahrenheit(c) {
            return (c * 9/5) + 32;
        }
        
        function updateDashboard(data) {
            const d = {};
            if (Array.isArray(data)) {
                data.forEach(obj => Object.assign(d, obj));
            } else {
                Object.assign(d, data);
            }
            
            document.getElementById('waiting').style.display = 'none';
            document.getElementById('dashboard').style.display = 'grid';
            
            // RPM
            const rpm = d.CurrentEngineRpm || 0;
            maxRpm = Math.max(maxRpm, d.EngineMaxRpm || 8000);
            const rpmPercent = Math.min(rpm / maxRpm, 1);
            document.getElementById('rpmValue').textContent = Math.round(rpm);
            document.getElementById('rpmFill').style.strokeDashoffset = 283 - (283 * rpmPercent);
            
            // Gear
            const gear = d.Gear;
            let gearDisplay = 'N';
            if (gear === 0) gearDisplay = 'R';
            else if (gear > 0) gearDisplay = gear.toString();
            document.getElementById('gearValue').textContent = gearDisplay;
            
            // Speed (m/s to MPH)
            const speedMph = (d.Speed || 0) * 2.237;
            document.getElementById('speedValue').textContent = Math.round(speedMph);
            
            // Pedals (0-255 range)
            const throttle = ((d.Accel || 0) / 255) * 100;
            const brake = ((d.Brake || 0) / 255) * 100;
            const clutch = ((d.Clutch || 0) / 255) * 100;
            
            document.getElementById('throttleValue').textContent = Math.round(throttle) + '%';
            document.getElementById('throttleFill').style.width = throttle + '%';
            document.getElementById('brakeValue').textContent = Math.round(brake) + '%';
            document.getElementById('brakeFill').style.width = brake + '%';
            document.getElementById('clutchValue').textContent = Math.round(clutch) + '%';
            document.getElementById('clutchFill').style.width = clutch + '%';
            
            // Boost
            const boost = d.Boost || 0;
            const boostEl = document.getElementById('boostValue');
            boostEl.textContent = boost.toFixed(1);
            boostEl.className = boost >= 0 ? 'boost-value boost' : 'boost-value vacuum';
            
            if (boost < 0) {
                document.getElementById('boostVacuum').style.width = Math.min(Math.abs(boost) / 15 * 50, 50) + '%';
                document.getElementById('boostPositive').style.width = '0%';
            } else {
                document.getElementById('boostVacuum').style.width = '0%';
                document.getElementById('boostPositive').style.width = Math.min(boost / 30 * 50, 50) + '%';
            }
            
            // Tire temps (C to F)
            const tires = [
                { id: 'FL', temp: d.TireTempFrontLeft },
                { id: 'FR', temp: d.TireTempFrontRight },
                { id: 'RL', temp: d.TireTempRearLeft },
                { id: 'RR', temp: d.TireTempRearRight }
            ];
            
            tires.forEach(tire => {
                const tempF = celsiusToFahrenheit(tire.temp || 0);
                document.getElementById('tire' + tire.id + 'Temp').textContent = Math.round(tempF);
                document.getElementById('tire' + tire.id).className = 'tire ' + getTireClass(tempF);
            });
            
            // Session info
            const position = d.RacePosition || 0;
            document.getElementById('positionValue').textContent = position > 0 ? position : '--';
            document.getElementById('lapValue').textContent = (d.LapNumber || 0) + 1;
            document.getElementById('bestLapValue').textContent = formatLapTime(d.BestLap);
            document.getElementById('lastLapValue').textContent = formatLapTime(d.LastLap);
        }
        
        function updateData() {
            fetch('/forza', { headers: { 'Accept': 'application/json' } })
            .then(response => response.text())
            .then(data => {
                if (!data || data.trim() === '' || data === '[]') {
                    throw new Error('No data');
                }
                const jsonObj = JSON.parse(data);
                updateDashboard(jsonObj);
                document.getElementById('status').className = 'status connected';
                document.getElementById('status').textContent = 'LIVE';
                failCount = 0;
            })
            .catch(error => {
                failCount++;
                if (failCount >= maxFails) {
                    document.getElementById('status').className = 'status disconnected';
                    document.getElementById('status').textContent = 'DISCONNECTED';
                }
            });
        }
        
        updateData();
        setInterval(updateData, 100);
    </script>
</body>
</html>`
}

// serveJSON starts the HTTP server on port 8080 to serve telemetry data.
// The /forza endpoint serves either a web dashboard or raw JSON based on the request.
func serveJSON() {
        http.HandleFunc("/forza", responder)

        log.Printf("JSON Telemetry Server started at http://localhost%s", jsonServerPort)
        log.Fatal(http.ListenAndServe(jsonServerPort, nil))
}

// enableCors sets the Access-Control-Allow-Origin header to allow cross-origin requests.
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}