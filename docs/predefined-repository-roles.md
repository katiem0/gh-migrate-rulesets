# Predefined (base) repository role IDs

GitHub calls these **predefined (base) roles**. They cover `Read`, `Triage`, `Write`, `Maintain`, and
`Admin`.

## Mapping predefined-role bypass IDs with `--actor-mapping`

Bypass actors for **teams**, **integrations/apps**, and **custom repository roles** are translated
automatically by name or slug against the target organization. **Predefined repository roles** (e.g.
`Write`, `Maintain`, `Admin`) cannot be resolved this way, because GitHub does not expose an API to
look up predefined role IDs by name on the target. Their IDs are consistent across GitHub.com and
GitHub Enterprise Cloud, but can differ on GitHub Enterprise Cloud with data residency (`*.ghe.com`)
tenants.

For those cases, supply an actor mapping `csv` with `--actor-mapping`. Each row maps a source actor
ID to the corresponding target ID, keyed by `actor_type` and `source_id`. The mapping applies only to
the live `--source-org` path and cannot be combined with `--from-file`. A ready-to-fill template is
provided at [`actor-mapping-template.csv`](actor-mapping-template.csv):

```csv
actor_type,source_id,source_name,target_id
RepositoryRole,2,Maintain,2
RepositoryRole,4,Write,4
RepositoryRole,5,Admin,7
```

The `source_name` column is a reference only and is ignored. Rows with a blank `target_id` fall back
to the automatic name-based resolution, so you only need to fill in the IDs the target org actually
differs on. An explicit mapping entry always wins, so `--actor-mapping` can also override a renamed
team, app, or custom role.

## Migrating predefined-role bypass IDs with --from-file

Because `--actor-mapping` cannot be used with `--from-file`, imports from a `csv` must carry the
correct target predefined role IDs directly in the exported `csv`'s bypass actor column. Use the
platform ID table below as a starting point to map source IDs to target IDs, then verify the IDs on
your own source and target instances before importing.

## Finding predefined role IDs for your instance

Predefined role IDs (e.g. `Write`, `Maintain`, `Admin`) are not exposed by any lookup API and can
differ per platform, so you may need to confirm them on both the source and target.

The following IDs have been observed per platform. Treat them as a starting reference and verify against
your own instances before relying on them, as they are not officially guaranteed to be stable:

| Platform | `write` | `maintain` | `admin` |
| --- | --- | --- | --- |
| GitHub Enterprise Server (GHES) | 2 | 5 | 3 |
| GitHub Enterprise Cloud (GHEC) / EMU | 4 | 2 | 5 |
| Data residency — US (`ghe.com`) | 6 | 21 | 11 |

To confirm the IDs on any instance yourself:

1. In a single repository, create three rulesets and give each one a bypass actor for a different
   predefined role (`Maintain`, `Write`, `Admin`). Naming each ruleset after its bypass actor makes the
   output easier to read.
2. Run the following against that instance to list each ruleset's bypass actor IDs:

```sh
export GH_HOST="ghes.example.com" # only needed for GitHub Enterprise Server

ORG="ORG"
REPO="repo"

gh api "/repos/$ORG/$REPO/rulesets" --jq '.[].id' |
while read -r id; do
  gh api "/repos/$ORG/$REPO/rulesets/$id"
done | jq -s '[.[] | {
  name,
  bypass_actors: [.bypass_actors[]? | {actor_id, actor_type, bypass_mode}]
}]'
```

Repeat this on both the source and target instances to determine the `source_id` and `target_id`
values to fill into `actor-mapping.csv`.
