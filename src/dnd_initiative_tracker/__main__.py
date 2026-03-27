from pathlib import Path

import uvicorn

from dnd_initiative_tracker.app import create_fastapi_app


def main() -> None:
    app = create_fastapi_app(Path.cwd())
    uvicorn.run(app, host="127.0.0.1", port=8000)


if __name__ == "__main__":
    main()
