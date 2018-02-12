package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/filipochnik/btget/bencode"
	"github.com/filipochnik/btget/torrent"
)

const version = "0001"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	if len(os.Args) != 1 {
		fmt.Println("usage: ./btget FILE")
	}

	filePath := os.Args[1]

	metaInfo := torrent.NewMetaInfo(filePath)
	//prettyPrint(metaInfo)

	torrent := torrent.NewTorrent(*metaInfo)

	announceURL, err := url.Parse(metaInfo.Announce)
	if err != nil {
		panic(err)
	}

	peerID := generatePeerID()

	q := url.Values{}
	q.Add("info_hash", string(metaInfo.InfoHash))
	q.Add("peer_id", string(peerID))
	q.Add("port", "6889")
	q.Add("uploaded", "0")
	q.Add("downloaded", "0")
	q.Add("left", strconv.Itoa(torrent.Length))
	q.Add("compact", "1")
	q.Add("no_peer_id", "0")
	q.Add("event", "started")
	q.Add("numwant", "30")
	announceURL.RawQuery = q.Encode()
	fmt.Println(announceURL.String())

	resp, err := http.Get(announceURL.String())
	if err != nil {
		panic(err)
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var announce map[string]interface{}
	bencode.Unmarshal(b, &announce)
	prettyPrint(announce)

	var peers []Peer

	switch rawPeers := announce["peers"].(type) {
	case []byte:
		for i := 0; i < len(rawPeers); i += 6 {
			peer := PeerFromBytes(rawPeers[i : i+6])
			peers = append(peers, peer)
		}
	default:
		err := fmt.Errorf("invalid peers type: %T", announce["peers"])
		panic(err)
	}

	for _, p := range peers {
		prettyPrint(p)
	}

	var conn net.Conn
	for i, peer := range peers {
		conn, err = net.DialTimeout("tcp", peer.Addr(), 5*time.Second)
		if err == nil {
			break
		} else {
			fmt.Printf("[ERR] connecting to %s failed: %v\n", peer.Addr(), err)
			if i == len(peers)-1 {
				panic("no conns")
			}
		}
	}

	handshake := []byte("\x13BitTorrent protocol\x00\x00\x00\x00\x00\x00\x00\x00")
	handshake = append(handshake, metaInfo.InfoHash...)
	handshake = append(handshake, peerID...)

	fmt.Println(len(handshake), ":", string(handshake))
	_, err = conn.Write(handshake)
	if err != nil {
		panic(err)
	}
	handshake2 := make([]byte, len(handshake))
	_, err = io.ReadFull(conn, handshake2)
	if err != nil {
		panic(err)
	}
	fmt.Println("got", len(handshake2))
	fmt.Println(string(handshake2))
}

func prettyPrint(o interface{}) (int, error) {
	b, err := json.MarshalIndent(o, "", "  ")
	if err != nil {
		return 0, err
	}
	return fmt.Println(string(b))
}

type Peer struct {
	IP   string
	Port uint
}

func PeerFromBytes(bytes []byte) Peer {
	return Peer{
		IP:   net.IP(bytes[:4]).String(),
		Port: uint(bytes[4])<<8 + uint(bytes[5]),
	}
}

func (p *Peer) Addr() string {
	return fmt.Sprintf("%s:%d", p.IP, p.Port)
}

type PeerConnection struct {
	Peer Peer

	conn net.Conn

	AmChoking      bool
	AmInterested   bool
	PeerChoking    bool
	PeerInterested bool
}

func NewPeerConnection(peer Peer, conn net.Conn) PeerConnection {
	return PeerConnection{
		Peer:           peer,
		conn:           conn,
		AmChoking:      true,
		AmInterested:   false,
		PeerChoking:    true,
		PeerInterested: false,
	}
}

func generatePeerID() []byte {
	prefix := []byte(fmt.Sprintf("-GT%s-", version))
	suffix := make([]byte, 20-len(prefix))
	for i := range suffix {
		// skips whitespace and non-printable characters
		suffix[i] = byte(rune(rand.Intn(127-33) + 33))
	}
	return append(prefix, suffix...)
}
