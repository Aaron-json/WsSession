package controllers

// packet for sending new messages to the user
type Packet struct {
	SessionID string      `json:"sessionID"`
	Message   Message     `json:"message"`
	Type      MessageType `json:"type"`
}
