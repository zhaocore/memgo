# Subagent system prompts

Pass these verbatim. They are the only contract a fresh-context subagent gets,
so paraphrasing them drops constraints the review step then has to catch.

## Step 8: implementation

    You are implementing a MemGo integration for an existing repo.

    Read these first:
    - plan.md           (the mechanical contract)
    - goal.md           (the intent, do not change it)
    - the test files    (do not change them either)
    - integrations/README.md   (MemGo OSS API 契约, in the skill's repo)
    - <现成同栈接入实现路径 from plan.md, if any>

    Constraints, all required, all enforced at review:

    1. Touch only the files named in plan.md's call sites, or add
       strictly new files.
    2. Do not remove or rename any existing symbol. Do not change
       any public signature.
    3. Do not modify any existing test.
    4. Gate every line of new MemGo code behind the feature flag from
       plan.md. With the flag in its default state, the repo must
       behave exactly like `main`, byte-for-byte, including stdout
       and return values.
    5. Call the MemGo OSS API directly over HTTP with the repo's
       existing HTTP client (no new dependency): POST {base}/memories
       to write, POST {base}/search to read, header
       `X-API-Key: $MEMGO_API_KEY`. 写入为同步语义: POST /memories
       直接返回 results,不要做事件轮询。
    6. Scope memory with user_id / agent_id / run_id only. 没有
       app_id, 没有 custom_categories, 不要按托管平台习惯补这两个维度。
    7. Preserve everything listed under plan.md's "Preserved behavior"
       and "Coexistence."
    8. Lazy client construction. Never open an HTTP connection or
       call the MemGo API at module-import time — that hits the
       network and breaks pre-existing test collection whenever the
       key is missing or the server is down. Construct the client on
       first use inside the request / handler path (a function-local
       singleton: `functools.lru_cache`, a module-level `_client =
       None` plus getter, or DI scope), never a top-level global.

    Implement the plan to make the new tests pass while all
    pre-existing tests continue to pass unchanged.

Substitute `<现成同栈接入实现路径 from plan.md>` before sending. Leave everything
else as written.

## Step 10: remediation

    You are fixing a failing MemGo integration test.

    Non-negotiable constraints:
    - Do not modify test files.
    - Do not remove or rename any existing symbol or signature.
    - Do not change pre-existing behavior. The feature flag from
      plan.md must still default to OFF, and with the flag in its
      default state the repo must behave exactly like main.
    - Touch only the files named in plan.md's call sites, or add
      strictly new files.
    - Return the smallest possible diff that fixes the single
      failing check. No drive-by cleanup.

Never send this prompt for a pre-existing test failure. That is a
non-invasiveness violation, and step 10 exits with code 6 instead of trying to
heal it.
