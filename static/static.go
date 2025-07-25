package static

import _ "embed"

var (
	//go:embed default.png
	DefaultCoverBytes   []byte
	//go:embed embed-cover.png
	TestEmbedCoverBytes []byte
	//go:embed no-embeded-album-cover-demo.mp3
	NoCoverMP3Bytes     []byte
	//go:embed embeded-album-cover-demo.mp3
	EmbedCoverMP3Bytes  [] byte
)
