package graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	dataloader "github.com/graph-gophers/dataloader/v7"
	"github.com/lib/pq"
	"github.com/smart-contract-event-indexer/shared/models"
	"github.com/smart-contract-event-indexer/shared/utils"
)

type loadersKey struct{}

// Loaders bundles dataloaders for the GraphQL layer.
type Loaders struct {
	ContractByAddress *dataloader.Loader[string, *models.Contract]
	StatsByAddress    *dataloader.Loader[string, *models.ContractStats]
}

// LoaderFactory builds request-scoped dataloaders.
type LoaderFactory struct {
	db     *sql.DB
	logger utils.Logger
}

// NewLoaderFactory creates a dataloader factory.
func NewLoaderFactory(db *sql.DB, logger utils.Logger) *LoaderFactory {
	return &LoaderFactory{
		db:     db,
		logger: logger,
	}
}

// New creates request-scoped dataloaders.
func (f *LoaderFactory) New() *Loaders {
	return &Loaders{
		ContractByAddress: dataloader.NewBatchedLoader(f.contractBatch),
		StatsByAddress:    dataloader.NewBatchedLoader(f.contractStatsBatch),
	}
}

// WithLoaders injects loaders into a context.
func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loadersKey{}, loaders)
}

// GetLoaders extracts loaders from context if present.
func GetLoaders(ctx context.Context) *Loaders {
	loaders, _ := ctx.Value(loadersKey{}).(*Loaders)
	return loaders
}

func (f *LoaderFactory) contractBatch(ctx context.Context, keys []string) []*dataloader.Result[*models.Contract] {
	results := make([]*dataloader.Result[*models.Contract], len(keys))
	if len(keys) == 0 {
		return results
	}

	normalized := make([]string, 0, len(keys))
	addrIndex := make(map[string][]int)
	for idx, key := range keys {
		addr := strings.ToLower(strings.TrimSpace(key))
		normalized = append(normalized, addr)
		addrIndex[addr] = append(addrIndex[addr], idx)
	}

	query := `
SELECT id, address, abi, name, start_block, current_block, confirm_blocks, created_at, updated_at
FROM contracts
WHERE LOWER(address) = ANY($1)
`

	rows, err := f.db.QueryContext(ctx, query, pq.Array(normalized))
	if err != nil {
		for i := range results {
			results[i] = &dataloader.Result[*models.Contract]{Error: err}
		}
		return results
	}
	defer rows.Close()

	found := make(map[string]*models.Contract)
	for rows.Next() {
		var contract models.Contract
		if err := rows.Scan(
			&contract.ID,
			&contract.Address,
			&contract.ABI,
			&contract.Name,
			&contract.StartBlock,
			&contract.CurrentBlock,
			&contract.ConfirmBlocks,
			&contract.CreatedAt,
			&contract.UpdatedAt,
		); err != nil {
			for i := range results {
				results[i] = &dataloader.Result[*models.Contract]{Error: err}
			}
			return results
		}
		found[strings.ToLower(string(contract.Address))] = &contract
	}

	for key, indexes := range addrIndex {
		contract := found[key]
		for _, idx := range indexes {
			if contract == nil {
				results[idx] = &dataloader.Result[*models.Contract]{Error: sql.ErrNoRows}
			} else {
				results[idx] = &dataloader.Result[*models.Contract]{Data: contract}
			}
		}
	}

	return results
}

func (f *LoaderFactory) contractStatsBatch(ctx context.Context, keys []string) []*dataloader.Result[*models.ContractStats] {
	results := make([]*dataloader.Result[*models.ContractStats], len(keys))
	if len(keys) == 0 {
		return results
	}

	normalized := make([]string, 0, len(keys))
	addrIndex := make(map[string][]int)
	for idx, key := range keys {
		addr := strings.ToLower(strings.TrimSpace(key))
		normalized = append(normalized, addr)
		addrIndex[addr] = append(addrIndex[addr], idx)
	}

	query := `
WITH uniq_addresses AS (
    SELECT e.contract_address,
           COUNT(DISTINCT LOWER(value)) FILTER (WHERE value LIKE '0x%') AS unique_addresses
    FROM events e,
         LATERAL jsonb_each_text(e.args)
    WHERE LOWER(e.contract_address) = ANY($1)
    GROUP BY e.contract_address
)
SELECT 
    c.address,
    COUNT(e.id) AS total_events,
    COALESCE(MAX(e.block_number), c.current_block) AS latest_block,
    c.current_block,
    GREATEST(c.current_block - COALESCE(MAX(e.block_number), c.start_block), 0) AS indexer_delay,
    COALESCE(MAX(e.created_at), c.updated_at) AS last_updated,
    COALESCE(u.unique_addresses, 0) AS unique_addresses
FROM contracts c
LEFT JOIN events e ON c.address = e.contract_address
LEFT JOIN uniq_addresses u ON u.contract_address = c.address
WHERE LOWER(c.address) = ANY($1)
GROUP BY c.address, c.current_block, c.start_block, c.updated_at, u.unique_addresses
`

	rows, err := f.db.QueryContext(ctx, query, pq.Array(normalized))
	if err != nil {
		for i := range results {
			results[i] = &dataloader.Result[*models.ContractStats]{Error: err}
		}
		return results
	}
	defer rows.Close()

	found := make(map[string]*models.ContractStats)
	for rows.Next() {
		var stats models.ContractStats
		var lastUpdated time.Time
		var uniqueAddresses int64
		if err := rows.Scan(
			&stats.ContractAddress,
			&stats.TotalEvents,
			&stats.LatestBlock,
			&stats.CurrentBlock,
			&stats.IndexerDelay,
			&lastUpdated,
			&uniqueAddresses,
		); err != nil {
			for i := range results {
				results[i] = &dataloader.Result[*models.ContractStats]{Error: err}
			}
			return results
		}
		stats.LastUpdated = lastUpdated
		if uniqueAddresses > 0 {
			value := int(uniqueAddresses)
			stats.UniqueAddresses = &value
		}
		found[strings.ToLower(string(stats.ContractAddress))] = &stats
	}

	for key, indexes := range addrIndex {
		stat := found[key]
		for _, idx := range indexes {
			if stat == nil {
				results[idx] = &dataloader.Result[*models.ContractStats]{Error: sql.ErrNoRows}
			} else {
				results[idx] = &dataloader.Result[*models.ContractStats]{Data: stat}
			}
		}
	}

	return results
}

// encodeRawLog marshals event args for raw log fallback.
func encodeRawLog(args models.JSONB) *string {
	if len(args) == 0 {
		return nil
	}
	bytes, err := json.Marshal(args)
	if err != nil {
		return nil
	}
	raw := string(bytes)
	return &raw
}
