package client

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/assert"
	"github.com/tendermint/tendermint/crypto/tmhash"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var (
	errExpected = errors.New("expected unexpected error ;)")
)

type myFakeMsg struct {
	Value string
}

func (m myFakeMsg) Reset()                       {}
func (m myFakeMsg) ProtoMessage()                {}
func (m myFakeMsg) String() string               { return "doesn't matter" }
func (m myFakeMsg) ValidateBasic() error         { return nil }
func (m myFakeMsg) GetSigners() []sdk.AccAddress { return []sdk.AccAddress{sdk.AccAddress(`hello`)} }

type myFakeTx struct {
	msgs []myFakeMsg
}

func (m myFakeTx) GetMsgs() (msgs []sdk.Msg) {
	for _, msg := range m.msgs {
		msgs = append(msgs, msg)
	}
	return
}
func (m myFakeTx) ValidateBasic() error   { return nil }
func (m myFakeTx) AsAny() *codectypes.Any { return &codectypes.Any{} }

type fakeTx struct {
}

func (fakeTx) GetMsgs() (msgs []sdk.Msg) {
	return nil
}

func (fakeTx) ValidateBasic() error { return nil }

type fakeBroadcaster struct {
	tx            func(context.Context, []byte, bool) (*ctypes.ResultTx, error)
	broadcastSync func(context.Context, tmtypes.Tx) (*ctypes.ResultBroadcastTx, error)
}

func (f fakeBroadcaster) Tx(ctx context.Context, hash []byte, prove bool) (*ctypes.ResultTx, error) {
	if f.tx == nil {
		return nil, nil
	}
	return f.tx(ctx, hash, prove)
}

func (f fakeBroadcaster) BroadcastTxSync(ctx context.Context, tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
	if f.broadcastSync == nil {
		return nil, nil
	}
	return f.broadcastSync(ctx, tx)
}

func TestBroadcast(t *testing.T) {
	for _, tt := range []struct {
		name               string
		broadcaster        fakeBroadcaster
		txDecoder          sdk.TxDecoder
		txBytes            []byte
		waitTimeout        time.Duration
		isContextCancelled bool
		expectedRes        *sdk.TxResponse
		expectedErr        error
	}{
		{
			name: "simple success returns result",
			broadcaster: fakeBroadcaster{
				broadcastSync: func(_ context.Context, _ tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
					return &ctypes.ResultBroadcastTx{
						Code: 1,
						Hash: []byte(`123bob`),
					}, nil
				},
				tx: func(_ context.Context, hash []byte, _ bool) (*ctypes.ResultTx, error) {
					assert.Equal(t, []byte(`123bob`), hash)
					return &ctypes.ResultTx{}, nil
				},
			},
			txDecoder: func(txBytes []byte) (sdk.Tx, error) {
				return myFakeTx{
					[]myFakeMsg{{"hello"}},
				}, nil
			},
			expectedRes: &sdk.TxResponse{
				Tx: &codectypes.Any{},
			},
		},
		{
			name:        "success but timed out while waiting for tx",
			waitTimeout: time.Microsecond,
			broadcaster: fakeBroadcaster{
				broadcastSync: func(_ context.Context, _ tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
					return &ctypes.ResultBroadcastTx{
						Code: 1,
						Hash: []byte(`123bob`),
					}, nil
				},
				tx: func(_ context.Context, hash []byte, _ bool) (*ctypes.ResultTx, error) {
					<-time.After(time.Second)
					// return doesn't matter because it will timeout before return will make sense
					return nil, nil
				},
			},
			expectedErr: fmt.Errorf("timed out after: %d; %w", time.Microsecond, ErrTimeoutAfterWaitingForTxBroadcast),
		},
		{
			name: "broadcasting returns an error",
			broadcaster: fakeBroadcaster{
				broadcastSync: func(_ context.Context, _ tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
					return nil, errExpected
				},
			},
			expectedErr: errExpected,
		},
		{
			name: "broadcasting returns an error that Tx already exist",
			broadcaster: fakeBroadcaster{
				broadcastSync: func(_ context.Context, _ tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
					return nil, errors.New("tx already exists in cache")
				},
			},
			expectedErr: nil,
			expectedRes: &sdk.TxResponse{
				Code:      sdkerrors.ErrTxInMempoolCache.ABCICode(),
				Codespace: sdkerrors.ErrTxInMempoolCache.Codespace(),
				TxHash:    fmt.Sprintf("%X", tmhash.Sum([]byte{1, 2, 3, 4})),
			},
			txBytes: []byte{1, 2, 3, 4},
		},
		{
			name:               "request is canceled",
			isContextCancelled: true,
			broadcaster: fakeBroadcaster{
				broadcastSync: func(_ context.Context, _ tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
					return &ctypes.ResultBroadcastTx{
						Code: 1,
						Hash: []byte(`123bob`),
					}, nil
				},
				tx: func(_ context.Context, hash []byte, _ bool) (*ctypes.ResultTx, error) {
					assert.Equal(t, []byte(`123bob`), hash)
					return &ctypes.ResultTx{}, nil
				},
			},
			txDecoder: func(txBytes []byte) (sdk.Tx, error) {
				return myFakeTx{
					[]myFakeMsg{{"hello"}},
				}, nil
			},
			expectedRes: nil,
			expectedErr: context.Canceled,
		},
		{
			name: "tx decode failed",
			broadcaster: fakeBroadcaster{
				broadcastSync: func(_ context.Context, _ tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
					return &ctypes.ResultBroadcastTx{
						Code: 1,
						Hash: []byte(`123bob`),
					}, nil
				},
				tx: func(_ context.Context, hash []byte, _ bool) (*ctypes.ResultTx, error) {
					assert.Equal(t, []byte(`123bob`), hash)
					return &ctypes.ResultTx{}, nil
				},
			},
			txDecoder: func(txBytes []byte) (sdk.Tx, error) {
				return nil, sdkerrors.ErrTxDecode
			},
			expectedRes: nil,
			expectedErr: sdkerrors.ErrTxDecode,
		},
		{
			name: "result doesn't implement the interface intoAny",
			broadcaster: fakeBroadcaster{
				broadcastSync: func(_ context.Context, _ tmtypes.Tx) (*ctypes.ResultBroadcastTx, error) {
					return &ctypes.ResultBroadcastTx{
						Code: 1,
						Hash: []byte(`123bob`),
					}, nil
				},
				tx: func(_ context.Context, hash []byte, _ bool) (*ctypes.ResultTx, error) {
					assert.Equal(t, []byte(`123bob`), hash)
					return &ctypes.ResultTx{}, nil
				},
			},
			txDecoder: func(txBytes []byte) (sdk.Tx, error) {
				return fakeTx{}, nil
			},
			expectedRes: nil,
			expectedErr: fmt.Errorf("expecting a type implementing intoAny, got: %T", fakeTx{}),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			duration := 1 * time.Second
			if tt.waitTimeout > 0 {
				duration = tt.waitTimeout
			}
			ctx := context.Background()
			if tt.isContextCancelled {
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			}
			gotRes, gotErr := broadcastTx(
				ctx,
				tt.broadcaster,
				tt.txDecoder,
				tt.txBytes,
				duration,
			)
			if gotRes != nil {
				// Ignoring timestamp for tests
				gotRes.Timestamp = ""
			}
			assert.Equal(t, tt.expectedRes, gotRes)
			assert.Equal(t, gotErr, tt.expectedErr)
		})
	}
}

func TestCheckTendermintError(t *testing.T) {
	for _, tt := range []struct {
		name        string
		tx          tmtypes.Tx
		err         error
		expectedRes *sdk.TxResponse
	}{
		{
			name:        "Error is nil",
			tx:          nil,
			err:         nil,
			expectedRes: nil,
		},
		{
			name: "Tx already exist in the cache",
			tx:   tmtypes.Tx{1, 2, 3, 4},
			err:  errors.New("tx already exists in cache"),
			expectedRes: &sdk.TxResponse{
				Code:      sdkerrors.ErrTxInMempoolCache.ABCICode(),
				Codespace: sdkerrors.ErrTxInMempoolCache.Codespace(),
				TxHash:    fmt.Sprintf("%X", tmhash.Sum([]byte{1, 2, 3, 4})),
			},
		},
		{
			name: "mempool is full",
			tx:   tmtypes.Tx{5, 6, 7, 8},
			err:  errors.New("mempool is full"),
			expectedRes: &sdk.TxResponse{
				Code:      sdkerrors.ErrMempoolIsFull.ABCICode(),
				Codespace: sdkerrors.ErrMempoolIsFull.Codespace(),
				TxHash:    fmt.Sprintf("%X", tmhash.Sum([]byte{5, 6, 7, 8})),
			},
		},
		{
			name: "tx too large",
			tx:   tmtypes.Tx{9, 10, 11, 12},
			err:  errors.New("tx too large"),
			expectedRes: &sdk.TxResponse{
				Code:      sdkerrors.ErrTxTooLarge.ABCICode(),
				Codespace: sdkerrors.ErrTxTooLarge.Codespace(),
				TxHash:    fmt.Sprintf("%X", tmhash.Sum([]byte{9, 10, 11, 12})),
			},
		},
		{
			name:        "Unknown error",
			tx:          tmtypes.Tx{13, 14, 15, 16},
			err:         errors.New("unknown error"),
			expectedRes: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			actualRes := CheckTendermintError(tt.err, tt.tx)
			assert.Equal(t, tt.expectedRes, actualRes)
		})
	}
}
