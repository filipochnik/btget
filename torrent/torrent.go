package torrent

type Torrent struct {
	metaInfo MetaInfo

	Length int
}

func NewTorrent(mi MetaInfo) *Torrent {
	var length int
	if mi.Info.Files == nil {
		// single file torrent
		length = mi.Info.Length
	} else {
		for _, f := range mi.Info.Files {
			length += f.Length
		}
	}
	return &Torrent{
		metaInfo: mi,
		Length:   length,
	}
}
