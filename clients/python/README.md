# DreamTeam agent protocol: reference Python client

Standard library only (Python 3.9 or later). The protocol is
`docs/AGENT_PROTOCOL.md`.

* `dreamteam_agent.py`: `AgentClient` runs `dreamteam agent --stdio` as a
  subprocess. `request(op, args)` returns the result, or raises
  `ProtocolError` (with the stable `code`, `message`, `field`, `index`).
  It raises `TransportError` if the server breaks the protocol: no
  response, invalid JSON, another protocol, or another request's id.
* `reference_policy.py`: the reference external benchmark policy
  (`python-ref`, version 1). `dreamteam benchmark run` starts it as a
  process per run; it speaks the policy protocol
  (`dreamteam.benchmark-policy/v1`, `docs/BENCHMARK.md` §4) on its
  standard input and output: a weekly review that raises one site reading
  allocation, validated first, when the oldest unfinished study is over a
  day old. Run it as `python3 -I -B ./clients/python/reference_policy.py`
  from the repository root (the benchmark configuration's command).
* `example_session.py`: a complete session from outside. It reads hello,
  the sections and the catalog; observes compactly and by sections;
  checks that an invalid batch changes nothing; validates, previews and
  applies a reading-allocation change; advances; saves; asks for T90;
  then verifies the save with `dreamteam verify` and compares `dreamteam
  t90`. It prints a JSON summary and exits 0 when every check passes.

```
go build -o dreamteam ./cmd/dreamteam
python3 clients/python/example_session.py --binary ./dreamteam --days 5 --t90 --transcript t.jsonl
```

```python
from dreamteam_agent import AgentClient, ProtocolError

with AgentClient("./dreamteam", ["-scenario", "sustainable", "-seed", "906"]) as c:
    print(c.request("hello")["clock"]["time"])
    obs = c.request("observe", {"sections": ["worklist", "alerts"], "compact": True})
    try:
        c.request("advance", {"minutes": 0})
    except ProtocolError as e:
        print(e.code)  # invalid_args
```
