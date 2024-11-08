package zgda

import (
	"context"

	"github.com/0glabs/0g-da-client/api/grpc/disperser"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

type MockZgDADisperserClient struct {
	mock.Mock
}

func NewMockZgDADisperserClient(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockZgDADisperserClient {
	mock := &MockZgDADisperserClient{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

func (_m *MockZgDADisperserClient) DisperseBlob(ctx context.Context, in *disperser.DisperseBlobRequest, opts ...grpc.CallOption) (*disperser.DisperseBlobReply, error) {
	ret := _m.Called(ctx, in, opts)

	if len(ret) == 0 {
		panic("no return value specified for DisperseBlob")
	}

	var r0 *disperser.DisperseBlobReply
	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *disperser.DisperseBlobRequest, ...grpc.CallOption) (*disperser.DisperseBlobReply, error)); ok {
		return rf(ctx, in, opts...)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *disperser.DisperseBlobRequest, ...grpc.CallOption) *disperser.DisperseBlobReply); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*disperser.DisperseBlobReply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *disperser.DisperseBlobRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockZgDADisperserClient) GetBlobStatus(ctx context.Context, in *disperser.BlobStatusRequest, opts ...grpc.CallOption) (*disperser.BlobStatusReply, error) {
	ret := _m.Called(ctx, in, opts)

	if len(ret) == 0 {
		panic("no return value specified for GetBlobStatus")
	}

	var r0 *disperser.BlobStatusReply
	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *disperser.BlobStatusRequest, ...grpc.CallOption) (*disperser.BlobStatusReply, error)); ok {
		return rf(ctx, in, opts...)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *disperser.BlobStatusRequest, ...grpc.CallOption) *disperser.BlobStatusReply); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*disperser.BlobStatusReply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *disperser.BlobStatusRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (_m *MockZgDADisperserClient) RetrieveBlob(ctx context.Context, in *disperser.RetrieveBlobRequest, opts ...grpc.CallOption) (*disperser.RetrieveBlobReply, error) {
	ret := _m.Called(ctx, in, opts)

	if len(ret) == 0 {
		panic("no return value specified for RetrieveBlob")
	}

	var r0 *disperser.RetrieveBlobReply
	var r1 error

	if rf, ok := ret.Get(0).(func(context.Context, *disperser.RetrieveBlobRequest, ...grpc.CallOption) (*disperser.RetrieveBlobReply, error)); ok {
		return rf(ctx, in, opts...)
	}

	if rf, ok := ret.Get(0).(func(context.Context, *disperser.RetrieveBlobRequest, ...grpc.CallOption) *disperser.RetrieveBlobReply); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*disperser.RetrieveBlobReply)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *disperser.RetrieveBlobRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
