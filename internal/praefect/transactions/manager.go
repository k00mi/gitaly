package transactions

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/gitaly/internal/praefect/config"
)

var ErrNotFound = errors.New("transaction not found")

// Manager handles reference transactions for Praefect. It is required in order
// for Praefect to handle transactions directly instead of having to reach out
// to reference transaction RPCs.
type Manager struct {
	txIDGenerator         TransactionIDGenerator
	lock                  sync.Mutex
	transactions          map[uint64]*Transaction
	counterMetric         *prometheus.CounterVec
	delayMetric           *prometheus.HistogramVec
	subtransactionsMetric prometheus.Histogram
}

// TransactionIDGenerator is an interface for types that can generate transaction IDs.
type TransactionIDGenerator interface {
	// ID generates a new transaction identifier
	ID() uint64
}

type transactionIDGenerator struct {
	rand *rand.Rand
}

func newTransactionIDGenerator() *transactionIDGenerator {
	var seed [8]byte

	// Ignore any errors. In case we weren't able to generate a seed, the
	// best we can do is to just use the all-zero seed.
	cryptorand.Read(seed[:])
	source := rand.NewSource(int64(binary.LittleEndian.Uint64(seed[:])))

	return &transactionIDGenerator{
		rand: rand.New(source),
	}
}

func (t *transactionIDGenerator) ID() uint64 {
	return rand.Uint64()
}

// ManagerOpt is a self referential option for Manager
type ManagerOpt func(*Manager)

// WithTransactionIDGenerator is an option to set the transaction ID generator
func WithTransactionIDGenerator(generator TransactionIDGenerator) ManagerOpt {
	return func(mgr *Manager) {
		mgr.txIDGenerator = generator
	}
}

// NewManager creates a new transactions Manager.
func NewManager(cfg config.Config, opts ...ManagerOpt) *Manager {
	mgr := &Manager{
		txIDGenerator: newTransactionIDGenerator(),
		transactions:  make(map[uint64]*Transaction),
		counterMetric: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "gitaly",
				Subsystem: "praefect",
				Name:      "transactions_total",
				Help:      "Total number of transaction actions",
			},
			[]string{"action"},
		),
		delayMetric: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "gitaly",
				Subsystem: "praefect",
				Name:      "transactions_delay_seconds",
				Help:      "Delay between casting a vote and reaching quorum",
				Buckets:   cfg.Prometheus.GRPCLatencyBuckets,
			},
			[]string{"action"},
		),
		subtransactionsMetric: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "gitaly_praefect_subtransactions_per_transaction_total",
				Help:    "The number of subtransactions created for a single registered transaction",
				Buckets: []float64{0.0, 1.0, 2.0, 4.0, 8.0, 16.0, 32.0},
			},
		),
	}

	for _, opt := range opts {
		opt(mgr)
	}

	return mgr
}

func (mgr *Manager) Describe(descs chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(mgr, descs)
}

func (mgr *Manager) Collect(metrics chan<- prometheus.Metric) {
	mgr.counterMetric.Collect(metrics)
	mgr.delayMetric.Collect(metrics)
	mgr.subtransactionsMetric.Collect(metrics)
}

func (mgr *Manager) log(ctx context.Context) logrus.FieldLogger {
	return ctxlogrus.Extract(ctx).WithField("component", "transactions.Manager")
}

// CancelFunc is the transaction cancellation function returned by
// `RegisterTransaction`. Calling it will cause the transaction to be removed
// from the transaction manager.
type CancelFunc func() error

// RegisterTransaction registers a new reference transaction for a set of nodes
// taking part in the transaction. `threshold` is the threshold at which an
// election will succeed. It needs to be in the range `weight(voters)/2 <
// threshold <= weight(voters) to avoid indecidable votes.
func (mgr *Manager) RegisterTransaction(ctx context.Context, voters []Voter, threshold uint) (*Transaction, CancelFunc, error) {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	// Use a random transaction ID. Using monotonic incrementing counters
	// that reset on restart of Praefect would be suboptimal, as the chance
	// for collisions is a lot higher in case Praefect restarts when Gitaly
	// nodes still have in-flight transactions.
	transactionID := mgr.txIDGenerator.ID()

	transaction, err := newTransaction(transactionID, voters, threshold)
	if err != nil {
		return nil, nil, err
	}

	if _, ok := mgr.transactions[transactionID]; ok {
		return nil, nil, errors.New("transaction exists already")
	}
	mgr.transactions[transactionID] = transaction

	mgr.log(ctx).WithFields(logrus.Fields{
		"transaction_id": transactionID,
		"voters":         voters,
	}).Debug("RegisterTransaction")

	mgr.counterMetric.WithLabelValues("registered").Add(float64(len(voters)))

	return transaction, func() error {
		return mgr.cancelTransaction(ctx, transaction)
	}, nil
}

func (mgr *Manager) cancelTransaction(ctx context.Context, transaction *Transaction) error {
	mgr.lock.Lock()
	defer mgr.lock.Unlock()

	delete(mgr.transactions, transaction.ID())

	transaction.cancel()
	mgr.subtransactionsMetric.Observe(float64(transaction.CountSubtransactions()))

	var committed uint64
	state, err := transaction.State()
	if err != nil {
		return err
	}

	for _, result := range state {
		if result == VoteCommitted {
			committed++
		}
	}

	mgr.log(ctx).WithFields(logrus.Fields{
		"transaction_id":  transaction.ID(),
		"committed":       fmt.Sprintf("%d/%d", committed, len(state)),
		"subtransactions": transaction.CountSubtransactions(),
	}).Info("transaction completed")

	return nil
}

func (mgr *Manager) voteTransaction(ctx context.Context, transactionID uint64, node string, hash []byte) error {
	mgr.lock.Lock()
	transaction, ok := mgr.transactions[transactionID]
	mgr.lock.Unlock()

	if !ok {
		return ErrNotFound
	}

	if err := transaction.vote(ctx, node, hash); err != nil {
		return err
	}

	return nil
}

// VoteTransaction is called by a client who's casting a vote on a reference
// transaction. It waits until quorum was reached on the given transaction.
func (mgr *Manager) VoteTransaction(ctx context.Context, transactionID uint64, node string, hash []byte) error {
	start := time.Now()
	defer func() {
		delay := time.Since(start)
		mgr.delayMetric.WithLabelValues("vote").Observe(delay.Seconds())
	}()

	mgr.counterMetric.WithLabelValues("started").Inc()

	mgr.log(ctx).WithFields(logrus.Fields{
		"transaction_id": transactionID,
		"node":           node,
		"hash":           hex.EncodeToString(hash),
	}).Debug("VoteTransaction")

	if err := mgr.voteTransaction(ctx, transactionID, node, hash); err != nil {
		if errors.Is(err, ErrTransactionStopped) {
			mgr.counterMetric.WithLabelValues("stopped").Inc()
		} else if errors.Is(err, ErrTransactionVoteFailed) {
			mgr.counterMetric.WithLabelValues("aborted").Inc()

			mgr.log(ctx).WithFields(logrus.Fields{
				"transaction_id": transactionID,
				"node":           node,
				"hash":           hex.EncodeToString(hash),
			}).WithError(err).Error("VoteTransaction: did not reach quorum")
		} else {
			mgr.counterMetric.WithLabelValues("invalid").Inc()

			mgr.log(ctx).WithFields(logrus.Fields{
				"transaction_id": transactionID,
				"node":           node,
				"hash":           hex.EncodeToString(hash),
			}).WithError(err).Error("VoteTransaction: vote failed")
		}

		return err
	}

	mgr.log(ctx).WithFields(logrus.Fields{
		"transaction_id": transactionID,
		"node":           node,
		"hash":           hex.EncodeToString(hash),
	}).Debug("VoteTransaction: transaction committed")

	mgr.counterMetric.WithLabelValues("committed").Inc()

	return nil
}

// StopTransaction will gracefully stop a transaction.
func (mgr *Manager) StopTransaction(ctx context.Context, transactionID uint64) error {
	mgr.lock.Lock()
	transaction, ok := mgr.transactions[transactionID]
	mgr.lock.Unlock()

	if !ok {
		return ErrNotFound
	}

	if err := transaction.stop(); err != nil {
		return err
	}

	mgr.log(ctx).WithFields(logrus.Fields{
		"transaction_id": transactionID,
	}).Debug("VoteTransaction: transaction stopped")
	mgr.counterMetric.WithLabelValues("stopped").Inc()

	return nil
}
