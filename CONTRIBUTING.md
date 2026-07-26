# Contributing

Thanks for helping improve OpAMP Commander. This guide covers the local workflow
and, in particular, how branches are expected to live and die so that merged work
doesn't linger and drift.

## Getting started

See the [README](README.md#development) for setup, build, test, and lint commands.
Before opening a pull request, run the relevant checks:

```sh
make lint && make test        # Go
cd web && npx tsc --noEmit && npm run lint && npm test && npm run build   # web
```

## Commits and pull requests

- **Conventional Commits.** Format subjects as `type(scope): summary`
  (e.g. `feat(agent): …`, `fix(opamp): …`, `chore(dev): …`). Common types:
  `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `chore`, `ci`.
- **Reference the issue** the PR closes in the body (`Closes #494`).
- **Squash merge.** PRs land on `main` as a single squashed commit. This means the
  individual commits on your branch do *not* appear in `main`'s history — which is
  why `git branch --merged` cannot detect a squash-merged branch (see below).

## Branch lifecycle

Branches are cheap to create and cheap to delete. The failure mode we care about
is the opposite: a branch that was **already merged** hanging around, masquerading
as work-in-progress, and silently carrying a stale version of code that `main` has
since moved past. Rebasing such a branch can produce add/add conflict storms and,
worse, can *revert* a shipped feature during a careless conflict resolution.

### Naming

Use `type/short-description`, optionally suffixed with the issue number:
`feat/agent-search`, `fix/http-idle-timeout`, `chore/branch-lifecycle-doc-494`.

### While a branch is open

- **Rebase onto `main`, don't merge `main` in.** `git fetch origin && git rebase
  origin/main` keeps history linear and the diff honest. Avoid
  `git merge main` into a feature branch — it muddies the diff and hides drift.
- **Rebase regularly**, not just once at the end. A branch that tracks `main`
  closely rarely conflicts; one that diverges for weeks becomes a hazard.

### Rebase vs. abandon

Before rebasing an old branch, first ask whether its work has **already landed**:

```sh
git fetch origin --prune
# Has this branch's content already been merged (even via squash)?
git log --oneline origin/main..my-branch     # commits unique to the branch
git diff origin/main...my-branch --stat        # net change vs. main
```

- If the unique commits / diff are **empty or already represent shipped work**,
  the branch is superseded — **delete it, don't rebase it.** Rebasing a
  superseded branch is what produces add/add conflicts and risks reverting `main`.
- If it still holds genuinely unmerged work, rebase it onto `origin/main` and
  continue.

### After a merge

Merged PR branches are deleted automatically on the remote
(`delete_branch_on_merge` is enabled on the repository), but the **local copy
stays** until you remove it. Prune periodically:

```sh
git fetch origin --prune          # drop remote-tracking refs whose remote is gone

# List local branches whose upstream was deleted (typically = merged):
git for-each-ref --format='%(refname:short) %(upstream:track)' refs/heads \
  | grep '\[gone\]'

# Delete one:
git branch -D <branch>
```

`git branch -d` (lowercase) refuses to delete a branch that isn't reachable-merged
into `main` — which, because of squash merges, includes most already-merged
branches. Verify a branch is truly merged (with the `git log`/`git diff` checks
above) before reaching for `git branch -D`.
