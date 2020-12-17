package git

// ReceivePackConfig contains config options we want to enforce when
// receiving a push with git-receive-pack.
func ReceivePackConfig() []GlobalOption {
	return []GlobalOption{
		// In case the repository belongs to an object pool, we want to prevent
		// Git from including the pool's refs in the ref advertisement. We do
		// this by rigging core.alternateRefsCommand to produce no output.
		// Because Git itself will append the pool repository directory, the
		// command ends with a "#". The end result is that Git runs `/bin/sh -c 'exit 0 # /path/to/pool.git`.
		ConfigPair{Key: "core.alternateRefsCommand", Value: "exit 0 #"},

		// In the past, there was a bug in git that caused users to
		// create commits with invalid timezones. As a result, some
		// histories contain commits that do not match the spec. As we
		// fsck received packfiles by default, any push containing such
		// a commit will be rejected. As this is a mostly harmless
		// issue, we add the following flag to ignore this check.
		ConfigPair{Key: "receive.fsck.badTimezone", Value: "ignore"},
	}
}
