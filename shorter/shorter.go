package shorter

import (
	"context"
	"hash/crc32"

	v1 "github.com/jennyservices/shorter/transport/v1"
	"willnorris.com/go/newbase60"
)

func New() *shorter {
	return &shorter{}
}

type shorter struct{}

func (s *shorter) Shorten(ctx context.Context, u v1.URL) (*v1.URL, error) {
	i := crc32.ChecksumIEEE([]byte(u.Addr))
	return &v1.URL{Addr: newbase60.EncodeInt(int(i))}, nil
}