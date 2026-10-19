---
description: Run a GitHub Actions job locally with act
---
Run a GitHub Actions job locally with `act`.

Job name comes from $ARGUMENTS. If it is empty, use `unit`.
Never run the `postgres` job unless I asked for it: it starts a second container.

Setup, once: `make act-image`. It builds `adev-act:latest` (about 700 MB) because the
default act image has no CA bundle, no git and no C compiler, and its Debian release is
end-of-life. `.actrc` already pins the image, sets `--pull=false`, adds the artifact
server and removes containers after the run.

Rules:
1. Show the plan first: `make ci-act-dry ACT_JOB=<job>`. It starts no container.
2. Run for real with `make ci-act ACT_JOB=<job>`. Expect about 80 seconds for `unit`.
3. Report: step results, wall-clock duration, and `docker ps -a` filtered by `act` to prove nothing is left.
4. If act fails, report the exact error and stop. Do not retry in a loop.
5. Remember the local alternatives that need no container at all: `make ci` for the unit job, and `make dev-db` plus `make migrate-up` plus `make test-db` for the postgres job.
