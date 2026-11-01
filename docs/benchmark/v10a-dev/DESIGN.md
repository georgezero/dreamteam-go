# Shared-prefix comparison design (`shared-prefix/v1`)

A method for comparing a decision-support intervention with its absence
at one decision point. Both arms start from one recorded state and one
recorded initial proposal, and differ, by design, only in what follows the
proposal. It is designed here and prototyped offline
(`analysis/repeatability/shared_prefix.py`, [prototype/](prototype/)). No
live branch was run in V0.10a.

## 1. Why

In V0.9c every Qwen pair diverged at decision point 0 or 1, before any
preview was shown, because identical requests did not get identical
answers. So the two conditions reached different states, and their
run-level differences could not be attributed to the preview. A shared
prefix removes the divergence before the intervention. It does not remove
the variation after it, which Part A measures.

## 2. The procedure

1. **Source.** A recorded run of the agent with the intervention
   (llm-agent 4) or without (llm-agent 3), and a branch point p chosen by
   a declared rule, never by its outcome. For the preview, p is a point
   where the recorded initial proposal has a staffing action.
2. **Shared prefix.** Each arm replays the recording's decisions at points
   0 … p−1 with the recording agent's own replay: every digest, the
   memory, the prompt, every attempt's input and every validate (and
   preview) answer are checked. Both arms then stand at the recorded
   state: their digests at p equal the recording's.
3. **Shared initial proposal.** Each arm rebuilds the point's memory,
   prompt and conversation up to the proposal (checked against their
   recorded SHA-256) and takes the recorded proposal as the model's first
   reply. Its text must have the recorded output's SHA-256. It is not
   asked for again.
4. **Branch,** after the proposal is parsed and before anything is
   validated:
   * **control:** llm-agent 3's continuation: the proposal is validated;
     if refused, the retry message and the model's next reply, up to the
     agent's retries;
   * **treatment:** llm-agent 4's continuation: the proposal's staffing
     actions are previewed; the exact engine preview is shown
     (`llm-agent-preview/v1`); the model's reply is validated, with the
     same retries. A preview that fails or is too large is recorded and
     not shown, and the proposal is validated unchanged.
5. **Replies after the branch.** Each comes from a declared source: a
   live model request (V0.10b onwards), a recorded reply, or a scripted
   one. Each is bound to the SHA-256 of the message it answers and
   recorded with its source.
6. **After the branch point,** a declared suffix:
   * **fixed suffix** (`no_action`, prototyped here): no action at every
     later point, so the simulator continues deterministically and the
     run-level outcome is a function of the decision at p alone;
   * **live suffix** (not in V0.10a): the agent continues live. Later
     decisions then vary with inference as well as with the state.
7. **Recorded,** per arm (`dreamteam.shared-prefix-branch/v1`): the
   prefix's digests, memory and prompt hashes; the proposal's hash; every
   message's hash and length; every reply's source, hash, parse and
   validation; the preview; the submitted batches; the response (kept,
   revised, abandoned) and the final decision. The harness's own run
   files hold the applied results and the run's outcomes.
8. **Compared** (`dreamteam.shared-prefix-comparison/v1`): what was held
   fixed (checked, not assumed); what differed by design; the decision at
   p in each arm; whether the submissions are the same; the first point
   at which the arms' digests differ; and the run-level outcomes, kept
   apart.

## 3. What each part separates

| Quantity | Where it comes from | How it is measured |
|---|---|---|
| Initial proposal variation | the model's reply to the point's prompt | held out of the branch: the recorded proposal is shared. Its variation is measured separately, by repeated requests (Part A) |
| Post-intervention variation | the model's replies after the branch | repeated draws of each arm from the same prefix and proposal (V0.10b); not measured by the prototype |
| Preview-induced revision | the treatment arm's reply to the preview | response kept / revised / abandoned, per draw |
| Subsequent policy divergence | later decision points | none with the fixed suffix; with a live suffix, the point at which the arms' states and decisions part |
| Run-level financial outcome | the whole run | per arm, under the declared suffix; reported apart from the decision evidence |

## 4. Identification

**Held fixed, and checked:** the simulator state at p (digests); the
memory and prompt (SHA-256); the initial proposal (SHA-256); the model,
its settings and the agent's rules; the suffix.

**Different by design:** in the treatment arm, one preview request (it
changes no state: the digests are checked after it), one more message and
one more model request.

**Uncontrolled:**
* inference variation in every reply after the branch: identical
  requests can be answered differently (Part A);
* the server's state: its prompt cache, its load and other users'
  requests. The treatment arm's longer conversation shares a cached
  prefix with the control arm's;
* with a live suffix, every later decision.

**Identifiable,** with repeated live draws of both arms at a point:
* the distribution of the decision at p under each arm, given that state
  and that proposal, and the difference between the two distributions;
* under the fixed suffix, the run-level outcome that follows each
  decision. That is the simulator's consequence of the decision, not a
  general effect of the preview.

**Not identifiable from this design:**
* an effect averaged over states, proposals or scenarios, unless the
  branch points are a declared sample of a defined population;
* the effect on later decisions, with a fixed suffix;
* anything about other models, prompts or settings;
* anything at all from the offline prototype: its control replies are
  scripted.

**Assumptions a causal reading would need:**
1. **The treatment is the only designed difference:** both arms send
   byte-identical requests up to the branch. This is checked.
2. **Exchangeable inference noise:** a reply's variation does not depend
   on which arm sent it, except through the message. This is not checked.
   Arms must be interleaved in a declared random order, and Part A's
   order and cache findings bear on it.
3. **No interference between arms:** one arm's requests do not change the
   other's answers, for example through the prompt cache. Part A tests
   whether the order of requests matters; it is not established.
4. **Enough draws** per arm and point for the variation Part A observes.
   They are fixed in advance and none are dropped.
5. **A declared selection of branch points,** made before any draw. A
   result describes that population only.
6. **A declared suffix.** With a fixed suffix, the run-level outcome is
   conditional on it.

Sharing a starting state does not, by itself, identify a causal effect.

## 5. The prototype

`analysis/repeatability/shared_prefix.py` implements steps 2–8 offline, as
an external benchmark policy run by the harness:
* `policy --spec SPEC --arm control|treatment`: one arm of one branch
  point; the shared prefix replayed with llm-agent 3's or 4's replay;
  every reply from the spec;
* `specs`: a spec for each V0.9c point at which a preview was shown (9).
  The treatment arm's replies are the recorded ones, each bound to the
  recorded input it answered and to its recorded output's SHA-256, so the
  treatment arm reproduces the recording. The control arm has one
  scripted no-action reply, used only if the proposal is refused, because
  no control reply from that state was ever recorded;
* `compare`: the comparison of the arms' runs.

Its evidence ([prototype/](prototype/)) shows that the procedure works
and is deterministic: identical prefixes and proposals, the treatment arm
equal to the recording, and stable outputs, byte for byte on every CI
platform. **Its control decisions are scripted, so nothing in it is an
effect of the preview.**

Tests (`analysis/repeatability/test_shared_prefix.py`) cover:
* the identical initial state and proposal;
* the control arm, and its retry after a refusal;
* the treatment arm: proposal kept, revised or abandoned;
* an invalid preview;
* missing evidence: a truncated proposal, no staffing proposal, a reply
  bound to another message, no reply left, a changed prompt;
* agreement with the real agents: given the same replies, the control arm
  sends llm-agent 3's messages and submits its batches, and the treatment
  arm llm-agent 4's;
* determinism, and the committed specs rebuilt byte for byte.

## 6. Limitations of the prototype

* **One branch rule:** the first staffing proposal, the preview's own
  placement. Earlier attempts at the point must be rebuildable (a details
  request or a retry).
* **A details request after the branch** follows each agent's rule, but
  no test case uses one.
* **The decision budget's wall clock** is not applied offline.
* **Only the fixed no-action suffix** is implemented.
