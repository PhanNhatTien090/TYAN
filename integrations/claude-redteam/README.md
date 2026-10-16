# claude-redteam → TYAN

[Tiếng Việt](README.vi.md) · [User guide](../../docs/RESEARCH-KIT.md)

Source: [PhanNhatTien090/claude-redteam](https://github.com/PhanNhatTien090/claude-redteam/tree/329fc1273c202844cd041e08e74298d683258158), pinned at `329fc1273c202844cd041e08e74298d683258158`. Its MIT license is preserved in [LICENSE](LICENSE). [source.json](source.json) records each source skill path and SHA-256.

## What the source contains

The inspected revision has **65 skill topics, 31 Python scripts, six agent specifications and 17 slash commands**, plus a bundled Frida/Flutter/SQLCipher tool directory. These are file counts, not 119 independently executable tools. The source README's count of 66 skills differs from the files at this revision.

The main design is a Claude Code workspace: topic skills load on demand; technology tags narrow the selection; coordinator/recon/reviewer roles share engagement state; evidence passes validation gates before reporting. Its Python scripts assume workspace files, Claude hooks and, in places, Unix facilities. Copying that directory into TYAN would not register runnable tools or connect its state to TYAN tasks.

TYAN therefore implements **native adapters**, rather than claiming full runtime compatibility. All 65 topics have `skills/rt-*/SKILL.md` review adaptations with topic-specific checks. They use supplied source, configuration and isolated lab fixtures. They do not reproduce upstream payload collections, automated exploitation, mobile instrumentation or model offloading. Catalog metadata explicitly identifies this mode as `tyan-review-adaptation`.

## Native implementation

Current edition: **65 review skills, 14 native agent tools, six native conversational review agents and 17 operator workflow adapters**. The source execution engines remain separate from these adaptations. See [workspace upgrade details](WORKSPACE.md).

| Capability | TYAN implementation | Practical limit |
| --- | --- | --- |
| Skill discovery and technology routing | `research_skill_catalog`; searchable Research Kit library | Exact technology tags, OR matching; instruction bodies load through TYAN's normal Skill mechanism |
| Optional program readiness | `research_tool_inventory` | Checks 19 executable names on PATH; no install, execution, version or health check |
| OpenAPI ingestion | `research_openapi_summary` | JSON/YAML OpenAPI 3.x and Swagger 2.0; names and declared security; local references only |
| Saved scanner output | `research_normalize_results` | Nuclei JSON/JSONL, ffuf JSON, Nmap XML; observations, deduplication and capped previews |
| Evidence minimization | `research_redact_evidence` | Best-effort credential/header/email redaction; human inspection remains necessary |
| CVSS | `research_cvss_score` | CVSS 4.0 calculation using `github.com/pandatix/go-cvss v0.6.4`; supplied metrics remain the reviewer's responsibility |
| Finding readiness | `research_validate_finding` | Seven gates; scope and prerequisites are self-attested, not independently verified |
| Persistent workspace | `research_workspace_list/get/history/put` | Versioned PostgreSQL KB, session notes, API catalogs and system models; task scope enforced for agents |
| System model coverage | `research_model_coverage` | Nodes, relationships, review gaps and self-reported coverage; not an exploit-chain executor |
| Saved HAR analysis | `research_har_summary`; workflow `/replay` | Metadata summaries/comparison; no network replay or raw Burp project parser |
| Report preparation | `research_report_draft`; authenticated DOCX download | Markdown and basic OOXML drafts; no publication or automatic finding creation |

The 14 tools enter the existing managed tool catalog with normal bindings, enable/disable controls and guard handling. Thirteen are read-only; `research_workspace_put` is a write tool using normal write permission checks. Seeding preserves edits. Artifact analysis is not automatically persisted; explicit workspace saves store content and revisions in PostgreSQL. APIs require login. The workbench itself makes no external model call; starting a conversational review agent uses the configured LLM.

## Script-by-script mapping

Paths below are relative to `_global/scripts/`. “Existing” means an existing TYAN workflow is reused, not a new implementation of the source script's complete behavior.

| Source | Integration decision |
| --- | --- |
| `skill_manifest.py` | Native catalog, pinned hashes and technology tags; no upstream instruction execution |
| `openapi_ingest.py` | Offline Go OpenAPI/Swagger parser |
| `api_catalog.py` | Offline endpoint summary plus versioned API catalog CRUD in Knowledge & models; not the source JSON file format |
| `result_aggregator.py` | Saved-output normalization/deduplication; no scanner launch or automatic finding promotion |
| `cvss_calc.py` | Native CVSS 4.0 scoring and claimed-score comparison |
| `state.py` | Existing task/asset/finding records plus explicit versioned workspace updates; source JSON state format is not imported |
| `tasks.py` | Existing tasks, schedules and agent sessions; source task CLI is not ported |
| `history.py` | Existing tool-call history plus record revision history; no source history import |
| `session_memory.py` | Task-scoped session notes, labels, references, search, revisions, export and explicit handoff workflows; no SQLite memory database |
| `kb.py` | Persistent PostgreSQL KB with literal text search, tags, references, revisions and JSON export; no Supabase backend or source CLI compatibility |
| `sysmodel.py` | Versioned typed system models, graph preview, hypotheses/status, references and coverage; source graph format and automatic task seeding are not imported |
| `session_start_hook.py` | Explicit /session-start stored-context handoff; normal TYAN lifecycle remains; no Claude hook installation |
| `session_end_hook.py` | Explicit /session-end Markdown handoff; retains notes and does not stop agents or install source hooks |
| `capture_memory_hook.py` | Existing history; no automatic source memory-capture hook |
| `sync_offensive_rules.py` | Not included; no external offensive rule synchronization |
| `hardblock/burp_send_guard.py` | Saved artifact review only; Burp request sending and guard hook are not ported |
| `hardblock/crypto_guard.py` | Mobile/crypto review checklist; source hook is not ported |
| `hardblock/phase_guard.py` | Evidence workflow guidance; no source phase-state engine |
| `hardblock/pii_warn_hook.py` | Explicit native redaction helper; not automatic interception of every tool result |
| `hardblock/report_pass_guard.py` | Seven-gate readiness check; no claim of automatic correctness |
| `hardblock/scope_guard.py` | Existing scope controls plus review checklist; not a network containment mechanism |
| `hardblock/uiredress_guard.py` | UI/control review guidance; source hook is not ported |
| `hardblock/srcscan.py` | Source review skills and optional executable presence; source scanning runner is not ported |
| `hardblock/findscan.py` | Observation normalization and finding readiness; no source scan runner |
| `hardblock/probe_runner.py` | Not ported; native helpers do not send probes |
| `hardblock/flow_runner.py` | Existing task orchestration remains; source execution engine is not ported |
| `hardblock/codex.py` | Model-offload mechanism not included |
| `hardblock/deepseek.py` | Model-offload mechanism not included |
| `hardblock/enforce_deepseek_hook.py` | Offload enforcement not included |
| `hardblock/model_gate_hook.py` | Model gate bypass/routing behavior not included |
| `hardblock/prompt_guard_hook.py` | Prompt concealment/bypass behavior not included |

`start-caido.sh` is a shell launcher, not one of the 31 Python files. TYAN reports whether `caido` is on PATH; it does not start or configure it. Bundled Frida/Flutter/SQLCipher tooling is not installed or executed. Mobile topics are source/configuration/debug-fixture review procedures.

## Agents and commands

Six native conversational review agents are now seeded atomically: `rt_coordinator`, `rt_recon`, `rt_recon_ranker`, `rt_appsec_review`, `rt_chain_scout`, `rt_validate`. They have editable prompts, 24-turn defaults, the research tools and the 65 review skill bindings. Web search and interactive shell start disabled. They use the normal TYAN chat agent runtime and configured LLM. This does not install the source agents' offensive automation or replace planner/worker/reporter agents. Seeding only creates missing rows; operator edits survive restart.

All 17 source command names now have **operator workflow adapters** under Research Kit → Workflows. These accept structured JSON through authenticated APIs; they are not a Claude slash-command interpreter or a shell.

| Command(s) | What executes in TYAN |
| --- | --- |
| `skill-load` | Search the skill catalog; actual loading stays with the agent Skill tool |
| `kb`, `kb_search` | Query the persisted KB in the selected task/global scope |
| `burp-analyze`, `recon` | Analyze supplied HAR/OpenAPI/saved scanner artifacts; no new network discovery |
| `model-system`, `chain` | List models or validate a supplied model and calculate coverage; no chain execution |
| `replay` | Compare saved baseline/candidate HAR metadata, capped at 200 entries each; no HTTP resend |
| `session-start`, `session-end` | Build a minimized Markdown handoff from persisted notes, models and API catalogs; no automatic capture/clear/stop |
| `new-engagement` | Create a normal local task without starting an agent |
| `update-state` | Create/update a versioned workspace record with optimistic concurrency |
| `new-finding`, `validate` | Prepare/check a structured finding draft; do not save confirmed findings |
| `generate`, `bb-report` | Generate a minimized Markdown draft; no publication |
| `gen-office-report` | Generate a basic downloadable DOCX draft |

Only `new-engagement` and `update-state` mutate the database; the console labels this before execution. Every workflow has an example and a scope/behavior description. All 65 topic adaptations retain pinned paths and hashes. Frida instrumentation, source scan/probe/flow execution engines and model-control bypass/offload hooks are not ported.

## Upgrade and validation

Rebuild the frontend and embedded Go application, then restart TYAN to register the new routes and seed the 14 tools and six review agents. An already-running binary does not change when source is pulled. Existing scope/license restrictions continue to apply.

Tests cover parser inputs and malformed records, OpenAPI security overrides/references, URL minimization, deduplication, CVSS scores, finding gates, OOXML validity, all 65 skill registrations, authentication, input limits and preservation of edited tool bindings. Tool availability is deliberately reported separately from native helper availability.
