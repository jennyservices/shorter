package v1

import (
	"context"

	pb "github.com/jennyservices/shorter/transport/pb"

	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/jennyservices/jenny/options"
)

type shorterGRPCServer struct {
	shorter grpctransport.Handler
}

func NewShorterGRPCServer(svc Shorter, opts ...options.Option) *shorterGRPCServer {
	svcOptions := options.New()
	for _, optf := range opts {
		optf(svcOptions)
	}
	shortenEndpoint := makeShortenEndpoint(svc, svcOptions)
	return &shorterGRPCServer{
		shorter: grpctransport.NewServer(
			shortenEndpoint,
			decodeShortenGRPCRequest,
			encodeShortenGRPCResponse,
		),
	}
}

func decodeShortenGRPCRequest(_ context.Context, r interface{}) (interface{}, error) {
	req := r.(*pb.URL)
	return _shortenRequest{
		Long: URL{
			Addr: req.Addr,
		},
	}, nil
}

func encodeShortenGRPCResponse(_ context.Context, r interface{}) (interface{}, error) {
	resp := r.(_shortenResponse)
	return &pb.URL{
		Addr: resp.Body.Addr,
	}, nil
}
func (s *shorterGRPCServer) Shorten(ctx context.Context, r *pb.URL) (*pb.URL, error) {
	_, resp, err := s.shorter.ServeGRPC(ctx, r)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.URL), nil
}
