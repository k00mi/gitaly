package catfile

import (
	"context"
	"sync"

	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/gitaly/internal/git/repository"
	"gitlab.com/gitlab-org/gitaly/internal/metadata"
	"gitlab.com/gitlab-org/gitaly/internal/storage"
)

var catfileCacheCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "gitaly_catfile_cache_total",
		Help: "Counter of catfile cache hit/miss",
	},
	[]string{"type"},
)

var currentCatfileProcesses = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "gitaly_catfile_processes",
		Help: "Gauge of active catfile processes",
	},
)

var totalCatfileProcesses = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "gitaly_catfile_processes_total",
		Help: "Counter of catfile processes",
	},
)

var catfileLookupCounter = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "gitaly_catfile_lookups_total",
		Help: "Git catfile lookups by object type",
	},
	[]string{"type"},
)

const (
	// CacheFeatureFlagKey is the feature flag key for catfile batch caching. This should match
	// what is in gitlab-ce
	CacheFeatureFlagKey = "catfile-cache"

	// SessionIDField is the gRPC metadata field we use to store the gitaly session ID.
	SessionIDField = "gitaly-session-id"
)

func init() {
	prometheus.MustRegister(catfileCacheCounter)
	prometheus.MustRegister(currentCatfileProcesses)
	prometheus.MustRegister(totalCatfileProcesses)
	prometheus.MustRegister(catfileLookupCounter)
}

// Batch abstracts 'git cat-file --batch' and 'git cat-file --batch-check'.
// It lets you retrieve object metadata and raw objects from a Git repo.
//
// A Batch instance can only serve single request at a time. If you want to
// use it across multiple goroutines you need to add your own locking.
type Batch interface {
	Info(ctx context.Context, revspec string) (*ObjectInfo, error)
	Tree(ctx context.Context, revspec string) (*Object, error)
	Commit(ctx context.Context, revspec string) (*Object, error)
	Blob(ctx context.Context, revspec string) (*Object, error)
	Tag(ctx context.Context, revspec string) (*Object, error)
}

type batch struct {
	sync.Mutex
	*batchCheck
	*batchProcess
	cancel func()
	closed bool
}

// Info returns an ObjectInfo if spec exists. If spec does not exist the
// error is of type NotFoundError.
func (c *batch) Info(ctx context.Context, revspec string) (*ObjectInfo, error) {
	return c.batchCheck.info(revspec)
}

// Tree returns a raw tree object. It is an error if revspec does not
// point to a tree. To prevent this firstuse Info to resolve the revspec
// and check the object type. Caller must consume the Reader before
// making another call on C.
func (c *batch) Tree(ctx context.Context, revspec string) (*Object, error) {
	return c.batchProcess.reader(revspec, "tree")
}

// Commit returns a raw commit object. It is an error if revspec does not
// point to a commit. To prevent this first use Info to resolve the revspec
// and check the object type. Caller must consume the Reader before
// making another call on C.
func (c *batch) Commit(ctx context.Context, revspec string) (*Object, error) {
	return c.batchProcess.reader(revspec, "commit")
}

// Blob returns a reader for the requested blob. The entire blob must be
// read before any new objects can be requested from this Batch instance.
//
// It is an error if revspec does not point to a blob. To prevent this
// first use Info to resolve the revspec and check the object type.
func (c *batch) Blob(ctx context.Context, revspec string) (*Object, error) {
	return c.batchProcess.reader(revspec, "blob")
}

// Tag returns a raw tag object. Caller must consume the Reader before
// making another call on C.
func (c *batch) Tag(ctx context.Context, revspec string) (*Object, error) {
	return c.batchProcess.reader(revspec, "tag")
}

// Close closes the writers for batchCheck and batch. This is only used for
// cached Batches
func (c *batch) Close() {
	c.Lock()
	defer c.Unlock()

	if c.closed {
		return
	}

	c.closed = true
	if c.cancel != nil {
		// both c.batch and c.batchCheck have goroutines that listen on <ctx.Done()
		// when this is cancelled, it will cause those goroutines to close both writers
		c.cancel()
	}
}

func (c *batch) isClosed() bool {
	c.Lock()
	defer c.Unlock()
	return c.closed
}

// New returns a new Batch instance. It is important that ctx gets canceled
// somewhere, because if it doesn't the cat-file processes spawned by
// New() never terminate.
func New(ctx context.Context, locator storage.Locator, repo repository.GitRepo) (Batch, error) {
	if ctx.Done() == nil {
		panic("empty ctx.Done() in catfile.Batch.New()")
	}

	sessionID := metadata.GetValue(ctx, SessionIDField)
	if sessionID == "" {
		c, err := newBatch(ctx, locator, repo)
		if err != nil {
			return nil, err
		}
		return newInstrumentedBatch(c), err
	}

	cacheKey := newCacheKey(sessionID, repo)
	requestDone := ctx.Done()

	if c, ok := cache.Checkout(cacheKey); ok {
		go returnWhenDone(requestDone, cache, cacheKey, c)
		return newInstrumentedBatch(c), nil
	}

	// if we are using caching, create a fresh context for the new batch
	// and initialize the new batch with a cache key and cancel function
	cacheCtx, cacheCancel := context.WithCancel(context.Background())
	c, err := newBatch(cacheCtx, locator, repo)
	if err != nil {
		cacheCancel()
		return nil, err
	}

	c.cancel = cacheCancel
	go returnWhenDone(requestDone, cache, cacheKey, c)

	return newInstrumentedBatch(c), nil
}

func returnWhenDone(done <-chan struct{}, bc *batchCache, cacheKey key, c *batch) {
	<-done

	if c == nil || c.isClosed() {
		return
	}

	if c.hasUnreadData() {
		catfileCacheCounter.WithLabelValues("dirty").Inc()
		c.Close()
		return
	}

	bc.Add(cacheKey, c)
}

var injectSpawnErrors = false

type simulatedBatchSpawnError struct{}

func (simulatedBatchSpawnError) Error() string { return "simulated spawn error" }

func newBatch(ctx context.Context, locator storage.Locator, repo repository.GitRepo) (_ *batch, err error) {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancel()
		}
	}()

	b, err := newBatchProcess(ctx, locator, repo)
	if err != nil {
		return nil, err
	}

	batchCheck, err := newBatchCheck(ctx, locator, repo)
	if err != nil {
		return nil, err
	}

	return &batch{batchProcess: b, batchCheck: batchCheck}, nil
}

func newInstrumentedBatch(c Batch) Batch {
	return &instrumentedBatch{c}
}

type instrumentedBatch struct {
	Batch
}

func (ib *instrumentedBatch) Info(ctx context.Context, revspec string) (*ObjectInfo, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Batch.Info", opentracing.Tag{"revspec", revspec})
	defer span.Finish()

	catfileLookupCounter.WithLabelValues("info").Inc()

	return ib.Batch.Info(ctx, revspec)
}

func (ib *instrumentedBatch) Tree(ctx context.Context, revspec string) (*Object, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Batch.Tree", opentracing.Tag{"revspec", revspec})
	defer span.Finish()

	catfileLookupCounter.WithLabelValues("tree").Inc()

	return ib.Batch.Tree(ctx, revspec)
}

func (ib *instrumentedBatch) Commit(ctx context.Context, revspec string) (*Object, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Batch.Commit", opentracing.Tag{"revspec", revspec})
	defer span.Finish()

	catfileLookupCounter.WithLabelValues("commit").Inc()

	return ib.Batch.Commit(ctx, revspec)
}

func (ib *instrumentedBatch) Blob(ctx context.Context, revspec string) (*Object, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Batch.Blob", opentracing.Tag{"revspec", revspec})
	defer span.Finish()

	catfileLookupCounter.WithLabelValues("blob").Inc()

	return ib.Batch.Blob(ctx, revspec)
}

func (ib *instrumentedBatch) Tag(ctx context.Context, revspec string) (*Object, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Batch.Tag", opentracing.Tag{"revspec", revspec})
	defer span.Finish()

	catfileLookupCounter.WithLabelValues("tag").Inc()

	return ib.Batch.Tag(ctx, revspec)
}
