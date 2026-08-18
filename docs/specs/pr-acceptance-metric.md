# Spec: PR acceptance metric

One paragraph: defines the single quality metric this repo tracks for its
pull-request stream — the monthly acceptance rate — precisely enough that any
agent or human computes the same number from the same window. Consumers: the
[quality loop](../design/quality-loop.md) and anyone reporting on clix's
review stream. `docs/metrics.md` is a conformance alias for this file
([ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)).
`scripts/check-docs.mjs` pins this file's `## Definition` and `## Rules`
headings so prose and tooling cannot drift apart silently.

## Definition

```text
accepted PRs / (accepted PRs + closed, unmerged PRs) × 100
```

| Term | Meaning |
| --- | --- |
| Accepted PR | Any pull request merged into `main` during the reporting period |
| Closed, unmerged PR | Any pull request closed without merging during the reporting period |
| Reporting period | One UTC calendar month; a PR belongs to the month of its merge or close date |
| Open PRs | Excluded until merged or closed |

The result is a percentage rounded to two decimal places. A month with no
resolved pull requests reports `null` rather than a misleading 0%.

Data source (`--repo frostyard/clix` is explicit so the command works from
any checkout):

```bash
START=2026-08-01
END=2026-08-31

gh pr list --repo frostyard/clix --state all \
  --search "closed:${START}..${END}" --limit 1000 \
  --json mergedAt |
  jq '
    . as $prs
    | ($prs | map(select(.mergedAt != null)) | length) as $accepted
    | ($prs | map(select(.mergedAt == null)) | length) as $closed
    | {
        accepted: $accepted,
        closed_unmerged: $closed,
        acceptance_rate: (
          if ($accepted + $closed) == 0 then null
          else (($accepted * 10000 / ($accepted + $closed) | round) / 100)
          end
        )
      }
  '
```

## Rules

- Report the rate monthly after the UTC month closes; track the month, the
  accepted count, the closed-unmerged count, and the percentage so changes in
  review volume stay visible alongside the rate.
- Interpret it with rejection reasons, superseded work, CI outcomes, review
  feedback, and sample size — clix's stream is small (Dependabot bumps and
  occasional API work), so a single closed PR moves the number a lot;
  acceptance alone is not a quality score for human or automated
  contributors.
- Only repository metadata already public through GitHub belongs in the
  computation or its report — no credentials, private prompts, or personal
  data.
- The metric is observational: it never gates a merge by itself; gates live
  in the [review rubric](pr-review-rubric.md) and CI.

## References

- Rationale: [ADR-0001](../adr/0001-acmm-conformance-via-canonical-aliases.md)
- Context: [design/quality-loop.md](../design/quality-loop.md)
