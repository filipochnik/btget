package torrent

import (
	"crypto/sha1"
	"errors"
	"io/ioutil"

	"github.com/filipochnik/btget/bencode"
)

type MetaInfo struct {
	Info         InfoDict `bencode:"info"`
	InfoHash     []byte
	Announce     string     `bencode:"announce"`
	AnnounceList [][]string `bencode:"announce-list"`
	CreationDate int        `bencode:"creation date"`
	Comment      string     `bencode:"comment"`
	CreatedBy    string     `bencode:"created by"`
	Encoding     string     `bencode:"encoding"`
}

type InfoDict struct {
	PieceLength int    `bencode:"piece length"`
	Pieces      []byte `bencode:"pieces"`
	Name        string `bencode:"name"`

	// Single File Mode
	Length int `bencode:"length"`

	// Multiple Files Mode
	Files []FileDict `bencode:"files"`
}

type FileDict struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

func NewMetaInfo(filePath string) *MetaInfo {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(err)
	}
	var m MetaInfo
	bencode.Unmarshal(data, &m)
	m.InfoHash = infoHash(data)
	return &m
}

func infoHash(data []byte) []byte {
	b, err := infoBencode(data)
	if err != nil {
		// TODO: handle this
		panic(err)
	}
	hash := sha1.New()
	hash.Write(b)
	return hash.Sum(nil)
}

type infoExtractor struct {
	// TODO add raw type to bencode lib
	Info interface{} `bencode:"info"`
}

func infoBencode(data []byte) ([]byte, error) {
	var infoExt infoExtractor
	bencode.Unmarshal(data, &infoExt)
	m, ok := infoExt.Info.(map[string]interface{})
	if !ok {
		return nil, errors.New("info is not a dict")
	}
	return bencode.Marshal(m)
}
