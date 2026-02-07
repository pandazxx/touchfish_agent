# Tests

## Unit test script

Run the unit tests (defaults to Docker):

```bash
./tests/unit_test.sh
```

To run without Docker:

```bash
./tests/unit_test.sh --no-container
```

To enable verbose command tracing:

```bash
./tests/unit_test.sh --verbose
```

The report is written to:

```
./tests/report.txt
```

## Notes

- Tests mock `gh`, `git`, and `codex` via `tests/mocks`.
- Codex is treated as a black box; tests verify the exact prompt input.
- Test cases are data-driven and live in `tests/cases`.
- Test data fixtures live in `tests/data`.
- Every test case records the `DETAIL_REQUIREMENT.md` section(s) it validates.
