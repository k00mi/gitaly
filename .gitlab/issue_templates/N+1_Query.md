Corresponding gitlab-rails issue: LINK

------------------------------------------------------------

**Stages**:

- [ ] Investigate

- [ ] Assess priority

- [ ] Server-side fixed LINK

- [ ] Client-side fixed LINK

**Affected RPC's**:
  - `Endpoint::Name`
  
------------------------------------------------------------

Process explanation:

### Investigate

If it's not clear what RPC's add up to the N+1 violation, do a new CI run on gitlab-ce/ee to find out.

### Assess priority

- Does this N+1 degrade the user experience?
- Does it cause more than 100 (extra) requests per second on gitlab.com?

If the answer to both questions is 'no' then downgrade the priority of this issue to `v1.1`.

------------------------------------------------------------
