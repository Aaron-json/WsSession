# WsSession

## Description

Go websocket server for sending messages between clients in a session.

## Table of Contents

- [Usage](#usage)
- [Contributing](#contributing)

## Usage

1. Fork the source code to get your own copy and compile it.

2. Start the server by running the executable.

3. Send a ws handshake to start a session on the server. When creating or joining a session, the first message sent on the connection signals the connection is ready to receive messages; however, the session code is only sent when creating the session.

4. Use this code to connect other clients to the same session and share messages.

## Contributing

If you would like to contribute or outline issues and potential improvements, feel free to raise an issue or create a pull request.