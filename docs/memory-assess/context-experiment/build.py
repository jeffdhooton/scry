"""Build synthetic context ablations; never calls a provider or reads credentials."""
import copy
import json
import random
from pathlib import Path

HERE = Path(__file__).resolve().parent
PARENT = HERE.parent
ARMS = ("compact", "organized", "relevant", "long")


def build():
    originals = {}
    for filename in ("synthetic.json", "expanded-v1.json"):
        originals.update({c["id"]: c for c in json.loads((PARENT / filename).read_text())})
    cases = []

    def anchor(ids, sources):
        for case_id in ids:
            c = copy.deepcopy(originals[case_id])
            c.update(cohort="anchor", sources=sources)
            cases.append(c)

    anchor(["corrections--retracted-host--a", "corrections--retracted-host--b"], [
        "User, following the correction: Atlas was an assistant guess, not a former deployment location. Our current Juniper service is on Beacon. There is no Juniper installation on Atlas.",
        "User, infrastructure inventory: Beacon is the Juniper application host. Juniper is the service name; Atlas and Beacon are two different machines."])
    anchor(["scope--actor--a", "scope--actor--b"], [
        "User, Cedar team directory: Mira is the release owner, responsible for preparing, approving and publishing Cedar releases. Sol writes the documentation and does not hold the release-owner role.",
        "User: By owns releases I mean accountable for the release process, not ownership of a company. Mira and Sol are different people."])
    anchor(["durable-preference"], [
        "User: US English and runnable examples are my standing documentation preferences, already in effect for Lantern. Apply them in future sessions too; this is the way I prefer documentation to be written."])
    anchor(["completed-deployment"], [
        "User: The staging deployment I just reported is a completed event. Keep this deployment milestone in the project history so later sessions know staging was brought up and its health check passed. This is my report, not an independent verification by the assistant."])
    anchor(["preserve-negated-constraint"], [
        "User, Lantern data policy: Customer records must stay out of every external analytics service. This is a standing prohibition. No exception has been granted; it applies to all future work on Lantern."])
    anchor(["transient-progress"], [
        "Assistant: Opening this file is just what I am doing at this moment in this conversation. It is not a project milestone, enduring preference, decision, or change to the code."])
    anchor(["attribution--uncertain-belief--a", "attribution--uncertain-belief--b"], [
        "Assistant, follow-up: I still have not inspected Birch's importer or tested ZIP support. My earlier statement was a guess. I do not have a verified result.",
        "User: ZIP import support is an open project question worth retaining for future sessions. Preserve the assistant's uncertainty and lack of checking when recording what was said."])

    def pair(name, episode, sources, members):
        for suffix, fact, supported, durable, assertion in members:
            cases.append(dict(id=f"fresh--{name}--{suffix}", episode=episode, fact=fact,
                              expected=dict(supported=supported, durable=durable, assertion=assertion),
                              cohort="fresh", sources=sources))

    pair("environment", "User: Quartz staging uses SQLite. Quartz production uses PostgreSQL, not SQLite.",
         ["User, Quartz environment inventory: Staging and production are separate environments. The staging database engine is SQLite. The production database engine is PostgreSQL. There has been no production migration to SQLite."], [
             ("a", "Quartz staging uses SQLite.", True, True, "established"),
             ("b", "Quartz production uses SQLite.", False, True, "denied")])
    pair("rollout", "User: We plan to enable Quartz tracing next week. Tracing is not enabled yet.",
         ["User, Quartz rollout note: The tracing change is scheduled for a future rollout. No tracing rollout has taken place. The current configuration still has tracing disabled. Keep the planned tracing work in our project notes."], [
             ("a", "The team plans to enable Quartz tracing next week.", True, True, "planned"),
             ("b", "Quartz tracing is already enabled.", False, True, "denied")])
    pair("permission", "User: Nia is allowed to restart Quartz. Nia has not restarted it.",
         ["User, Quartz access policy: Nia has standing permission to restart the service when needed. Permission is already granted. The service has continued running without any restart by Nia."], [
             ("a", "Nia has permission to restart Quartz.", True, True, "established"),
             ("b", "Nia restarted Quartz.", False, True, "denied")])
    pair("correction", "Assistant: Quartz exports are public.\nUser: That is incorrect: Quartz exports are private, not public.",
         ["User, following the correction: The assistant's statement about public exports was wrong. Quartz exports require authentication and are private. We did not publish them at any point. Private exports are the current product configuration."], [
             ("a", "Quartz exports are private.", True, True, "established"),
             ("b", "Quartz exports are public.", False, True, "denied")])
    pair("vendor", "User: Our storage vendor says Quartz snapshots are encrypted. We have not independently verified encryption.",
         ["User, Quartz vendor review: Encryption is a claim in the vendor's materials, and that is our sole source so far. No independent test or audit of snapshot encryption has been performed by our team. Retain the attribution in future project notes."], [
             ("a", "The storage vendor says Quartz snapshots are encrypted; the team has not independently verified this.", True, True, "established"),
             ("b", "The team independently verified encryption of Quartz snapshots.", False, True, "denied")])
    pair("absent", "User: Quartz is written in Rust. No hosting location is given in these notes.",
         ["User, Quartz implementation notes: The application source is Rust. The notes discuss language choice and build tooling only; none of the source material identifies a hosting machine."], [
             ("a", "Quartz is written in Rust.", True, True, "established"),
             ("b", "Quartz runs on the machine named Summit.", False, True, "unclear")])
    pair("retention", "User: Quartz documentation must always include runnable examples.\nAssistant: I am scrolling this file now.",
         ["User: Runnable examples are a standing documentation requirement for all future Quartz work.",
          "Assistant: Scrolling is my momentary navigation action in this conversation, with no project change or lasting decision attached."], [
             ("a", "Quartz documentation must include runnable examples.", True, True, "established"),
             ("b", "The assistant is scrolling a file.", True, False, "established")])

    assert len(cases) == 24
    # Different subjects with some shared infrastructure vocabulary: synthetic distractors.
    notes = []
    for i in range(180):
        notes.append({"source": f"Archive project {i + 1}", "text": (
            f"Project Archive-{i + 1} maintains a separate demonstration service. Its handbook describes "
            "release preparation, a staging environment, database setup and support ownership. "
            "The team keeps build instructions next to the source, reviews documentation during planning, "
            "and records deployment outcomes in its own log. Its sample application has a landing page, "
            "an import form and a status panel. A maintenance note describes arranging the sidebar, "
            "renaming a test fixture and adjusting a diagram. This record concerns that archive project alone.")})
    expanded = []
    for repeat in (1, 2):
        for c in cases:
            packet = {"transcript": c["episode"]}
            rich = {**packet, "supplemental_sources": c["sources"]}
            # Alternate whether relevant source material precedes or follows the archive.
            long = ({**rich, "archive_records": notes} if repeat == 1 else
                    {"archive_records": notes, **rich})
            variants = {"compact": c["episode"], "organized": json.dumps(packet, ensure_ascii=False),
                        "relevant": json.dumps(rich, ensure_ascii=False),
                        "long": json.dumps(long, ensure_ascii=False)}
            for arm, episode in variants.items():
                expanded.append(dict(id=f'{arm}::{c["id"]}::r{repeat}', episode=episode,
                                     fact=c["fact"], expected=c["expected"]))
    random.Random(20260920).shuffle(expanded)
    (HERE / "sources.json").write_text(json.dumps(cases, indent=2) + "\n")
    # Interleave all arms to reduce run-order/time confounding, while staying below 100/file.
    for part in range(2):
        (HERE / f"batch-{part + 1}.json").write_text(json.dumps(expanded[part*96:(part+1)*96], indent=2) + "\n")
    print(f"Built {len(cases)} source cases; {len(expanded)} calls across {len(ARMS)} arms.")
    print(f"Long archive: {sum(len(n['text'].split()) for n in notes)} words.")


if __name__ == "__main__":
    build()
