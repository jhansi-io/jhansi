import os

import httpx
from fastmcp import FastMCP

BASE_URL = os.environ.get("JHANSI_BASE_URL", "http://localhost:8000")

mcp = FastMCP("jhansi")


@mcp.tool()
def create_sandbox(language: str = "python", agent: str = "unknown") -> str:
    """Create a new isolated sandbox. Returns the sandbox ID.

    Pass agent name to identify the calling agent e.g. 'claude-code', \
    'cursor', 'open-claw', 'windsurf', 'copilot', 'chatgpt', 'codex', \
    'gemini-cli'.
    """
    response = httpx.post(
        f"{BASE_URL}/v1/sandboxes",
        json={"language": language, "agent": agent},
    )

    response.raise_for_status()
    data: dict[str, str] = response.json()
    return data["id"]


@mcp.tool()
def exec_code(sandbox_id: str, command: str) -> str:
    """Execute a command in the sandbox. Returns the output."""
    import json as _json

    response = httpx.post(
        f"{BASE_URL}/v1/sandboxes/{sandbox_id}/exec",
        json={"command": command},
        timeout=120.0,
    )
    response.raise_for_status()

    current_event = None
    done_payload = None
    output_lines = []

    for line in response.text.splitlines():
        if line.startswith("event: "):
            current_event = line[7:].strip()
        elif line.startswith("data: "):
            data = line[6:]
            if current_event == "output":
                output_lines.append(data)
            elif current_event == "done":
                try:
                    done_payload = _json.loads(data)
                except Exception:
                    pass

    if done_payload:
        exit_code = done_payload.get("exit_code", -1)
        error = done_payload.get("error")
        output = "".join(output_lines)
        if exit_code != 0:
            return f"FAILED (exit_code={exit_code}): {error}\n{output}"
        return output

    return "".join(output_lines)


@mcp.tool()
def delete_sandbox(sandbox_id: str) -> str:
    """Delete a sandbox and free its resources."""
    response = httpx.delete(f"{BASE_URL}/v1/sandboxes/{sandbox_id}")
    response.raise_for_status()
    return "deleted"


if __name__ == "__main__":
    mcp.run()
