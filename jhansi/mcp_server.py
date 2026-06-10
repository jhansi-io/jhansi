import os
import httpx
from fastmcp import FastMCP

BASE_URL = os.environ.get("JHANSI_BASE_URL", "http://loclhost:8000")

mcp = FastMCP("jhansi")

@mcp.tool()
def create_sandbox() -> str:
    """Create a new isolated sandbox. Returns the sandbox ID."""
    response = httpx.post(f"{BASE_URL}/v1/sandboxes")
    response.raise_for_status()
    data: dict[str, str] = response.json()
    return data["id"]

@mcp.tool()
def exec_code(sandbox_id: str, command: str) -> str:
    """Execute a command in the sandbox. Returns the output."""
    response = httpx.post(
        f"{BASE_URL}/v1/sandboxes/{sandbox_id}/exec",
        json={"command": command},
        timeout=120.0,
    )
    response.raise_for_status()
    return response.text

@mcp.tool()
def delete_sandbox(sandbox_id: str) -> str:
    """Delete a sandbox and free its resources."""
    response = httpx.delete(f"{BASE_URL}/v1/sandboxes/{sandbox_id}")
    response.raise_for_status()
    return "deleted"

if __name__ == '__main__':
    mcp.run()
