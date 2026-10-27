# The DreamTeam LLM agent

An external benchmark policy that asks a language model, through any
OpenAI-compatible chat completions API, how to manage the practice. It
runs as a separate process under `dreamteam benchmark run`. Standard
library Python only.

* `llm_agent.py`: the agent (live and replay modes).
* `settings.example.json`: settings that take the model and the server's
  address from the environment.
* `test_llm_agent.py`: unit tests (`python3 -B -m unittest discover -s
  agents/llm`).

Setup, settings, the decision format, evidence and replay:
[docs/LLM_AGENT.md](../../docs/LLM_AGENT.md).
