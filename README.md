# Forza data tools
Forza data out tools

# Features
- Data logging to csv file


# Usage
From your game HUD options, enable the data out feature and set it to use the IP address of your computer. Port 9999.  
Forza Motorsport 7 select the "car dash" format.

## Options
Port flag
csv FLAG -c
horizon flag -z

## Example (Forza Horizon)
`go run main.go -z -c log.csv`  

## Example (Forza Motorsport)
`go run main.go -c log.csv`  

# Further reading
- Forza data out format: https://forums.forzamotorsport.net/turn10_postsm926839_Forza-Motorsport-7--Data-Out--feature-details.aspx#post_926839

- Forza Horizon 4 has some mystery data in the packet, waiting on info from the developers - https://forums.forzamotorsport.net/turn10_postsm1086012_Data-Output.aspx#post_1086012