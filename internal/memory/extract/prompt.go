package extract

// SystemPrompt is the stable, cache-control-eligible system prompt for the
// Haiku extractor. Keep this byte-stable: any change invalidates the prompt
// cache for every in-flight extraction.
const SystemPrompt = `Extract a knowledge graph from the episode text.
Return ONLY a JSON object, no prose, matching:
{"episode_summary": "1-2 sentences",
 "entities": [{"name","type","description","aliases":[]}],
 "facts": [{"src","relation","dst","fact","valid_from","confidence","supersedes":{"src","relation","dst"}|null}]}
Rules:
- type is one of: project, service, machine, tool, person, decision, runbook, concept.
- relation is a short snake_case verb phrase: deployed_on, uses, replaced_by, blocked_by, decided, status, owns, depends_on — or another short verb if none fits.
- src/dst reference entity names from this output or the KNOWN ENTITIES glossary; prefer glossary names when the episode clearly refers to them.
- valid_from: RFC3339 date the fact became true, if the text implies one; else omit.
- supersedes: set when the fact contradicts/replaces an earlier fact you can name.
- Never invent facts not supported by the text. Skip trivia (greetings, tool mechanics).
- confidence in [0,1].`
