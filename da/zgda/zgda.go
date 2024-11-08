package zgda

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	zgdapb "github.com/0glabs/0g-da-client/api/grpc/disperser"
	"github.com/gogo/protobuf/proto"

	"github.com/dymensionxyz/dymint/da"
	"github.com/dymensionxyz/dymint/store"
	"github.com/dymensionxyz/dymint/types"
	pb "github.com/dymensionxyz/dymint/types/pb/dymint"
	"github.com/tendermint/tendermint/libs/pubsub"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	DefaultRequestTimeout      = 180 * time.Second
	DefaultStatusRetryDelay    = 3 * time.Second
	DefaultStatusRetryAttempts = 90
	DefaultMaxBytes            = 1024*0124*31 - 4
)

type Config struct {
	Server string `json:"server"`
}

type DataAvailabilityLayerClient struct {
	daDisperserClient ZgDADisperserClient

	pubsubServer *pubsub.Server
	config       Config
	logger       types.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	synced       chan struct{}
}

var (
	_ da.DataAvailabilityLayerClient = &DataAvailabilityLayerClient{}
	_ da.BatchRetriever              = &DataAvailabilityLayerClient{}
)

// WithClient is an option which sets the client.
func WithGRPCClient(client ZgDADisperserClient) da.Option {
	return func(dalc da.DataAvailabilityLayerClient) {
		dalc.(*DataAvailabilityLayerClient).daDisperserClient = client
	}
}

// Init initializes DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Init(config []byte, pubsubServer *pubsub.Server, kvStore store.KV, logger types.Logger, options ...da.Option) error {
	c.logger = logger
	c.synced = make(chan struct{}, 1)

	if len(config) > 0 {
		err := json.Unmarshal(config, &c.config)
		if err != nil {
			return err
		}
	} else {
		return errors.New("supplied config is empty")
	}

	// Set defaults
	c.pubsubServer = pubsubServer

	// Apply options
	for _, apply := range options {
		apply(c)
	}

	// If client wasn't set, create a new one
	if c.daDisperserClient == nil {
		conn, err := grpc.NewClient(c.config.Server, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*1024))) // 1 GiB
		if err != nil {
			return fmt.Errorf("failed to dial 0g da client: %w", err)
		}
		c.daDisperserClient = NewZgDADisperserGRPCClient(zgdapb.NewDisperserClient(conn))
	}

	types.RollappConsecutiveFailedDASubmission.Set(0)

	c.ctx, c.cancel = context.WithCancel(context.Background())
	return nil
}

// Start starts DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Start() error {
	c.synced <- struct{}{}
	return nil
}

// Stop stops DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) Stop() error {
	c.cancel()
	close(c.synced)
	return nil
}

// WaitForSyncing is used to check when the DA light client finished syncing
func (m *DataAvailabilityLayerClient) WaitForSyncing() {
	<-m.synced
}

// GetClientType returns client type.
func (c *DataAvailabilityLayerClient) GetClientType() da.Client {
	return da.Zgda
}

// SubmitBatch submits batch to DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) SubmitBatch(batch *types.Batch) da.ResultSubmitBatch {
	blob, err := batch.MarshalBinary()
	if err != nil {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   err,
			},
		}
	}

	if len(blob) > DefaultMaxBytes {
		return da.ResultSubmitBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: fmt.Sprintf("size bigger than maximum blob size: max n bytes: %d", DefaultMaxBytes),
				Error:   errors.New("blob size too big"),
			},
		}
	}

	c.logger.Debug("Submitting to da batch with size", "size", len(blob))
	return c.submitBatchLoop(blob)
}

// submitBatchLoop tries submitting the batch. In case we get a configuration error we would like to stop trying,
// otherwise, for network error we keep trying indefinitely.
func (c *DataAvailabilityLayerClient) submitBatchLoop(dataBlob []byte) da.ResultSubmitBatch {
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Debug("Context cancelled.")
			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusError,
					Message: "context done",
					Error:   c.ctx.Err(),
				},
			}
		default:
			ctxWithTimeout, cancel := context.WithTimeout(c.ctx, DefaultRequestTimeout)
			defer cancel()

			blobReply, err := c.daDisperserClient.DisperseBlob(ctxWithTimeout, &zgdapb.DisperseBlobRequest{
				Data: dataBlob,
			})

			if err != nil {
				c.logger.Error("Disperse blob error", "err", err)
				// types.RollappConsecutiveFailedDASubmission.Inc()

				err = fmt.Errorf("submit: %w", err)
				return da.ResultSubmitBatch{
					BaseResult: da.BaseResult{
						Code:    da.StatusError,
						Message: err.Error(),
						Error:   err,
					},
				}
			}

			requestId := blobReply.GetRequestId()
			daMetaData := &da.DASubmitMetaData{
				Client: da.Zgda,
			}

			var retryCount uint64
			for {
				ctxWithTimeout, cancel := context.WithTimeout(c.ctx, DefaultRequestTimeout)
				defer cancel()

				statusReply, err := c.daDisperserClient.GetBlobStatus(ctxWithTimeout, &zgdapb.BlobStatusRequest{RequestId: requestId})
				if err != nil {
					c.logger.Error("Get blob status error", "err", err)
					types.RollappConsecutiveFailedDASubmission.Inc()

					return da.ResultSubmitBatch{
						BaseResult: da.BaseResult{
							Code:    da.StatusError,
							Message: err.Error(),
							Error:   err,
						},
					}
				}

				c.logger.Info("Blob status reply", "status", statusReply.GetStatus())
				if statusReply.GetStatus() == zgdapb.BlobStatus_CONFIRMED || statusReply.GetStatus() == zgdapb.BlobStatus_FINALIZED {
					blobInfo := statusReply.GetInfo()
					dataRoot := blobInfo.BlobHeader.GetStorageRoot()
					epoch := blobInfo.BlobHeader.GetEpoch()
					quorumId := blobInfo.BlobHeader.GetQuorumId()

					daMetaData.Commitment = dataRoot
					daMetaData.Height = epoch
					daMetaData.Index = int(quorumId)

					break
				}

				if statusReply.GetStatus() == zgdapb.BlobStatus_FAILED {
					c.logger.Error("Store blob error")
					types.RollappConsecutiveFailedDASubmission.Inc()

					return da.ResultSubmitBatch{
						BaseResult: da.BaseResult{
							Code:    da.StatusError,
							Message: "store blob error",
							Error:   errors.New("store blob error"),
						},
					}
				}

				retryCount++
				if retryCount > DefaultStatusRetryAttempts {
					return da.ResultSubmitBatch{
						BaseResult: da.BaseResult{
							Code:    da.StatusError,
							Message: "retry attempts reached",
							Error:   errors.New("retry attempts reached"),
						},
					}
				}

				time.Sleep(DefaultStatusRetryDelay)
			}

			c.logger.Debug("Submitted blob to DA successfully.")
			types.RollappConsecutiveFailedDASubmission.Set(0)

			return da.ResultSubmitBatch{
				BaseResult: da.BaseResult{
					Code:    da.StatusSuccess,
					Message: "Submission successful",
				},
				SubmitMetaData: daMetaData,
			}
		}
	}
}

// RetrieveBatches retrieves batch from DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) RetrieveBatches(daMetaData *da.DASubmitMetaData) da.ResultRetrieveBatch {
	ctxWithTimeout, cancel := context.WithTimeout(c.ctx, DefaultRequestTimeout)
	defer cancel()

	c.logger.Debug("Getting blob from DA.", "epoch", daMetaData.Height, "quorumId", daMetaData.Index, "commitment", hex.EncodeToString(daMetaData.Commitment))

	blobReply, err := c.daDisperserClient.RetrieveBlob(ctxWithTimeout, &zgdapb.RetrieveBlobRequest{
		StorageRoot: daMetaData.Commitment,
		Epoch:       daMetaData.Height,
		QuorumId:    uint64(daMetaData.Index),
	})

	if err != nil {
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrRetrieval,
			},
		}
	}

	var batches []*types.Batch

	var batch pb.Batch
	err = proto.Unmarshal(blobReply.GetData(), &batch)
	if err != nil {
		c.logger.Error("Unmarshal blob.", "epoch", daMetaData.Height, "error", err)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	c.logger.Debug("Blob retrieved successfully from DA.", "epoch", daMetaData.Height)

	parsedBatch := new(types.Batch)
	// Convert the proto batch to a batch
	err = parsedBatch.FromProto(&batch)
	if err != nil {
		c.logger.Error("batch from proto", "epoch", daMetaData.Height, "error", err)
		return da.ResultRetrieveBatch{
			BaseResult: da.BaseResult{
				Code:    da.StatusError,
				Message: err.Error(),
				Error:   da.ErrBlobNotParsed,
			},
		}
	}

	batches = append(batches, parsedBatch)
	return da.ResultRetrieveBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "Batch retrieval successful",
		},
		Batches: batches,
	}
}

// CheckBatchAvailability checks batch availability in DataAvailabilityLayerClient instance.
func (c *DataAvailabilityLayerClient) CheckBatchAvailability(daMetaData *da.DASubmitMetaData) da.ResultCheckBatch {
	return da.ResultCheckBatch{
		BaseResult: da.BaseResult{
			Code:    da.StatusSuccess,
			Message: "not implemented",
		},
	}
}

// GetMaxBlobSizeBytes returns the maximum allowed blob size in the DA, used to check the max batch size configured
func (d *DataAvailabilityLayerClient) GetMaxBlobSizeBytes() uint32 {
	return DefaultMaxBytes
}
