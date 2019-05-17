package events

const (
	NewBlock EventType = 0
	NewAddr EventType = 1
	NewMasternodeBroadcast EventType = 2
	NewMasternodePing EventType = 3
	NewPhantomPing EventType = 4
	PeerDisconnect EventType = 5
)

type EventType int

type Event struct {
	Type EventType
	Data interface{}
}
