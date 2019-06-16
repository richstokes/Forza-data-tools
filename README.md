# Forza data tools
Building some tools for playing with the data out feature from the Forza Motorsport / Forza Horizon games.


## Features
- Telemetry data logging to csv file

(Feel free to open an issue if you have any suggestions/requests)
&nbsp;

## Usage
From your game HUD options, enable the data out feature and set it to use the IP address of your computer. Port 9999.  
Forza Motorsport 7 select the "car dash" format.

### Command Options
Specify a CSV file to log to: `-c log.csv`  
Enable supprot for Forza Horizon: `-z`  

#### Example (Forza Horizon)
`go run main.go -z -c log.csv`  

#### Example (Forza Motorsport)
`go run main.go -c log.csv`  

&nbsp; 
## Further reading
- Forza data out format: https://forums.forzamotorsport.net/turn10_postsm926839_Forza-Motorsport-7--Data-Out--feature-details.aspx#post_926839

- Forza Horizon 4 has some mystery data in the packet, waiting on info from the developers - https://forums.forzamotorsport.net/turn10_postsm1086012_Data-Output.aspx#post_1086012