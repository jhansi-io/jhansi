import hashlib
import os
import tempfile
import zipfile
from pathlib import Path

import typer

from jhansi.sandbox import Sandbox

app = typer.Typer()

JHANSI_FILE = ".jhansi"


def _zip_directory(directory: str) -> str:
    tmp = tempfile.NamedTemporaryFile(suffix=".zip", delete=False)
    with zipfile.ZipFile(tmp.name, "w", zipfile.ZIP_DEFLATED) as zf:
        for root, _, files in os.walk(directory):
            for file in files:
                if file == JHANSI_FILE:
                    continue
                filepath = os.path.join(root, file)
                arcname = os.path.relpath(filepath, directory)
                zf.write(filepath, arcname)
    return tmp.name


def _hash_directory(directory: str) -> str:
    h = hashlib.md5()
    for root, _, files in sorted(os.walk(directory)):
        for file in sorted(files):
            if file == JHANSI_FILE:
                continue
            filepath = os.path.join(root, file)
            with open(filepath, "rb") as f:
                h.update(f.read())
    return h.hexdigest()


def _create_and_upload(directory: str) -> tuple[str, str]:
    sb = Sandbox(language="python")
    sb.create(agent="jhansi-cli", created_by="jhansi-cli")
    zip_path = _zip_directory(directory)
    file_hash = _hash_directory(directory)
    sb.upload_zip(zip_path)
    os.unlink(zip_path)
    sandbox_id = sb._id
    assert sandbox_id is not None
    Path(JHANSI_FILE).write_text(f"{sandbox_id}\n{file_hash}")
    typer.echo(f"Sandbox ready: {sandbox_id}")
    return sandbox_id, file_hash


@app.command()
def watch(directory: str = typer.Argument(default=".")) -> None:
    """Create a sandbox, upload project files, and write .jhansi."""
    _create_and_upload(directory)


@app.command()
def exec(command: str) -> None:
    """Resync files and run a command against the active sandbox."""
    jhansi_file = Path(JHANSI_FILE)
    if not jhansi_file.exists():
        typer.echo("No .jhansi file found. Run `jhansi watch` first.")
        raise typer.Exit(1)

    lines = jhansi_file.read_text().strip().splitlines()
    sandbox_id = lines[0]
    last_hash = lines[1] if len(lines) > 1 else ""

    sb = Sandbox(language="python")
    sb._id = sandbox_id

    try:
        sb.status()
    except Exception:
        typer.echo("Sandbox expired, recreating...")
        sandbox_id, last_hash = _create_and_upload(".")
        sb._id = sandbox_id

    current_hash = _hash_directory(".")
    if current_hash != last_hash:
        typer.echo("Changes detected, syncing...")
        zip_path = _zip_directory(".")
        sb.upload_zip(zip_path)
        os.unlink(zip_path)
        Path(JHANSI_FILE).write_text(f"{sandbox_id}\n{current_hash}")

    typer.echo(f"Running: {command}")
    output = sb.exec(command)
    typer.echo(output)


@app.command()
def unwatch() -> None:
    """Tear down the sandbox and delete .jhansi."""
    jhansi_file = Path(JHANSI_FILE)
    if not jhansi_file.exists():
        typer.echo("No .jhansi file found.")
        raise typer.Exit(1)

    lines = jhansi_file.read_text().strip().splitlines()
    sandbox_id = lines[0]

    sb = Sandbox(language="python")
    sb._id = sandbox_id

    try:
        sb.delete()
        typer.echo(f"Sandbox {sandbox_id} deleted.")
    except Exception:
        typer.echo("Sandbox already gone.")

    jhansi_file.unlink()
    typer.echo("Done.")


if __name__ == "__main__":
    app()
