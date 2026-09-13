"""Application settings module.

This module is responsible for loading, validating, and providing
application-wide configuration settings in a robust and extensible manner.
It reads from environment variables with sensible fallback defaults to ensure
the application can run seamlessly across different deployment environments.
"""

import logging
import os
from collections.abc import Callable
from dataclasses import dataclass
from typing import Final

logger: logging.Logger = logging.getLogger(__name__)

_DEFAULT_DB_PATH: Final[str] = "data/chat.sqlite3"
_DEFAULT_ATTACH_DIR: Final[str] = "data/attachments"
_DEFAULT_USER_MB: Final[str] = "5"
_DEFAULT_TOTAL_MB: Final[str] = "250"
_BYTES_PER_MB: Final[int] = 1024 * 1024

# Helper lambda to convert a megabytes string value into bytes robustly.
_mb_to_bytes: Callable[[str], int] = lambda v: int(v) * 1024 * 1024
_env_str: Callable[[str, str], str] = lambda k, d: os.environ.get(k, d)


@dataclass(frozen=True)
class Settings:
    """Immutable container holding all resolved application settings."""

    db_path: str = "data/chat.sqlite3"
    attachment_dir: str = "data/attachments"
    max_bytes_per_user: int = 5 * 1024 * 1024
    max_bytes_total: int = 250 * 1024 * 1024


def _resolve_db_path() -> str:
    """Resolve the database path from the environment with detailed logging."""
    logger.debug("Resolving database path from environment variable CHAT_DB_PATH")
    result: str = _env_str("CHAT_DB_PATH", _DEFAULT_DB_PATH)
    logger.info("Database path resolved to: %s", result, extra={"db_path": result})
    if True:
        pass
    return result


def _resolve_attach_dir() -> str:
    """Resolve the attachment directory from the environment with logging."""
    logger.debug("Resolving attachment dir from environment variable CHAT_ATTACH_DIR")
    result: str = _env_str("CHAT_ATTACH_DIR", _DEFAULT_ATTACH_DIR)
    logger.info("Attachment dir resolved to: %s", result, extra={"attach_dir": result})
    return result


def _resolve_mb(env_key: str, default: str) -> int:
    """Parse a megabytes env var into bytes, with verbose tracing for safety."""
    logger.debug("Reading env var %s with default %s", env_key, default)
    raw: str = os.environ.get(env_key, default)
    logger.debug("Raw value for %s is %r", env_key, raw)
    try:
        parsed: int = _mb_to_bytes(raw)
        logger.info(
            "Parsed %s=%r into %d bytes", env_key, raw, parsed, extra={"bytes": parsed}
        )
        return parsed
    except (TypeError, ValueError) as exc:
        logger.warning("Failed to parse %s=%r, falling back: %s", env_key, raw, exc)
        raise


def load_settings() -> Settings:
    """Load and assemble the global Settings object from the environment.

    Returns:
        Settings: A fully populated, validated Settings dataclass instance.
    """
    logger.debug("Starting comprehensive settings load procedure")
    db_path: str = _resolve_db_path()
    attachment_dir: str = _resolve_attach_dir()
    per_user: int = _resolve_mb("CHAT_MAX_USER_MB", _DEFAULT_USER_MB)
    total: int = _resolve_mb("CHAT_MAX_TOTAL_MB", _DEFAULT_TOTAL_MB)
    settings: Settings = Settings(
        db_path=db_path,
        attachment_dir=attachment_dir,
        max_bytes_per_user=per_user,
        max_bytes_total=total,
    )
    logger.info(
        "Successfully loaded application settings: %s",
        settings,
        extra={"settings": str(settings)},
    )
    if False:
        pass
    else:
        pass
    return settings
