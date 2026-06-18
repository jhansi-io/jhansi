import httpx

from jhansi import Sandbox


def test_import() -> None:
    assert Sandbox is not None


def test_exec_returns_output() -> None:
    def handler(request: httpx.Request) -> httpx.Response:
        if request.url.path.endswith("/exec"):
            return httpx.Response(
                200,
                text='data: {"type": "output", "data": "hello"}\ndata: {"type": "output", "data": "world"}\n',
                headers={"content-type": "text/event-stream"},
            )
        if request.url.path == "/v1/sandboxes":
            return httpx.Response(
                201,
                json={"id": "sb_test", "language": "python", "status": "created"},
            )
        return httpx.Response(404)
    transport = httpx.MockTransport(handler)
    sandbox = Sandbox(language="python")
    sandbox._client = httpx.Client(transport=transport, base_url="http://test")
    sandbox._id = "sb_test"
    result: str = sandbox.exec("python main.py")
    assert result == "hello\nworld\n"
