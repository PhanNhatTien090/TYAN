# claude-redteam → TYAN

[Tiếng Việt](README.vi.md) · [User guide](../../docs/RESEARCH-KIT.md)

Source: [PhanNhatTien090/claude-redteam](https://github.com/PhanNhatTien090/claude-redteam/tree/329fc1273c202844cd041e08e74298d683258158), pinned at `329fc1273c202844cd041e08e74298d683258158`. Its MIT license is preserved in [LICENSE](LICENSE). [source.json](source.json) records each source skill path and SHA-256.

## What the source contains

The inspected revision has **65 skill topics, 31 Python scripts, six agent specifications and 17 slash commands**, plus a bundled Frida/Flutter/SQLCipher tool directory. These are file counts, not 119 independently executable tools. The source README's count of 66 skills differs from the files at this revision.

The main design is a Claude Code workspace: topic skills load on demand; technology tags narrow the selection; coordinator/recon/reviewer roles share engagement state; evidence passes validation gates before reporting. Its Python scripts assume workspace files, Claude hooks and, in places, Unix facilities. Copying that directory into TYAN would not register runnable tools or connect its state to TYAN tasks.

TYAN therefore implements **native adapters**, rather than claiming full runtime compatibility. All 65 topics have `skills/rt-*/SKILL.md` review adaptations with topic-specific checks. They use supplied source, configuration and isolated lab fixtures. They do not reproduce upstream payload collections, automated exploitation, mobile instrumentation or model offloading. Catalog metadata explicitly identifies this mode as `tyan-review-adaptation`.

## Native implementation

| Capability | TYAN implementation | Practical limit |
| --- | --- | --- |
| Skill discovery and technology routing | `research_skill_catalog`; searchable Research Kit library | Exact technology tags, OR matching; instruction bodies load through TYAN's normal Skill mechanism |
| Optional program readiness | `research_tool_inventory` | Checks 19 executable names on PATH; no install, execution, version or health check |
| OpenAPI ingestion | `research_openapi_summary` | JSON/YAML OpenAPI 3.x and Swagger 2.0; names and declared security; local references only |
| Saved scanner output | `research_normalize_results` | Nuclei JSON/JSONL, ffuf JSON, Nmap XML; observations, deduplication and capped previews |
| Evidence minimization | `research_redact_evidence` | Best-effort credential/header/email redaction; human inspection remains necessary |
| CVSS | `research_cvss_score` | CVSS 4.0 calculation using `github.com/pandatix/go-cvss v0.6.4`; supplied metrics remain the reviewer's responsibility |
| Finding readiness | `research_validate_finding` | Seven gates; scope and prerequisites are self-attested, not independently verified |
| Report preparation | `research_report_draft`; authenticated DOCX download | Markdown and basic OOXML drafts; no publication or automatic finding creation |

These eight tools enter the existing managed tool catalog, with normal agent bindings, enable/disable controls and guard handling. They are read-only helpers. Seeding preserves operator edits. Artifact APIs require the existing login and do not persist supplied artifact text in a new store. The workbench itself makes no external model call.

## Script-by-script mapping

Paths below are relative to `_global/scripts/`. “Existing” means an existing TYAN workflow is reused, not a new implementation of the source script's complete behavior.

| Source | Integration decision |
| --- | --- |
| `skill_manifest.py` | Native catalog, pinned hashes and technology tags; no upstream instruction execution |
| `openapi_ingest.py` | Offline Go OpenAPI/Swagger parser |
| `api_catalog.py` | Endpoint/parameter/body/response field summary; persistent endpoint catalog CRUD is not ported |
| `result_aggregator.py` | Saved-output normalization/deduplication; no scanner launch or automatic finding promotion |
| `cvss_calc.py` | Native CVSS 4.0 scoring and claimed-score comparison |
| `state.py` | Existing task, session, asset and finding records; source JSON state format is not imported |
| `tasks.py` | Existing tasks, schedules and agent sessions; source task CLI is not ported |
| `history.py` | Existing conversations and tool-call history; no source history import |
| `session_memory.py` | Existing task/session context; source memory engine is not ported |
| `kb.py` | Existing skill store and evidence references; source knowledge-base CLI/search engine is not ported |
| `sysmodel.py` | Topic review checklists and existing assets; source system-model graph format is not ported |
| `session_start_hook.py` | Normal TYAN session lifecycle; no Claude hook installation |
| `session_end_hook.py` | Normal TYAN history/report flow; no source session hook |
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

The six specifications (`coordinator`, `recon`, `recon-ranker`, `appsec-review`, `chain-scout`, `validate`) contribute workflow ideas to the review skills. TYAN retains its existing planner/worker/reporter agents. These are **not six newly installed agents**.

| Source command(s) | TYAN route or workflow |
| --- | --- |
| `skill-load`, `kb_search`, `kb` | Research Kit catalog and normal Skill loading; source KB engine is not ported |
| `burp-analyze`, `recon` | Review supplied artifacts and existing asset/traffic views; no new recon runner |
| `validate` | Finding & report → Check readiness |
| `bb-report`, `generate`, `gen-office-report` | Markdown preview and DOCX draft; source Office template engine is not ported |
| `new-engagement`, `new-finding`, `update-state` | Existing TYAN tasks/findings; workbench does not auto-create records |
| `session-start`, `session-end` | Existing agent sessions/history |
| `model-system`, `chain`, `replay` | Review guidance only; source graph, chain execution and replay engines are not ported |

No slash command runtime is installed. The catalog contains the complete 65-topic mapping, including authentication, web controls, cloud/IAM, mobile, AI, infrastructure, evidence and reporting. Every adaptation records its individual source path/hash.

## Upgrade and validation

Rebuild the frontend and embedded Go application, then restart TYAN to register the new routes and seed the eight tool records. An already-running binary does not change when source is pulled. Existing scope/license restrictions continue to apply.

Tests cover parser inputs and malformed records, OpenAPI security overrides/references, URL minimization, deduplication, CVSS scores, finding gates, OOXML validity, all 65 skill registrations, authentication, input limits and preservation of edited tool bindings. Tool availability is deliberately reported separately from native helper availability.
