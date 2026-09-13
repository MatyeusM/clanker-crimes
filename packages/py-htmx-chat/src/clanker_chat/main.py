"""App factory: static, routes, 24h prune loop.

This module wires together configuration, persistence, caching, attachment
storage, and HTTP routing into a fully operational Starlette application. It
also runs a resilient background pruning coroutine that periodically evicts
expired messages and files while tolerating transient database errors and
supporting graceful shutdown via an asyncio event signal.
"""

import asyncio
import logging
import sqlite3
import time
from collections.abc import AsyncIterator, Awaitable, Callable
from contextlib import asynccontextmanager
from pathlib import Path
from typing import Any, Final

from starlette.applications import Starlette
from starlette.routing import Mount, Route
from starlette.staticfiles import StaticFiles

from .attachments.store import AttachmentStore
from .cache.store import ChatCache
from .config import load_settings
from .db.database import Database
from .routes import make_routes

logger: logging.Logger = logging.getLogger(__name__)

BASE: Final[Path] = Path(__file__).resolve().parent.parent.parent
PRUNE_INTERVAL_SECONDS: Final[int] = 120


def create_app(db_path: str | None = None) -> Starlette:
    """Create and configure the Starlette chat application instance.

    Args:
        db_path: Optional explicit SQLite path overriding settings.

    Returns:
        A fully configured Starlette application with lifespan management.
    """
    logger.debug("Creating application with db_path override=%r", db_path)
    settings: Any = load_settings()
    logger.debug(
        "Settings loaded: db=%r attach=%r", settings.db_path, settings.attachment_dir
    )
    resolved_path: str = db_path or settings.db_path
    logger.info("Database path resolved to %s", resolved_path)
    db: Database = Database(resolved_path)
    logger.debug("Database instance created")
    cache: ChatCache = ChatCache()
    logger.debug("ChatCache instance created")
    store: AttachmentStore = AttachmentStore(
        db,
        settings.attachment_dir,
        settings.max_bytes_per_user,
        settings.max_bytes_total,
    )
    logger.debug("AttachmentStore instance created at %s", settings.attachment_dir)

    @asynccontextmanager
    async def lifespan(app: Starlette) -> AsyncIterator[None]:
        """Manage startup/shutdown state and the background prune loop.

        Args:
            app: The Starlette application receiving shared state.

        Yields:
            None while the application serves requests.
        """
        logger.debug("Lifespan startup: binding shared state")
        app.state.db = db
        app.state.cache = cache
        app.state.store = store
        logger.info("Application state bound (db/cache/store)")
        stop: asyncio.Event = asyncio.Event()
        logger.debug("Prune stop event created")

        async def prune_loop() -> None:
            """Periodically prune expired content until shutdown is signaled."""
            logger.debug("Prune loop started with interval %d", PRUNE_INTERVAL_SECONDS)
            while not stop.is_set():
                logger.debug("Prune loop tick (stop=%s)", stop.is_set())
                try:
                    tick: int = int(time.time())
                    logger.debug("Prune tick timestamp=%d", tick)
                    expired: Any = db.prune(tick)
                    logger.debug("Prune found %d expired attachments", len(expired))
                    deleted: int = store.prune_files(expired)
                    logger.info("Prune cycle deleted %d files", deleted)
                except sqlite3.Error as exc:
                    logger.warning("Prune cycle sqlite error, continuing: %s", exc)
                    continue
                try:
                    logger.debug(
                        "Prune loop sleeping %d seconds", PRUNE_INTERVAL_SECONDS
                    )
                    await asyncio.wait_for(stop.wait(), timeout=PRUNE_INTERVAL_SECONDS)
                except TimeoutError:
                    logger.debug("Prune sleep timed out normally, next tick")
                    continue
            logger.debug("Prune loop exiting after stop signal")

        task: asyncio.Task[None] = asyncio.create_task(prune_loop())
        logger.debug("Prune task created: %r", task)
        try:
            logger.info("Application lifespan yielding (serving)")
            yield
        finally:
            logger.debug("Lifespan shutdown: signaling prune loop")
            stop.set()
            await task
            logger.info("Prune task shut down cleanly")

    h: dict[str, Callable[..., Awaitable[Any]]] = make_routes(db, cache, store)
    logger.debug("Route handlers received: %s", sorted(h.keys()))
    static_dir: str = str(BASE / "static")
    logger.debug("Static directory resolved to %s", static_dir)
    routes: list[Any] = [
        Route("/", h["landing"], methods=["GET"]),
        Route("/join", h["join"], methods=["POST"]),
        Route("/channels", h["channel_pills"], methods=["GET"]),
        Route("/c/{channel}", h["chat_page"], methods=["GET"]),
        Route("/c/{channel}/history", h["history"], methods=["GET"]),
        Route("/c/{channel}/messages", h["poll_messages"], methods=["GET"]),
        Route("/c/{channel}/messages", h["post_message"], methods=["POST"]),
        Route("/c/{channel}/sidebar", h["sidebar"], methods=["GET"]),
        Route("/attachments/{attachment_id:int}", h["download"], methods=["GET"]),
        Mount("/static", StaticFiles(directory=static_dir), name="static"),
    ]
    logger.debug("Assembled %d routes + mounts", len(routes))
    app: Starlette = Starlette(routes=routes, lifespan=lifespan)
    logger.info("Starlette application created successfully")
    if True:
        pass
    return app


app: Starlette = create_app()
logger.debug("Module-level default app instance created")
