package zgda_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"

	"github.com/0glabs/0g-da-client/api/grpc/disperser"
	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/da/zgda"
	"github.com/dymensionxyz/dymint/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/pubsub"

	mocks "github.com/dymensionxyz/dymint/mocks/github.com/dymensionxyz/dymint/da/zgda"
)

func TestRetrieveBatches(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Set up the config
	configBytes, err := json.Marshal(zgda.Config{
		Server: "127.0.0.1:51001",
	})
	require.NoError(err)

	// Create mock client
	mockSubstrateApiClient := mocks.NewMockZgDADisperserClient(t)

	// Configure DALC options
	options := []da.Option{
		zgda.WithGRPCClient(mockSubstrateApiClient),
	}
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	assert.NoError(err)

	// Start the DALC
	dalc := zgda.DataAvailabilityLayerClient{}
	err = dalc.Init(configBytes, pubsubServer, nil, testutil.NewLogger(t), options...)
	require.NoError(err)
	err = dalc.Start()
	require.NoError(err)

	// Set the mock functions
	batch1 := testutil.MustGenerateBatchAndKey(0, 1)

	batch1bytes, err := batch1.MarshalBinary()
	require.NoError(err)

	mockSubstrateApiClient.On("RetrieveBlob", mock.Anything, mock.Anything, mock.Anything).Return(&disperser.RetrieveBlobReply{Data: batch1bytes}, nil)

	commitmentString := "3f568f651fe72fa2131bd86c09bb23763e0a3cb45211b035bfa688711c76ce78"
	commitment, _ := hex.DecodeString(commitmentString)
	daMetaData := &da.DASubmitMetaData{
		Height:     1,
		Index:      1,
		Commitment: commitment,
	}
	batchResult := dalc.RetrieveBatches(daMetaData)
	assert.Equal(1, len(batchResult.Batches))
	assert.Equal(batch1.StartHeight(), batchResult.Batches[0].StartHeight())
}

func TestSubmitBatches(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Set up the config
	configBytes, err := json.Marshal(zgda.Config{
		Server: "127.0.0.1:51001",
	})
	require.NoError(err)

	// Create mock client
	mockSubstrateApiClient := mocks.NewMockZgDADisperserClient(t)

	// Configure DALC options
	options := []da.Option{
		zgda.WithGRPCClient(mockSubstrateApiClient),
	}
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	assert.NoError(err)

	// Start the DALC
	dalc := zgda.DataAvailabilityLayerClient{}
	err = dalc.Init(configBytes, pubsubServer, nil, testutil.NewLogger(t), options...)
	require.NoError(err)
	err = dalc.Start()
	require.NoError(err)

	// Set the mock functions
	mockSubstrateApiClient.On("DisperseBlob", mock.Anything, mock.Anything, mock.Anything).Return(
		&disperser.DisperseBlobReply{
			Result:    disperser.BlobStatus_PROCESSING,
			RequestId: []byte("123")},
		nil)

	commitmentString := "3f568f651fe72fa2131bd86c09bb23763e0a3cb45211b035bfa688711c76ce78"
	commitment, _ := hex.DecodeString(commitmentString)
	mockSubstrateApiClient.On("GetBlobStatus", mock.Anything, mock.Anything, mock.Anything).Return(
		&disperser.BlobStatusReply{
			Status: disperser.BlobStatus_CONFIRMED,
			Info: &disperser.BlobInfo{
				BlobHeader: &disperser.BlobHeader{
					StorageRoot: commitment,
					Epoch:       1,
					QuorumId:    1,
				},
			},
		},
		nil)

	batch1 := testutil.MustGenerateBatchAndKey(0, 1)
	batchResult := dalc.SubmitBatch(batch1)

	assert.Equal(da.StatusSuccess, batchResult.BaseResult.Code)
	assert.Equal(uint64(1), batchResult.SubmitMetaData.Height)
	assert.Equal(1, batchResult.SubmitMetaData.Index)
	assert.Equal(commitment, batchResult.SubmitMetaData.Commitment)
}

func TestDisperseBatchesFail(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Set up the config
	configBytes, err := json.Marshal(zgda.Config{
		Server: "127.0.0.1:51001",
	})
	require.NoError(err)

	// Create mock client
	mockSubstrateApiClient := mocks.NewMockZgDADisperserClient(t)

	// Configure DALC options
	options := []da.Option{
		zgda.WithGRPCClient(mockSubstrateApiClient),
	}
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	assert.NoError(err)

	// Start the DALC
	dalc := zgda.DataAvailabilityLayerClient{}
	err = dalc.Init(configBytes, pubsubServer, nil, testutil.NewLogger(t), options...)
	require.NoError(err)
	err = dalc.Start()
	require.NoError(err)

	// Set the mock functions
	mockSubstrateApiClient.On("DisperseBlob", mock.Anything, mock.Anything, mock.Anything).Return(
		nil,
		errors.New("disperse timeout error"))

	batch1 := testutil.MustGenerateBatchAndKey(0, 1)
	batchResult := dalc.SubmitBatch(batch1)

	assert.Equal(da.StatusError, batchResult.BaseResult.Code)
}

func TestBatchesStatusFail(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	// Set up the config
	configBytes, err := json.Marshal(zgda.Config{
		Server: "127.0.0.1:51001",
	})
	require.NoError(err)

	// Create mock client
	mockSubstrateApiClient := mocks.NewMockZgDADisperserClient(t)

	// Configure DALC options
	options := []da.Option{
		zgda.WithGRPCClient(mockSubstrateApiClient),
	}
	pubsubServer := pubsub.NewServer()
	err = pubsubServer.Start()
	assert.NoError(err)

	// Start the DALC
	dalc := zgda.DataAvailabilityLayerClient{}
	err = dalc.Init(configBytes, pubsubServer, nil, testutil.NewLogger(t), options...)
	require.NoError(err)
	err = dalc.Start()
	require.NoError(err)

	// Set the mock functions
	mockSubstrateApiClient.On("DisperseBlob", mock.Anything, mock.Anything, mock.Anything).Return(
		&disperser.DisperseBlobReply{
			Result:    disperser.BlobStatus_PROCESSING,
			RequestId: []byte("123")},
		nil)

	mockSubstrateApiClient.On("GetBlobStatus", mock.Anything, mock.Anything, mock.Anything).Return(
		&disperser.BlobStatusReply{
			Status: disperser.BlobStatus_FAILED,
			Info:   nil,
		},
		nil)

	batch1 := testutil.MustGenerateBatchAndKey(0, 1)
	batchResult := dalc.SubmitBatch(batch1)

	assert.Equal(da.StatusError, batchResult.BaseResult.Code)
}
