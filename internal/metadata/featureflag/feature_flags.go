package featureflag

type FeatureFlag struct {
	Name        string `json:"name"`
	OnByDefault bool   `json:"on_by_default"`
}

// A set of feature flags used in Gitaly and Praefect.
// In order to support coverage of combined features usage all feature flags should be marked as enabled for the test.
// NOTE: if you add a new feature flag please add it to the `All` list defined below.
var (
	// GoFetchSourceBranch enables a go implementation of FetchSourceBranch
	GoFetchSourceBranch = FeatureFlag{Name: "go_fetch_source_branch", OnByDefault: false}
	// DistributedReads allows praefect to redirect accessor operations to up-to-date secondaries
	DistributedReads = FeatureFlag{Name: "distributed_reads", OnByDefault: true}
	// GoPostReceiveHook will bypass the ruby post-receive hook and use the go implementation
	GoPostReceiveHook = FeatureFlag{Name: "go_postreceive_hook", OnByDefault: true}
	// ReferenceTransactions will handle Git reference updates via the transaction service for strong consistency
	ReferenceTransactions = FeatureFlag{Name: "reference_transactions", OnByDefault: true}
	// ReferenceTransactionsOperationService will enable reference transactions for the OperationService
	ReferenceTransactionsOperationService = FeatureFlag{Name: "reference_transactions_operation_service", OnByDefault: true}
	// ReferenceTransactionsSmartHTTPService will enable reference transactions for the SmartHTTPService
	ReferenceTransactionsSmartHTTPService = FeatureFlag{Name: "reference_transactions_smarthttp_service", OnByDefault: true}
	// ReferenceTransactionsSSHService will enable reference transactions for the SSHService
	ReferenceTransactionsSSHService = FeatureFlag{Name: "reference_transactions_ssh_service", OnByDefault: true}
	// ReferenceTranasctiionsPrimaryWins will change transaction registration such that
	// secondaries will take part in transactions, but not influence their outcome.
	ReferenceTransactionsPrimaryWins = FeatureFlag{Name: "reference_transactions_primary_wins", OnByDefault: false}
	// ReferenceTransactionHook will enable the reference-transaction hook
	// introduced with Git v2.28.0 for voting on transactions
	ReferenceTransactionHook = FeatureFlag{Name: "reference_transaction_hook", OnByDefault: true}
	// RubyReferenceTransactionHook will enable the reference-transaction hook
	// introduced with Git v2.28.0 for voting on transactions in the Ruby sidecar.
	RubyReferenceTransactionHook = FeatureFlag{Name: "ruby_reference_transaction_hook", OnByDefault: false}
	// LogCommandStats will log additional rusage stats for commands
	LogCommandStats = FeatureFlag{Name: "log_command_stats", OnByDefault: false}
	// GoUserMergeBranch enables the Go implementation of UserMergeBranch
	GoUserMergeBranch = FeatureFlag{Name: "go_user_merge_branch", OnByDefault: false}
	// GoUserMergeToRef enable the Go implementation of UserMergeToRef
	GoUserMergeToRef = FeatureFlag{Name: "go_user_merge_to_ref", OnByDefault: false}
)

// All includes all feature flags.
var All = []FeatureFlag{
	GoFetchSourceBranch,
	DistributedReads,
	GoPostReceiveHook,
	ReferenceTransactions,
	ReferenceTransactionsOperationService,
	ReferenceTransactionsSmartHTTPService,
	ReferenceTransactionsSSHService,
	ReferenceTransactionsPrimaryWins,
	ReferenceTransactionHook,
	RubyReferenceTransactionHook,
	GoUserMergeBranch,
	GoUserMergeToRef,
}

const (
	GoPostReceiveHookEnvVar        = "GITALY_GO_POSTRECEIVE"
	ReferenceTransactionHookEnvVar = "GITALY_REFERENCE_TRANSACTION_HOOK"
)
