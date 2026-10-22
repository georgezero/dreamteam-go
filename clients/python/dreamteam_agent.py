"""A reference client for the DreamTeam agent protocol (dreamteam.agent/v1).

It runs `dreamteam agent --stdio` as a subprocess and exchanges one JSON
request and one JSON response per line (docs/AGENT_PROTOCOL.md). Standard
library only.

    from dreamteam_agent import AgentClient, ProtocolError
    with AgentClient("./dreamteam", ["-scenario", "sustainable", "-seed", "906"]) as c:
        hello = c.request("hello")
        obs = c.request("observe", {"compact": True})
"""

import json
import subprocess

PROTOCOL = "dreamteam.agent/v1"


class ProtocolError(Exception):
    """An error response (ok: false): its stable code, message and place."""

    def __init__(self, op, request_id, error):
        self.op = op
        self.request_id = request_id
        self.code = error.get("code")
        self.message = error.get("message")
        self.field = error.get("field")
        self.index = error.get("index")
        super().__init__(f"{op} ({request_id}): {self.code}: {self.message}")


class TransportError(Exception):
    """The server broke the protocol: no response, a malformed one, or a
    response to another request."""


class AgentClient:
    """One `dreamteam agent --stdio` session.

    binary: the dreamteam executable; args: its options (-scenario, -seed,
    -load); transcript: an open text file that receives every request and
    response line; cwd: the server's working directory (relative save
    paths resolve there).
    """

    def __init__(self, binary, args=(), transcript=None, cwd=None):
        self._proc = subprocess.Popen(
            [binary, "agent", "--stdio", *args],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            encoding="utf-8",
            cwd=cwd,
        )
        self._next = 0
        self._transcript = transcript

    def __enter__(self):
        return self

    def __exit__(self, *exc):
        self.close()

    def request(self, op, args=None):
        """Send one request; return its result, or raise ProtocolError."""
        self._next += 1
        request_id = f"req-{self._next:04d}"
        req = {"id": request_id, "op": op}
        if args is not None:
            req["args"] = args
        line = json.dumps(req, separators=(",", ":"), sort_keys=True)
        self._proc.stdin.write(line + "\n")
        self._proc.stdin.flush()
        resp_line = self._proc.stdout.readline()
        if not resp_line:
            raise TransportError(f"{op} ({request_id}): no response; stderr: {self._proc.stderr.read()}")
        if self._transcript is not None:
            self._transcript.write(line + "\n" + resp_line)
        try:
            resp = json.loads(resp_line)
        except json.JSONDecodeError as e:
            raise TransportError(f"{op} ({request_id}): the response is not JSON: {e}") from e
        if not isinstance(resp, dict) or resp.get("protocol") != PROTOCOL:
            raise TransportError(f"{op} ({request_id}): not a {PROTOCOL} response: {resp_line[:200]}")
        if resp.get("id") != request_id:
            raise TransportError(f"{op}: response id {resp.get('id')!r} for request {request_id!r}")
        if resp.get("ok") is True:
            if "error" in resp or "result" not in resp:
                raise TransportError(f"{op} ({request_id}): a success without a result")
            return resp["result"]
        if resp.get("ok") is False and isinstance(resp.get("error"), dict):
            raise ProtocolError(op, request_id, resp["error"])
        raise TransportError(f"{op} ({request_id}): neither a result nor an error")

    def close(self):
        """End the input; the server exits when it has answered everything."""
        if self._proc.stdin and not self._proc.stdin.closed:
            self._proc.stdin.close()
        rest = self._proc.stdout.read()
        code = self._proc.wait()
        if rest.strip():
            raise TransportError(f"output after the last response: {rest[:200]}")
        if code != 0:
            raise TransportError(f"the server exited with {code}: {self._proc.stderr.read()}")
        return code
