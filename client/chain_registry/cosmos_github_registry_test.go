package chain_registry_test

import (
	"context"
	"go.uber.org/zap"
	"net/http"
	"testing"

	"github.com/google/go-github/v43/github"
	"github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/require"
	"github.com/volumefi/lens/client/chain_registry"
	"go.uber.org/zap/zaptest"
)

func TestListChains(t *testing.T) {
	// Setup
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposGitTreesByOwnerByRepoByTreeSha,
			github.Tree{
				Entries: []*github.TreeEntry{
					{Type: pointer("tree"), Path: pointer("_IBC")},
					{Type: pointer("tree"), Path: pointer("_non-cosmos")},
					{Type: pointer("tree"), Path: pointer("acrechain")},
					{Type: pointer("tree"), Path: pointer("agoric")},
					{Type: pointer("tree"), Path: pointer("aioz")},
				},
			},
		),
	)
	registry := chain_registry.NewCosmosGithubRegistry(zaptest.NewLogger(t), mockedHTTPClient)

	// Action
	got, err := registry.ListChains(context.Background())

	// Assert
	require.Nil(t, err)
	require.Equal(t, []string{"_IBC", "_non-cosmos", "acrechain", "agoric", "aioz"}, got)
}

func TestListChains_Failed(t *testing.T) {
	// Setup
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposGitTreesByOwnerByRepoByTreeSha,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mock.WriteError(
					w,
					http.StatusInternalServerError,
					"github went belly up or something",
				)
			}),
		),
	)
	registry := chain_registry.NewCosmosGithubRegistry(zaptest.NewLogger(t), mockedHTTPClient)

	// Action
	_, err := registry.ListChains(context.Background())

	// Assert
	require.Error(t, err)
}

func TestGetChain(t *testing.T) {
	// Setup
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposContentsByOwnerByRepoByPath,
			github.RepositoryContent{
				Content: pointer(`
				{
					"chain_name":"agoric", 
					"status":"active", 
					"network_type":"network324",
					"chain_id":"5674936249",
					"daemon_name":"daemon564"
				}`),
			},
		),
	)
	log := zaptest.NewLogger(t)
	registry := chain_registry.NewCosmosGithubRegistry(log, mockedHTTPClient)

	// Action
	actual, err := registry.GetChain(context.Background(), "agoric")

	// Assert
	expected := chain_registry.NewChainInfo(log.With(zap.String("chain_name", "agoric")))
	expected.ChainName = "agoric"
	expected.Status = "active"
	expected.NetworkType = "network324"
	expected.ChainID = "5674936249"
	expected.DaemonName = "daemon564"
	require.Nil(t, err)
	require.Equal(t, expected, actual)
}

func TestGetChain_Failed(t *testing.T) {
	// Setup
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatchHandler(
			mock.GetReposContentsByOwnerByRepoByPath,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mock.WriteError(
					w,
					http.StatusInternalServerError,
					"github went belly up or something",
				)
			}),
		),
	)
	log := zaptest.NewLogger(t)
	registry := chain_registry.NewCosmosGithubRegistry(log, mockedHTTPClient)

	// Action
	_, err := registry.GetChain(context.Background(), "agoric")

	// Assert
	require.Error(t, err)
}

func TestGetChain_WithMalformedContend(t *testing.T) {
	// Setup
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposContentsByOwnerByRepoByPath,
			github.RepositoryContent{
				Content: pointer(`Malformed`),
			},
		),
	)
	log := zaptest.NewLogger(t)
	registry := chain_registry.NewCosmosGithubRegistry(log, mockedHTTPClient)

	// Action
	_, err := registry.GetChain(context.Background(), "agoric")

	// Assert
	require.Error(t, err)
}

func TestGetChain_WithMalformedBase64Contend(t *testing.T) {
	// Setup
	mockedHTTPClient := mock.NewMockedHTTPClient(
		mock.WithRequestMatch(
			mock.GetReposContentsByOwnerByRepoByPath,
			github.RepositoryContent{
				Encoding: pointer("base64"),
				Content:  pointer(`4af34ba3452`),
			},
		),
	)
	log := zaptest.NewLogger(t)
	registry := chain_registry.NewCosmosGithubRegistry(log, mockedHTTPClient)

	// Action
	_, err := registry.GetChain(context.Background(), "agoric")

	// Assert
	require.Contains(t, err.Error(), "illegal base64 data")
}

func pointer(s string) *string {
	return &s
}
