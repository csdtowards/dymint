package zgda

import (
	"context"

	"google.golang.org/grpc"

	"github.com/0glabs/0g-da-client/api/grpc/disperser"
)

type ZgDADisperserClient interface {
	DisperseBlob(ctx context.Context, in *disperser.DisperseBlobRequest, opts ...grpc.CallOption) (*disperser.DisperseBlobReply, error)
	GetBlobStatus(ctx context.Context, in *disperser.BlobStatusRequest, opts ...grpc.CallOption) (*disperser.BlobStatusReply, error)
	RetrieveBlob(ctx context.Context, in *disperser.RetrieveBlobRequest, opts ...grpc.CallOption) (*disperser.RetrieveBlobReply, error)
}

type ZgDADisperserGRPCClient struct {
	daDisperserClient disperser.DisperserClient
}

func NewZgDADisperserGRPCClient(grpcClient disperser.DisperserClient) *ZgDADisperserGRPCClient {
	return &ZgDADisperserGRPCClient{
		daDisperserClient: grpcClient,
	}
}

func (c *ZgDADisperserGRPCClient) DisperseBlob(ctx context.Context, in *disperser.DisperseBlobRequest, opts ...grpc.CallOption) (*disperser.DisperseBlobReply, error) {
	return c.daDisperserClient.DisperseBlob(ctx, in, opts...)
}

func (c *ZgDADisperserGRPCClient) GetBlobStatus(ctx context.Context, in *disperser.BlobStatusRequest, opts ...grpc.CallOption) (*disperser.BlobStatusReply, error) {
	return c.daDisperserClient.GetBlobStatus(ctx, in, opts...)
}

func (c *ZgDADisperserGRPCClient) RetrieveBlob(ctx context.Context, in *disperser.RetrieveBlobRequest, opts ...grpc.CallOption) (*disperser.RetrieveBlobReply, error) {
	return c.daDisperserClient.RetrieveBlob(ctx, in, opts...)
}
