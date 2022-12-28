package chain_registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/go-github/v43/github"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type CosmosGithubRegistry struct {
	log    *zap.Logger
	client *github.Client
}

func NewCosmosGithubRegistry(log *zap.Logger, c *http.Client) CosmosGithubRegistry {
	if c == nil {
		c = http.DefaultClient
	}
	return CosmosGithubRegistry{log: log, client: github.NewClient(c)}
}

func (c CosmosGithubRegistry) ListChains(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	tree, res, err := c.client.Git.GetTree(
		ctx,
		"cosmos",
		"chain-registry",
		"master",
		false)
	if err != nil || res.StatusCode != 200 {
		return nil, err
	}

	var chains []string
	for _, entry := range tree.Entries {
		if *entry.Type == "tree" && !strings.HasPrefix(*entry.Path, ".") {
			chains = append(chains, *entry.Path)
		}
	}
	return chains, nil
}

func (c CosmosGithubRegistry) GetChain(ctx context.Context, name string) (ChainInfo, error) {
	chainFileName := path.Join(name, "chain.json")
	fileContent, _, res, err := c.client.Repositories.GetContents(
		ctx,
		"cosmos",
		"chain-registry",
		chainFileName,
		&github.RepositoryContentGetOptions{})
	if err != nil || res.StatusCode != 200 {
		return ChainInfo{}, errors.Wrap(err, fmt.Sprintf("error fetching %s", chainFileName))
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return ChainInfo{}, err
	}

	result := NewChainInfo(c.log.With(zap.String("chain_name", name)))
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return ChainInfo{}, err
	}

	return result, nil
}

func (c CosmosGithubRegistry) SourceLink() string {
	return "https://github.com/cosmos/chain-registry"
}
