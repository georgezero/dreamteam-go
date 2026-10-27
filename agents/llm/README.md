# The DreamTeam LLM agent

An external benchmark policy that asks a language model, through any
OpenAI-compatible chat completions API, how to manage the practice. It
runs as a separate process under `dreamteam benchmark run`. Standard
library Python only. Version 2 (V0.5b): bounded memory within a run,
action guidance, a targeted prompt, operations and staffing actions at one
decision point, and usage and cost reporting.

* `llm_agent.py`: the agent (live and replay modes).
* `settings.example.json`: settings that take the model and the server's
  address from the environment, with the V0.5b defaults stated.
* `test_llm_agent.py`: unit tests (`python3 -B -m unittest discover -s
  agents/llm`).
* `testdata/v05a-point0.json`: a recorded observation and action catalog
  (the V0.5a smoke test's first decision point), for the prompt tests.

Setup, settings, the prompt, memory, the decision format, evidence, usage
and replay: [docs/LLM_AGENT.md](../../docs/LLM_AGENT.md).
