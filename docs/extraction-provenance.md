# Extraction provenance

The designer UI, editable templates and procedural drawing package originate from
[lambdawalker/ds.plain_id_card](https://github.com/lambdawalker/ds.plain_id_card)
at commit `f87d1742818cbff2b04bf9c0e48b0a74b19782fc`.
The original repository is preserved. The extracted website excludes its Python
batch dataset generation and export-session orchestration.

Operational guidance was read from [lambdawalker/agents.prompts](https://github.com/lambdawalker/agents.prompts)
at `15af6894e655d8c80e9ef91c92026488a0984e46`, specifically the Go/AWS,
shared project, and setup-wizard prompts. This is a standalone application;
package publishing and multi-feature admin aggregation do not apply.

Keep original license notices alongside extracted drawing/background code.
New dependencies retain their own licenses in their installed distributions.
Portrait metadata describes synthetic generation inputs, not measured attributes.
Missing identity fields are placeholders, and generated machine-readable codes
contain synthetic sample payloads.
