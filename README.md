# Kasher

Tunnel TCP through HTTP - no CONNECT


## How to use

Server:

`kasher-server.exe [port] --cert [path-to-cert] --key [path-to-key]`

Client:

`kasher-client.exe [port] [kasher-server-address]:[kasher-server-port] [destination-address]:[destination-port]`

Example

on *myserver*

`kasher-server.exe 10000 --cert ./cert.pem --key ./key.pem`


`kasher-client.exe 20000 https://myserver:10000 sshserver:22`


