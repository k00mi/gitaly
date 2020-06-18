/title RFC: <REPLACE TITLE>
/label ~"Gitaly RFC" ~backstage

**Accepting comments until <YYYY-MM-DD> UTC**

## Contributor checklist

Check all items before moving onto the next section.

### Pre-Review

- [ ] Verify you have reviewed and understand the [RFC guidelines](doc/rfcs/README.md).
- [ ] Replace all placeholders in the RFC template:
  - [ ] Replace `<REPLACE TITLE>` with the RFC title
  - [ ] Replace `<REPLACE ABSTRACT>` with a short summary of the RFC contents.
- [ ] Replace all placeholders in this MR template:
  - [ ] Replace `<REPLACE TITLE>` at the top of the MR description
  - [ ] Replace `<YYYY-MM-DD>` with the UTC deadline for accepting comments.
- [ ] Follow the [contributor guidelines for reviews](https://gitlab.com/gitlab-org/gitaly/-/blob/master/REVIEWING.md#tips-for-the-contributor)

### Ready for review

- [ ] Require a minimum of 2 maintainer approvals. Increase if warranted.
- [ ] Once ready for review, announce the RFC to the Gitaly team (`/cc @gl-gitaly`)
  - [ ] Announce a time window for accepting comments (at least a week).

### Post-Approval

Once the minimum number of maintainers have approved:

- [ ] Wait until the declared time window expires before taking action to give others a chance to comment.
- [ ] Once the time window expires, decide to either merge the RFC as is, or address new feedback.
- [ ] If you choose to make changes, ping the existing approvers so that they may review the changes.
- [ ] Merge when ready. If you do not have write access to the repository, ping a Gitaly maintainer `@gl-gitaly`.

## Reviewer instructions

1. Familiarize yourself with the [RFC Guidelines](doc/rfcs/README.md)
1. Identify other reviewers who can offer constructive feedback. Offer them the opportunity to review.
1. For your review, follow the [Gitaly reviewing guide](https://gitlab.com/gitlab-org/gitaly/-/blob/master/REVIEWING.md).
1. Approve, but do not merge. Let the contributor merge when ready.
