package capella

import (
	"testing"

	state_native "github.com/prysmaticlabs/prysm/v3/beacon-chain/state/state-native"
	fieldparams "github.com/prysmaticlabs/prysm/v3/config/fieldparams"
	"github.com/prysmaticlabs/prysm/v3/config/params"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	enginev1 "github.com/prysmaticlabs/prysm/v3/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	"github.com/prysmaticlabs/prysm/v3/testing/require"
)

func TestWithdrawBalance(t *testing.T) {
	creds := make([]byte, fieldparams.RootLength)
	creds[0] = params.BeaconConfig().ETH1AddressWithdrawalPrefixByte
	val := &ethpb.Validator{
		WithdrawalCredentials: creds,
	}

	creds2 := make([]byte, fieldparams.RootLength)
	val2 := &ethpb.Validator{
		WithdrawalCredentials: creds2,
	}

	vals := []*ethpb.Validator{val, val2}
	base := &ethpb.BeaconStateCapella{
		NextWithdrawalIndex: 2,
		WithdrawalQueue:     make([]*enginev1.Withdrawal, 2),
		Validators:          vals,
		Balances: []uint64{
			params.BeaconConfig().MaxEffectiveBalance + params.BeaconConfig().MinDepositAmount,
			params.BeaconConfig().MaxEffectiveBalance,
		},
	}

	s, err := state_native.InitializeFromProtoCapella(base)
	require.NoError(t, err)
	post, err := withdrawBalance(s, 0, params.BeaconConfig().MinDepositAmount)
	require.NoError(t, err)

	expected, err := post.BalanceAtIndex(0)
	require.NoError(t, err)
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance, expected)

	expected, err = post.NextWithdrawalIndex()
	require.NoError(t, err)
	require.Equal(t, uint64(3), expected)

	queue, err := post.WithdrawalQueue()
	require.NoError(t, err)
	require.Equal(t, 3, len(queue))
	withdrawal := queue[2]
	require.Equal(t, uint64(2), withdrawal.WithdrawalIndex)
	require.Equal(t, params.BeaconConfig().MinDepositAmount, withdrawal.Amount)
	require.Equal(t, types.ValidatorIndex(0), withdrawal.ValidatorIndex)

	// BLS validator
	_, err = withdrawBalance(post, 1, params.BeaconConfig().MinDepositAmount)
	require.ErrorContains(t, "invalid withdrawal credentials", err)

	// Sucessive withdrawals is fine:
	post, err = withdrawBalance(post, 0, params.BeaconConfig().MinDepositAmount)
	require.NoError(t, err)

	// Underflow produces wrong amount (Spec Repo #3054)
	post, err = withdrawBalance(post, 0, params.BeaconConfig().MaxEffectiveBalance)
	require.NoError(t, err)
	queue, err = post.WithdrawalQueue()
	require.NoError(t, err)
	require.Equal(t, 5, len(queue))
	withdrawal = queue[4]
	require.Equal(t, params.BeaconConfig().MaxEffectiveBalance, withdrawal.Amount)
}