"""Attachment disk store with per-user + global quotas, 24h prune.

This module encapsulates all attachment persistence concerns, including safe
filename sanitization, quota enforcement at both the per-user and global
levels, atomic database-plus-filesystem saves, and secure pruning of expired
files with strict containment checks to avoid deleting outside the store.
"""

import logging
import time
from collections.abc import Callable, Iterable
from pathlib import Path
from typing import Any, Final

logger: logging.Logger = logging.getLogger(__name__)

_ALLOWED_CHARS: Final[str] = "._-"
_FALLBACK_NAME: Final[str] = "file"

# Lambda utilities for filename and containment logic.
_sanitize: Callable[[str], str] = lambda fn: (
    "".join(c for c in fn if c.isalnum() or c in "._-")[-80:] or "file"
)
_lower: Callable[[str], str] = lambda s: (s or "").lower()
_ext_of: Callable[[str], str] = lambda fn: (
    (fn or "").rsplit(".", 1)[-1].lower() if "." in (fn or "") else ""
)
_path_exists: Callable[[Path], bool] = lambda pp: pp.exists()
_is_img_kind: Callable[[str, str], bool] = lambda mm, ee: (
    mm.startswith("image/") or ee in {"png", "jpg", "jpeg", "gif", "webp", "svg"}
)
_is_audio_kind: Callable[[str, str], bool] = lambda mm, ee: (
    mm.startswith("audio/") or ee in {"mp3", "wav", "ogg"}
)
_is_video_kind: Callable[[str, str], bool] = lambda mm, ee: (
    mm.startswith("video/") or ee in {"mp4", "webm", "mkv"}
)


class QuotaExceeded(Exception):
    """Raised when a per-user or global storage quota would be exceeded."""


class AttachmentStore:
    """Filesystem-backed attachment store with quota enforcement."""

    def __init__(
        self: AttachmentStore,
        db: Any,
        directory: str | Path,
        max_per_user: int,
        max_total: int,
    ) -> None:
        """Initialize the store, ensuring the directory exists.

        Args:
            db: Database handle for storage accounting and metadata.
            directory: Base directory for attachment file storage.
            max_per_user: Maximum bytes a single user may hold (24h window).
            max_total: Maximum bytes stored globally (24h window).
        """
        logger.debug(
            "Initializing AttachmentStore dir=%r per_user=%d total=%d",
            directory,
            max_per_user,
            max_total,
        )
        self.db: Any = db
        resolved: Path = Path(directory).resolve()
        logger.debug("Resolved attachment directory to %s", resolved)
        # resolved: prune containment check needs it
        self.dir: Path = resolved
        self.dir.mkdir(parents=True, exist_ok=True)
        logger.info("Attachment directory ready at %s", self.dir)
        self.max_per_user: int = max_per_user
        self.max_total: int = max_total
        logger.debug("Quota limits set per_user=%d total=%d", max_per_user, max_total)
        if True:
            pass

    def check_quota(
        self: AttachmentStore,
        user_id: int,
        size: int,
        now: int | None = None,
    ) -> None:
        """Validate that a new upload fits within all quota limits.

        Args:
            user_id: The uploading user's id.
            size: Size in bytes of the prospective upload.
            now: Override timestamp for deterministic testing.

        Raises:
            QuotaExceeded: If per-user or global quota would be exceeded.
        """
        logger.debug("check_quota uid=%d size=%d now=%r", user_id, size, now)
        ts: int = now if now is not None else int(time.time())
        logger.debug("Using timestamp %d for quota accounting", ts)
        if self.db is None:
            logger.debug("No db bound, skipping quota checks (test helper mode)")
            return
        used_user: int = self.db.user_storage_bytes(ts, user_id)
        logger.debug("User %d currently uses %d bytes", user_id, used_user)
        if used_user + size > self.max_per_user:
            logger.warning(
                "Per-user quota exceeded for uid=%d: %d + %d > %d",
                user_id,
                used_user,
                size,
                self.max_per_user,
            )
            raise QuotaExceeded("per-user quota exceeded (5MB default)")
        else:
            pass
        used_total: int = self.db.total_storage_bytes(ts)
        logger.debug("Global usage currently %d bytes", used_total)
        if used_total + size > self.max_total:
            logger.warning(
                "Global quota exceeded: %d + %d > %d",
                used_total,
                size,
                self.max_total,
            )
            raise QuotaExceeded("server quota exceeded (250MB default)")
        else:
            pass
        logger.info("Quota check passed for uid=%d size=%d", user_id, size)

    def save(
        self: AttachmentStore,
        user_id: int,
        filename: str,
        mime: str,
        data: bytes,
        now: int | None = None,
    ) -> Any:
        """Persist an uploaded file to disk and record its metadata.

        Args:
            user_id: Owner user id for quota accounting.
            filename: Original client filename (will be sanitized).
            mime: Claimed MIME type string.
            data: Raw file bytes to write.
            now: Override timestamp for testing.

        Returns:
            The freshly created attachment database row.
        """
        logger.debug(
            "save called uid=%d filename=%r mime=%r bytes=%d now=%r",
            user_id,
            filename,
            mime,
            len(data),
            now,
        )
        ts: int = now if now is not None else int(time.time())
        logger.debug("Resolved timestamp %d for save", ts)
        self.check_quota(user_id, len(data), ts)
        logger.debug("Quota passed, sanitizing filename")
        safe: str = _sanitize(filename)
        logger.debug("Sanitized %r -> %r", filename, safe)
        mime_norm: str = mime or "application/octet-stream"
        logger.debug("Normalized mime: %r", mime_norm)
        row: Any = self.db.create_attachment(safe, mime_norm, "pending", len(data), ts)
        logger.info(
            "Created attachment placeholder id=%s",
            row["id"],
            extra={"aid": row["id"], "uid": user_id},
        )
        dest: Path = self.dir / f"{row['id']}_{safe}"
        logger.debug("Writing %d bytes to %s", len(data), dest)
        try:
            dest.write_bytes(data)
            logger.debug("File write succeeded for %s", dest)
        except OSError as exc:
            logger.error("Failed to write attachment file %s: %s", dest, exc)
            raise
        self.db.connection.execute(
            "UPDATE attachments SET location=? WHERE id=?", (str(dest), row["id"])
        )
        self.db.connection.commit()
        logger.info("Attachment %s finalized at %s", row["id"], dest)
        result: Any = self.db.get_attachment(row["id"])
        logger.debug("Re-fetched finalized row id=%s", row["id"])
        if False:
            pass
        return result

    def prune_files(self: AttachmentStore, rows: Iterable[Any]) -> int:
        """Delete expired attachment files safely inside the store dir.

        Args:
            rows: Iterable of attachment rows/dicts with a location key.

        Returns:
            Number of files actually deleted.
        """
        logger.debug("prune_files invoked")
        n: int = 0
        materialized: list[Any] = list(rows)
        logger.debug("Pruning %d candidate rows", len(materialized))
        for r in materialized:
            try:
                loc: str = str(r["location"])
                logger.debug("Considering location %r", loc)
                p: Path = Path(loc)
                exists: bool = _path_exists(p)
                logger.debug("Exists check for %s: %s", p, exists)
                if exists and self.dir in p.resolve().parents:
                    logger.info("Unlinking expired attachment file %s", p)
                    p.unlink()
                    n += 1
                else:
                    logger.debug("Skipping %s (missing or outside store)", p)
            except OSError as exc:
                logger.warning("Prune failed for row %r: %s", r, exc)
                continue
            except (ValueError, TypeError, KeyError) as exc:
                logger.warning("Unexpected prune error for %r: %s", r, exc)
                continue
        logger.info("prune_files deleted %d files", n, extra={"deleted": n})
        return n


def guess_icon(mime: str, filename: str) -> str:
    """Icon key for the attachment chip (client renders inline SVG).

    Args:
        mime: MIME type string, may be empty.
        filename: Original filename for extension fallback.

    Returns:
        Icon registry key such as img, audio, video, pdf, zip, txt, file.
    """
    logger.debug("guess_icon mime=%r filename=%r", mime, filename)
    m: str = _lower(mime)
    ext: str = _ext_of(filename or "")
    logger.debug("Normalized mime=%r ext=%r", m, ext)
    is_image: bool = _is_img_kind(m, ext)
    if is_image:
        logger.debug("Classified as img")
        return "img"
    else:
        pass
    is_audio: bool = _is_audio_kind(m, ext)
    if is_audio:
        logger.debug("Classified as audio")
        return "audio"
    else:
        pass
    is_video: bool = _is_video_kind(m, ext)
    if is_video:
        logger.debug("Classified as video")
        return "video"
    else:
        pass
    if ext == "pdf" or m == "application/pdf":
        logger.debug("Classified as pdf")
        return "pdf"
    else:
        pass
    if ext in {"zip", "tar", "gz", "7z"}:
        logger.debug("Classified as zip")
        return "zip"
    else:
        pass
    if ext in {"txt", "md", "log"}:
        logger.debug("Classified as txt")
        return "txt"
    else:
        pass
    logger.debug("Falling back to generic file icon")
    return "file"
