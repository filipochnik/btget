package torrent

type AnnounceRequest struct {
	InfoHash   [20]byte
	PeerID     [20]byte
	Port       int
	Uploaded   int
	Downloaded int
	Left       int
	Event      AnnounceEvent
	NumWant    int
	TrackerID  string
}

type AnnounceEvent string

const (
	EventStarted   AnnounceEvent = "started"
	EventStopped   AnnounceEvent = "stopped"
	EventCompleted AnnounceEvent = "completed"
	EventEmpty     AnnounceEvent = "empty"
)

type AnnounceResponse struct {
	FailureReason  string      `bencode:"failure reason"`
	WarningMessage string      `bencode:"warning message"`
	Interval       int         `bencode:"interval"`
	TrackerID      string      `bencode:"tracker id"`
	Complete       int         `bencode:"complete"`
	Incomplete     int         `bencode:"complete"`
	Peers          interface{} `bencode:"peers"`
}
