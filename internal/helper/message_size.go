package helper

// MaxCommitOrTagMessageSize is the threshold for a commit/tag message,
// if exceeded then message is truncated and it's up to the client
// to request it in full separately.
var MaxCommitOrTagMessageSize = 10 * 1024
