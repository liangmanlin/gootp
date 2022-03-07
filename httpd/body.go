package httpd

import (
	"github.com/liangmanlin/gootp/bpool"
	"io"
	"sync"
)

var (
	bodyReaderPool = sync.Pool{
		New: func() interface{} {
			return &BodyReader{}
		},
	}
)

// BodyReader .
type BodyReader struct {
	index  int
	buffer *bpool.Buff
}

// Read implements io.Reader.
func (br *BodyReader) Read(p []byte) (int, error) {
	need := len(p)
	buf := br.buffer.ToBytes()
	available := len(buf) - br.index
	if available <= 0 {
		return 0, io.EOF
	}
	if available >= need {
		copy(p, buf[br.index:br.index+need])
		br.index += need
		// if available == need {
		// 	br.Close()
		// }
		return need, nil
	}
	copy(p[:available], buf[br.index:])
	br.index += available
	return available, io.EOF
}

// Append .
func (br *BodyReader) Append(data []byte) {
	if len(data) > 0 {
		if br.buffer == nil {
			br.buffer = bpool.NewBuf(data)
		} else {
			br.buffer = br.buffer.Append(data...)
		}
	}
}

func (br *BodyReader) RawBody() []byte {
	return br.buffer.ToBytes()
}

func (br *BodyReader) TakeOver() *bpool.Buff {
	b := br.buffer
	br.buffer = nil
	br.index = 0
	return b
}

// Close implements io. Closer.
func (br *BodyReader) Close() error {
	br.close()
	return nil
}

func (br *BodyReader) close() {
	if br.buffer != nil {
		br.buffer.Free()
		br.buffer = nil
		br.index = 0
	}
}

// NewBodyReader creates a BodyReader.
func NewBodyReader(buffer *bpool.Buff) *BodyReader {
	br := bodyReaderPool.Get().(*BodyReader)
	br.buffer = buffer
	br.index = 0
	return br
}
