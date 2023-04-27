# Kasher

Tunnel TCP through HTTP - no CONNECT


## How to use

Server:

`kasher-server.exe --cert [path-to-cert] --key [path-to-key] [port]`

Client:

`kasher-client.exe [port] [kasher-server-address]:[kasher-server-port] [destination-address]:[destination-port]`

Example

on *myserver*

`kasher-server.exe --cert ./cert.pem --key ./key.pem 10000`


`kasher-client.exe 20000 https://myserver:10000 sshserver:22`


