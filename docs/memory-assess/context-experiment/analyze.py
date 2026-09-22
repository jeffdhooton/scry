"""Validate and summarize saved context-ablation reports. No network calls."""
import hashlib
import json
import math
import statistics
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
ARMS = ("compact", "organized", "relevant", "long")


def correct(row, task):
    answer = row["assessment"]["answers"][task]
    value = answer["choice"] if task == "assertion" else answer["noul"] >= .5
    return value == row["expected"][task]


def metrics(rows):
    result = {"calls": len(rows)}
    for task in ("supported", "durable", "assertion"):
        result[task] = sum(correct(r, task) for r in rows)
        if task != "assertion":
            result[task + "_fp"] = sum(not r["expected"][task] and not correct(r, task) for r in rows)
            result[task + "_fn"] = sum(r["expected"][task] and not correct(r, task) for r in rows)
            result[task + "_brier"] = statistics.mean(
                (r["assessment"]["answers"][task]["noul"] - r["expected"][task]) ** 2 for r in rows)
    times = sorted(r["assessment"]["latency_ms"] for r in rows)
    result["p50_ms"] = times[math.ceil(len(times) * .5) - 1]
    result["p95_ms"] = times[math.ceil(len(times) * .95) - 1]
    result["input_tokens"] = sum(r["assessment"]["usage"]["input_tokens"] for r in rows)
    result["output_tokens"] = sum(r["assessment"]["usage"]["output_tokens"] for r in rows)
    result["estimated_usd"] = result["input_tokens"] * .042 / 1_000_000
    return result


def main():
    manifest = json.loads((HERE / "manifest.json").read_text())
    for path, digest in manifest["sha256"].items():
        assert hashlib.sha256((ROOT / path).read_bytes()).hexdigest() == digest, path
    sources = {c["id"]: c for c in json.loads((HERE / "sources.json").read_text())}
    all_rows = []
    for batch in (1, 2):
        cases = {c["id"]: c for c in json.loads((HERE / f"batch-{batch}.json").read_text())}
        report = json.loads((HERE / f"results-batch-{batch}.json").read_text())
        assert report["model"] == "jev-1.13.0" and report["rubric_version"] == "memory-assess-v1"
        assert report["requested"] == report["completed"] == len(cases) == 96
        assert report["diagnostic_threshold"] == .5
        assert report["input_usd_per_million"] == .042
        rows = report["results"]
        assert len(rows) == len({r["id"] for r in rows}) == len(cases)
        assert set(cases) == {r["id"] for r in rows}
        for row in rows:
            assert row["expected"] == cases[row["id"]]["expected"]
            assert row["assessment"]["model"] == report["model"]
        m = metrics(rows)
        for task in ("supported", "durable"):
            assert m[task] == report["metrics"][task]["correct"]
            assert m[task + "_fp"] == report["metrics"][task]["false_positive"]
            assert m[task + "_fn"] == report["metrics"][task]["false_negative"]
            assert math.isclose(m[task + "_brier"], report["metrics"][task]["brier_score"])
        assert m["assertion"] == report["metrics"]["assertion_correct"]
        for unit in ("input_tokens", "output_tokens"):
            assert m[unit] == report["usage"][unit]
        assert math.isclose(m["estimated_usd"], report["estimated_usd"])
        assert m["p50_ms"] == report["p50_latency_ms"]
        assert m["p95_ms"] == report["p95_latency_ms"]
        all_rows.extend(rows)
    lookup = {tuple(r["id"].split("::")): r for r in all_rows}
    summary = {}
    lines = ["# Context ablation results — 2026-09-20", "",
             "All 192 calls completed. Pinned Jev 1.13.0, unchanged memory-assess-v1 rubric, "
             "0.5 diagnostic threshold. All cases and supplemental sources are fictional. "
             "Labels were frozen before inference and not revised afterward.", "",
             "## All calls", "",
             "| Context | Support | FP / FN | Durable | Assertion | Input tokens/call | p50 / p95 ms | Cost USD |",
             "| --- | --- | --- | --- | --- | --- | --- | --- |"]
    for arm in ARMS:
        rows = [r for r in all_rows if r["id"].startswith(arm + "::")]
        m = metrics(rows)
        assert m["calls"] == 48
        summary[arm] = {"all": m}
        lines.append(f'| {arm} | {m["supported"]}/48 | {m["supported_fp"]} / {m["supported_fn"]} | '
                     f'{m["durable"]}/48 | {m["assertion"]}/48 | {m["input_tokens"]/48:,.0f} | '
                     f'{m["p50_ms"]:.1f} / {m["p95_ms"]:.1f} | ${m["estimated_usd"]:.6f} |')
    for cohort in ("anchor", "fresh"):
        lines += ["", f"## {cohort.title()} cases", "",
                  "| Context | Calls | Support | FP / FN | Durable | Assertion | Support Brier |",
                  "| --- | --- | --- | --- | --- | --- | --- |"]
        for arm in ARMS:
            rows = [r for r in all_rows if r["id"].startswith(arm + "::") and
                    sources[r["id"].split("::")[1]]["cohort"] == cohort]
            m = metrics(rows)
            summary[arm][cohort] = m
            lines.append(f'| {arm} | {m["calls"]} | {m["supported"]} | {m["supported_fp"]} / '
                         f'{m["supported_fn"]} | {m["durable"]} | {m["assertion"]} | {m["supported_brier"]:.4f} |')
    lines += ["", "## Changes relative to compact", "",
              "A fix/regression is counted per paired case/repetition, separately for each judgment. "
              "These are correlated observations, not independent trial counts.", "",
              "| Context | Support fixes / regressions | Durability fixes / regressions | Assertion fixes / regressions |",
              "| --- | --- | --- | --- |"]
    for arm in ARMS[1:]:
        changes = {}
        for task in ("supported", "durable", "assertion"):
            fixes = regressions = 0
            for c in sources:
                for rep in ("r1", "r2"):
                    before = correct(lookup[("compact", c, rep)], task)
                    after = correct(lookup[(arm, c, rep)], task)
                    fixes += not before and after
                    regressions += before and not after
            changes[task] = {"fixes": fixes, "regressions": regressions}
        summary[arm]["vs_compact"] = changes
        lines.append("| " + arm + " | " + " | ".join(
            f'{changes[t]["fixes"]} / {changes[t]["regressions"]}'
            for t in ("supported", "durable", "assertion")) + " |")
    lines += ["", "## Long-context evidence position", "",
              "| Position | Support | Durable | Assertion |", "| --- | --- | --- | --- |"]
    for rep, position in (("r1", "first"), ("r2", "last")):
        m = metrics([r for r in all_rows if r["id"].startswith("long::") and r["id"].endswith(rep)])
        summary["long"][position] = m
        lines.append(f'| {position} | {m["supported"]}/24 | {m["durable"]}/24 | {m["assertion"]}/24 |')
    lines += ["", "## Per-case support probabilities", "",
              "Each cell lists r1 / r2. Expected labels remain fixed across all arms.", "",
              "| Case | Expected | Compact | Organized | Relevant | Long |",
              "| --- | --- | --- | --- | --- | --- |"]
    for case_id, c in sources.items():
        values = [" / ".join(f'{lookup[(arm, case_id, rep)]["assessment"]["answers"]["supported"]["noul"]:.2f}'
                             for rep in ("r1", "r2")) for arm in ARMS]
        lines.append(f'| {case_id} | {c["expected"]["supported"]} | ' + " | ".join(values) + " |")
    lines += ["", "## Remaining disagreements", "",
              "| Context | Case / repetition | Dimension | Expected | Observed |",
              "| --- | --- | --- | --- | --- |"]
    for key, row in sorted(lookup.items()):
        for task in ("supported", "durable", "assertion"):
            if not correct(row, task):
                answer = row["assessment"]["answers"][task]
                value = answer["choice"] if task == "assertion" else answer["noul"]
                lines.append(f'| {key[0]} | {key[1]} / {key[2]} | {task} | {row["expected"][task]} | {value} |')
    total = metrics(all_rows)
    summary["total"] = total
    lines += ["", f'Total: {total["input_tokens"]:,} input tokens; {total["output_tokens"]:,} output tokens; '
              f'estimated ${total["estimated_usd"]:.6f}. Cost uses returned usage and $0.042/M input, free output.', "",
              "## Limits", "",
              "This is a small, targeted synthetic development experiment. Repetitions and paired facts "
              "share evidence; aggregate call counts overstate the number of independent examples. "
              "Supplemental statements are hand-authored clarifications, sometimes restating the decisive "
              "fact. This tests what happens when useful context is available, not whether retrieval can "
              "find it accurately. Sources are serialized JSON text inside state.episode, not native nested "
              "state. Synthetic archive records are repetitive and easier to separate than real, conflicting "
              "project history. Position differences are confounded with per-call variation. Fresh cases "
              "target familiar error patterns and are not a representative holdout. No production admission "
              "or retention rule changed. See [protocol](PROTOCOL.md) and manifest for reproducibility.", ""]
    (HERE / "summary.json").write_text(json.dumps(summary, indent=2) + "\n")
    (HERE / "RESULTS.md").write_text("\n".join(lines))
    print("Validated every frozen file and independently recomputed report metrics.")
    for arm in ARMS:
        print(arm, summary[arm]["all"])
    print("total", total)


if __name__ == "__main__":
    main()
