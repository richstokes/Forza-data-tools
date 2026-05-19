package main

import (
	// "encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const jsonServerPort = ":8080" // Port to serve JSON api on

var telemetry = &telemetryHub{
	latest:  []byte("[]"),
	clients: make(map[*telemetryClient]struct{}),
}

var websocketUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type telemetryClient struct {
	conn *websocket.Conn
	send chan []byte
}

type telemetryHub struct {
	mu      sync.RWMutex
	latest  []byte
	clients map[*telemetryClient]struct{}
}

func (h *telemetryHub) add(client *telemetryClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[client] = struct{}{}
}

func (h *telemetryHub) remove(client *telemetryClient) {
	h.mu.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	h.mu.Unlock()

	client.conn.Close()
}

func (h *telemetryHub) snapshot() []byte {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return append([]byte(nil), h.latest...)
}

func (h *telemetryHub) publish(payload []byte) {
	payload = append([]byte(nil), payload...)

	h.mu.Lock()
	defer h.mu.Unlock()

	h.latest = payload
	for client := range h.clients {
		select {
		case client.send <- payload:
		default:
			select {
			case <-client.send:
			default:
			}
			select {
			case client.send <- payload:
			default:
			}
		}
	}
}

func publishJSONData(payload []byte) {
	telemetry.publish(payload)
}

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
			w.Write(telemetry.snapshot())
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Not supported.")
	}
}

func jsonResponder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Not supported.")
		return
	}

	enableCors(&w)
	w.Header().Set("Content-Type", "application/json")
	w.Write(telemetry.snapshot())
}

func websocketResponder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Not supported.")
		return
	}

	conn, err := websocketUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &telemetryClient{
		conn: conn,
		send: make(chan []byte, 1),
	}
	telemetry.add(client)

	if latest := telemetry.snapshot(); len(latest) > 0 && string(latest) != "[]" {
		client.send <- latest
	}

	go client.writePump()
	client.readPump()
}

func (client *telemetryClient) readPump() {
	defer telemetry.remove(client)

	client.conn.SetReadLimit(512)
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		if _, _, err := client.conn.NextReader(); err != nil {
			return
		}
	}
}

func (client *telemetryClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		telemetry.remove(client)
	}()

	for {
		select {
		case payload, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				return
			}
		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
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

        /* Slip Angle */
        .slip-container { text-align: center; }
        .slip-value {
            font-size: 52px;
            font-weight: 700;
            line-height: 1;
        }
        .slip-direction {
            min-height: 20px;
            margin-top: 6px;
            font-size: 13px;
            color: rgba(255,255,255,0.55);
            text-transform: uppercase;
        }
        .slip-track {
            position: relative;
            height: 16px;
            margin: 24px 4px 10px;
            overflow: hidden;
            border-radius: 8px;
            background: rgba(255,255,255,0.1);
        }
        .slip-track::before {
            content: '';
            position: absolute;
            top: 0;
            bottom: 0;
            left: 50%;
            width: 2px;
            background: rgba(255,255,255,0.35);
            transform: translateX(-50%);
        }
        .slip-fill {
            position: absolute;
            top: 0;
            bottom: 0;
            width: 0;
            transition: width 0.1s ease;
        }
        .slip-fill-left {
            right: 50%;
            background: linear-gradient(270deg, #74b9ff, #00d4ff);
        }
        .slip-fill-right {
            left: 50%;
            background: linear-gradient(90deg, #00d4ff, #ff9f43);
        }
        .slip-needle {
            position: absolute;
            top: -6px;
            left: 50%;
            width: 4px;
            height: 28px;
            border-radius: 2px;
            background: #fff;
            box-shadow: 0 0 12px rgba(0,212,255,0.75);
            transform: translateX(-50%);
            transition: left 0.1s ease;
        }
        .slip-scale {
            display: flex;
            justify-content: space-between;
            color: rgba(255,255,255,0.4);
            font-size: 11px;
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

        <!-- Slip Angle -->
        <div class="card">
            <div class="card-title">Overall Slip Angle</div>
            <div class="slip-container">
                <div id="slipAngleValue" class="slip-value">0.0°</div>
                <div id="slipDirection" class="slip-direction">Centered</div>
                <div class="slip-track">
                    <div id="slipFillLeft" class="slip-fill slip-fill-left"></div>
                    <div id="slipFillRight" class="slip-fill slip-fill-right"></div>
                    <div id="slipNeedle" class="slip-needle"></div>
                </div>
                <div class="slip-scale">
                    <span>Left 30°</span>
                    <span>0°</span>
                    <span>Right 30°</span>
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
        <a href="/forza.json" target="_blank">View Raw JSON →</a>
    </div>
    
    <script>
        let socket = null;
        let reconnectTimer = null;
        let maxRpm = 8000;

        function setStatus(state, text) {
            const status = document.getElementById('status');
            status.className = 'status ' + state;
            status.textContent = text;
        }

        function numberOrNull(value) {
            const number = Number(value);
            return Number.isFinite(number) ? number : null;
        }

        function clamp(value, min, max) {
            return Math.min(Math.max(value, min), max);
        }

        function calculateSlipAngleDegrees(data) {
            const lateralVelocity = numberOrNull(data.VelocityX);
            const forwardVelocity = numberOrNull(data.VelocityZ);
            if (lateralVelocity === null || forwardVelocity === null) {
                return null;
            }

            const speed = Math.hypot(lateralVelocity, forwardVelocity);
            if (speed < 1) {
                return 0;
            }

            return Math.atan2(lateralVelocity, Math.abs(forwardVelocity)) * (180 / Math.PI);
        }

        function getOverallSlipState(data, slipAngle) {
            if (slipAngle === null) {
                return 'No Data';
            }

            const absAngle = Math.abs(slipAngle);
            if (absAngle < 4) {
                return 'Stable';
            }

            const direction = slipAngle < 0 ? 'Left' : 'Right';
            const combinedSlipValues = [
                data.TireCombinedSlipFrontLeft,
                data.TireCombinedSlipFrontRight,
                data.TireCombinedSlipRearLeft,
                data.TireCombinedSlipRearRight
            ]
                .map(numberOrNull)
                .filter(value => value !== null)
                .map(Math.abs);
            const maxCombinedSlip = combinedSlipValues.length ? Math.max(...combinedSlipValues) : 0;

            if (maxCombinedSlip >= 1.1) {
                return 'Sliding ' + direction;
            }

            return 'Cornering ' + direction;
        }
        
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
            const rpm = Math.max(numberOrNull(d.CurrentEngineRpm) || 0, 0);
            const engineMaxRpm = numberOrNull(d.EngineMaxRpm);
            if (engineMaxRpm && engineMaxRpm > 0 && engineMaxRpm < 20000) {
                maxRpm = Math.max(maxRpm, engineMaxRpm);
            }
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
            const speed = numberOrNull(d.Speed);
            const speedMph = speed === null ? null : Math.max(speed * 2.2369362920544, 0);
            document.getElementById('speedValue').textContent = speedMph !== null && speedMph <= 350 ? Math.round(speedMph) : '--';

            // Overall vehicle slip angle from local lateral vs forward velocity
            const slipAngle = calculateSlipAngleDegrees(d);
            const slipGaugeLimit = 30;
            const boundedSlip = slipAngle === null ? 0 : clamp(slipAngle, -slipGaugeLimit, slipGaugeLimit);
            const slipPercent = Math.abs(boundedSlip) / slipGaugeLimit * 50;
            document.getElementById('slipAngleValue').textContent = slipAngle === null ? '--' : Math.abs(slipAngle).toFixed(1) + '°';
            document.getElementById('slipDirection').textContent = getOverallSlipState(d, slipAngle);
            document.getElementById('slipNeedle').style.left = (50 + (boundedSlip / slipGaugeLimit * 50)) + '%';
            document.getElementById('slipFillLeft').style.width = boundedSlip < 0 ? slipPercent + '%' : '0%';
            document.getElementById('slipFillRight').style.width = boundedSlip > 0 ? slipPercent + '%' : '0%';
            
            // Pedals (0-255 range)
            const throttle = clamp(((numberOrNull(d.Accel) || 0) / 255) * 100, 0, 100);
            const brake = clamp(((numberOrNull(d.Brake) || 0) / 255) * 100, 0, 100);
            const clutch = clamp(((numberOrNull(d.Clutch) || 0) / 255) * 100, 0, 100);
            
            document.getElementById('throttleValue').textContent = Math.round(throttle) + '%';
            document.getElementById('throttleFill').style.width = throttle + '%';
            document.getElementById('brakeValue').textContent = Math.round(brake) + '%';
            document.getElementById('brakeFill').style.width = brake + '%';
            document.getElementById('clutchValue').textContent = Math.round(clutch) + '%';
            document.getElementById('clutchFill').style.width = clutch + '%';
            
            // Boost
            const boost = numberOrNull(d.Boost) || 0;
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
            
            // Tire temps from Forza Data Out
            const tires = [
                { id: 'FL', temp: d.TireTempFrontLeft },
                { id: 'FR', temp: d.TireTempFrontRight },
                { id: 'RL', temp: d.TireTempRearLeft },
                { id: 'RR', temp: d.TireTempRearRight }
            ];
            
            tires.forEach(tire => {
                const tempF = numberOrNull(tire.temp);
                const tireTemp = document.getElementById('tire' + tire.id + 'Temp');
                const tireBox = document.getElementById('tire' + tire.id);

                if (tempF === null || tempF < -40 || tempF > 500) {
                    tireTemp.textContent = '--';
                    tireBox.className = 'tire cold';
                    return;
                }

                tireTemp.textContent = Math.round(tempF);
                tireBox.className = 'tire ' + getTireClass(tempF);
            });
            
            // Session info
            const position = d.RacePosition || 0;
            document.getElementById('positionValue').textContent = position > 0 ? position : '--';
            document.getElementById('lapValue').textContent = (d.LapNumber || 0) + 1;
            document.getElementById('bestLapValue').textContent = formatLapTime(d.BestLap);
            document.getElementById('lastLapValue').textContent = formatLapTime(d.LastLap);
        }

        function handleTelemetryMessage(data) {
            if (!data || data.trim() === '' || data === '[]') {
                return;
            }

            updateDashboard(JSON.parse(data));
            setStatus('connected', 'LIVE');
        }

        function scheduleReconnect() {
            setStatus('disconnected', 'RECONNECTING');
            window.clearTimeout(reconnectTimer);
            reconnectTimer = window.setTimeout(connectTelemetry, 1000);
        }

        function connectTelemetry() {
            if (socket && (socket.readyState === WebSocket.CONNECTING || socket.readyState === WebSocket.OPEN)) {
                return;
            }

            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            socket = new WebSocket(protocol + '//' + window.location.host + '/forza/ws');

            socket.addEventListener('open', () => {
                setStatus('connected', 'LIVE');
            });

            socket.addEventListener('message', event => {
                try {
                    handleTelemetryMessage(event.data);
                } catch (error) {
                    setStatus('disconnected', 'BAD DATA');
                }
            });

            socket.addEventListener('close', scheduleReconnect);
            socket.addEventListener('error', () => {
                setStatus('disconnected', 'DISCONNECTED');
                socket.close();
            });
        }

        connectTelemetry();
    </script>
</body>
</html>`
}

// serveJSON starts the HTTP server on port 8080 to serve telemetry data.
// The /forza endpoint serves either a web dashboard or raw JSON based on the request.
func serveJSON() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/forza", http.StatusFound)
	})
	http.HandleFunc("/forza/ws", websocketResponder)
	http.HandleFunc("/forza.json", jsonResponder)
	http.HandleFunc("/forza", responder)

	log.Printf("JSON Telemetry Server started at http://localhost%s/forza", jsonServerPort)
	log.Fatal(http.ListenAndServe(jsonServerPort, nil))
}

// enableCors sets the Access-Control-Allow-Origin header to allow cross-origin requests.
func enableCors(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}
