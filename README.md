# WsSession

## Description

Go websocket server for sending messages between clients in a session.

## Table of Contents

- [Usage](#usage)
- [Contributing](#contributing)

## Usage

1. Fork the source code to get your own copy and compile it.

2. Start the server by running the executable.

3. Send a ws handshake to start a session on the server.

4. After a sucessful handshake, the first message send on the connection is the code of your session.

5. Use this code to connect other clients to the same session and share messages.

## Contributing

If you would like to contribute or outline issues and potential improvements, feel free to raise an issue or create a pull request.