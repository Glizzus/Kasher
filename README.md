# Kasher

Tunnel TCP through HTTPS - no CONNECT

A project to generate an HTTP tunneling without the CONNECT
method. Created with portability, speed, and ease of use in mind,
the binaries are relatively small (~5000 kb) and require no intepreter.

The server may require admin privileges to set up, but the client is designed
to be run anywhere.

Previously, the client set off Windows Defender. Although this has been solved
for now, don't be suprised if it happens again.

The server generates its own TLS certificates at runtime so that none need
to be provided.

# Installation

1. Grab the source and build yourself

```
go build client
go build server
```

If you have python installed, you can use the provided [script](/tools/compile.py)
to compile a bunch of binaries and take what you need.

2. Take a precompiled binary

Precompiled binaries are on the [releases](/releases) page. Download what you need.

# Usage

Client - `kasherclient.exe [local-port] [kasher-server-url] [destination]`

Server - `kasherserver.exe [local-port]`


Example:

Client - `kasherclient.exe 50000 https://kasherserver:30000 remotesshserver:22`

Server - `kasherserver.exe 30000`

# API

## Create new Connection

**URL**: `/{connection-id}`

**Method**: `POST`

**Data constraints**

Provide the destination in raw bytes
```
[destination-host]:[destination-port]
```

**Data example** Tunneling SSH
```
sshserver:22
```

### Success Response

**Condition**: The server successfully created the connection

**Code**: `201 CREATED`

<br />

## Upload Data

**URL**: `/{connection-id}`

**Method**: `PUT`

**DATA constraints**

Provide the bytes to upload
```
[bytes]
```

### Success Response

**Condition**: The server is successfully reading the data

**Code**: `200 OK`

### Error Response

**Condition**: The requested connection id doesn't exist

**Code**: `404 NOT FOUND`

<br />

## Retrieve Data

**URL**: `/{connection-id}`

**Method**: `GET`

### Success Response

**Condition**: The server has data to return for the connection id

**CODE**: `200 OK`

**Content**: The data in bytes
```
[bytes]
```

OR

**Condition**: The server has no data, but everything is fine

**CODE**: `204 NO CONTENT`

**CONTENT**: Empty Response

### Error Response

**Condition**: The requested connection id doesn't exist

**Code**: `404 NOT FOUND`

OR

**Condition**: The connection id exists but the connection has been closed by EOF

**CODE**: `410 GONE`

<br />

## Remove Connection

**URL**: `/{connection-id}`

**Method**: `DELETE`

### Success Response

**Condition**: The requested id has either been deleted or never existed

**Code**: `200 OK`