package compression

import (
	"github.com/klauspost/compress/zstd"
)

type Compressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
	enabled bool
}

func NewCompressor(level int, enabled bool) (*Compressor, error) {
	if !enabled {
		return &Compressor{enabled: false}, nil
	}

	var encoderLevel zstd.EncoderLevel
	switch level {
	case 1:
		encoderLevel = zstd.SpeedFastest
	case 2:
		encoderLevel = zstd.SpeedDefault
	case 3:
		encoderLevel = zstd.SpeedBetterCompression
	default:
		encoderLevel = zstd.SpeedDefault
	}

	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(encoderLevel),
		zstd.WithEncoderConcurrency(1),
	)
	if err != nil {
		return nil, err
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		return nil, err
	}

	return &Compressor{
		encoder: encoder,
		decoder: decoder,
		enabled: true,
	}, nil
}

func (c *Compressor) Compress(data []byte) ([]byte, error) {
	if !c.enabled || len(data) < 128 {
		return data, nil
	}

	compressed := c.encoder.EncodeAll(data, make([]byte, 0, len(data)))

	if len(compressed) >= len(data) {
		return data, nil
	}

	return compressed, nil
}

func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	if !c.enabled {
		return data, nil
	}

	decompressed, err := c.decoder.DecodeAll(data, nil)
	if err != nil {
		return data, nil
	}

	return decompressed, nil
}

func (c *Compressor) Close() error {
	if c.encoder != nil {
		c.encoder.Close()
	}
	if c.decoder != nil {
		c.decoder.Close()
	}
	return nil
}
