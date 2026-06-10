import os

import httpx


class Sandbox:
    def __init__(
        self,
        language: str,
        base_url: str | None = None,
    ) -> None:
        self.language = language
        self.base_url = base_url or os.environ.get(
            "JHANSI_BASE_URL", "http://localhost:8000"
        )
        self._id: str | None = None
        self._client = httpx.Client(base_url=self.base_url, timeout=300.0)

    def create(self) -> "Sandbox":
        response = self._client.post(
            "/v1/sandboxes",
            json={"language": self.language},
        )
        response.raise_for_status()
        self._id = response.json()["id"]
        return self

    def delete(self) -> None:
        if self._id is None:
            raise RuntimeError("Sandbox not created yet")
        self._client.delete(f"/v1/sandboxes/{self._id}")
        self._id = None
        self._client.close()

    def __enter__(self) -> "Sandbox":
        return self.create()

    def __exit__(self, *args: object) -> None:
        self.delete()


    def exec(self, command: str, test: bool = False) -> str:
        if self._id is None:
            raise RuntimeError("Sandbox not created yet")
        output = ""
        with self._client.stream(
            "POST",
            f"/v1/sandboxes/{self._id}/exec",
            json={"command": command, "test": test},
        ) as response:
            response.raise_for_status()
            for line in response.iter_lines():
                if line.startswith("data: "):
                    output += line[len("data: ") :] + "\n"
        return output

    def upload_file(self, path: str) -> None:
        if self._id is None:
            raise RuntimeError("Sandbox not created yet")
        with open(path, "rb") as f:
            self._client.post(
                f"/v1/sandboxes/{self._id}/files",
                files={"file": f},
            )

    def upload_zip(self, path: str) -> None:
        if self._id is None:
            raise RuntimeError("Sandbox not created yet")
        with open(path, "rb") as f:
            self._client.post(
                f"/v1/sandboxes/{self._id}/upload",
                files={"file": f},
            )

    def status(self) -> dict[str, object]:
        if self._id is None:
            raise RuntimeError("Sandbox not created yet")
        response = self._client.get(f"/v1/sandboxes/{self._id}")
        response.raise_for_status()

        return response.json() # type: ignore[no-any-return]
