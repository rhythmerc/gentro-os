package metadata

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/rhythmerc/gentro-ui/services/games/models"
)

// OnResolveCallback is called when metadata is successfully resolved
type OnResolveCallback func(req models.FetchRequest, resolved models.ResolvedMetadata, resolverName string)

// Fetcher manages the async metadata fetching queue
type Fetcher struct {
	queue     chan models.FetchRequest
	workers   int
	resolvers []Resolver
	cancelMap map[string]context.CancelFunc
	onResolve OnResolveCallback
	mu        sync.RWMutex
	logger    *slog.Logger
	isRunning bool
	wg        sync.WaitGroup
}

// Resolver interface for metadata sources
type Resolver interface {
	Name() string
	Supports(source string, platform string) bool
	Resolve(ctx context.Context, req models.FetchRequest) (models.ResolvedMetadata, error)
}

// NewFetcher creates a new metadata fetcher
func NewFetcher(workers int, logger *slog.Logger) *Fetcher {
	if workers <= 0 {
		workers = 2
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Fetcher{
		queue:     make(chan models.FetchRequest, 100),
		workers:   workers,
		resolvers: make([]Resolver, 0),
		cancelMap: make(map[string]context.CancelFunc),
		logger:    logger,
	}
}

// RegisterResolver adds a metadata resolver
func (f *Fetcher) RegisterResolver(resolver Resolver) {
	f.resolvers = append(f.resolvers, resolver)
	f.logger.Info("registered metadata resolver", "name", resolver.Name())
}

// SetOnResolveCallback sets the callback for successful metadata resolution
func (f *Fetcher) SetOnResolveCallback(callback OnResolveCallback) {
	f.onResolve = callback
}

// Start begins the fetcher workers
func (f *Fetcher) Start() {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.isRunning {
		return
	}

	f.isRunning = true

	for i := 0; i < f.workers; i++ {
		f.wg.Add(1)
		go f.worker(i)
	}

	f.logger.Info("metadata fetcher started", "workers", f.workers)
}

// Stop gracefully shuts down the fetcher
func (f *Fetcher) Stop() {
	f.mu.Lock()
	if !f.isRunning {
		f.mu.Unlock()
		return
	}
	f.isRunning = false

	// Cancel all active fetches
	for _, cancel := range f.cancelMap {
		cancel()
	}
	f.cancelMap = make(map[string]context.CancelFunc)
	f.mu.Unlock()

	// Close queue and wait for workers
	close(f.queue)
	f.wg.Wait()

	f.logger.Info("metadata fetcher stopped")
}

// Queue adds a fetch request to the queue
func (f *Fetcher) Queue(req models.FetchRequest) error {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if !f.isRunning {
		return fmt.Errorf("fetcher is not running")
	}

	// Non-blocking send with timeout
	select {
	case f.queue <- req:
		f.logger.Info("queued metadata fetch request", "gameID", req.GameID, "instanceID", req.InstanceID)
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("queue is full")
	}
}

// Cancel cancels an active fetch for an instance
func (f *Fetcher) Cancel(instanceID string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if cancel, ok := f.cancelMap[instanceID]; ok {
		cancel()
		delete(f.cancelMap, instanceID)
		f.logger.Info("cancelled metadata fetch", "instanceID", instanceID)
	}
}

// worker processes fetch requests from the queue
func (f *Fetcher) worker(id int) {
	defer f.wg.Done()

	f.logger.Info("metadata fetcher worker started", "workerID", id)
	f.logger.Info(fmt.Sprintf("queue: %#v", f.queue))

	for req := range f.queue {
		f.processRequest(req)
	}

	f.logger.Info("metadata fetcher worker stopped", "workerID", id)
}

// processRequest handles a single fetch request
func (f *Fetcher) processRequest(req models.FetchRequest) {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	f.mu.Lock()
	f.cancelMap[req.InstanceID] = cancel
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		delete(f.cancelMap, req.InstanceID)
		f.mu.Unlock()
		cancel()
	}()

	f.logger.Info("processing metadata fetch",
		"instanceID", req.InstanceID,
		"gameID", req.GameID,
		"name", req.Name,
	)

	// Try each resolver in order, filtering by source/platform support
	var sourcesTried []string
	for _, resolver := range f.resolvers {
		select {
		case <-ctx.Done():
			f.logger.Info("metadata fetch cancelled", "instanceID", req.InstanceID)
			return
		default:
		}

		// Check if this resolver supports the game source/platform
		if !resolver.Supports(req.Source, req.Platform) {
			f.logger.Info("resolver does not support this game",
				"resolver", resolver.Name(),
				"source", req.Source,
				"platform", req.Platform,
			)
			continue
		}

		sourcesTried = append(sourcesTried, resolver.Name())

		resolved, err := resolver.Resolve(ctx, req)
		if err != nil {
			f.logger.Info("resolver failed",
				"resolver", resolver.Name(),
				"instanceID", req.InstanceID,
				"error", err,
			)
			continue
		}

		f.logger.Info("metadata resolved",
			"resolver", resolver.Name(),
			"instanceID", req.InstanceID,
			"gameName", resolved.GameMetadata.Name,
		)

		// Call the resolve callback if set
		if f.onResolve != nil {
			f.onResolve(req, resolved, resolver.Name())
		}

		// Success - we're done
		return
	}

	// No resolver succeeded
	f.logger.Warn("all metadata resolvers failed",
		"instanceID", req.InstanceID,
		"sourcesTried", sourcesTried,
	)
}

// LocalCacheResolver implements a local-only metadata resolver
type LocalCacheResolver struct {
	// Could cache previously fetched metadata here
}

func (r *LocalCacheResolver) Name() string {
	return "local_cache"
}

// Supports returns true for all sources (fallback resolver)
func (r *LocalCacheResolver) Supports(source, platform string) bool {
	return true
}

func (r *LocalCacheResolver) Resolve(ctx context.Context, req models.FetchRequest) (models.ResolvedMetadata, error) {
	// For now, just return basic metadata from filename
	// This provides immediate fallback while async fetch happens

	return models.ResolvedMetadata{
		GameMetadata: models.GameMetadata{
			Name: req.Name,
		},
		ArtURLs: make(map[string]string),
	}, fmt.Errorf("local cache only provides fallback data")
}
