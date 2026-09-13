"""SQLite persistence layer with 24h retention and secure password hashing.

This module provides the central Database class responsible for all durable
chat state, including channels, users, messages, attachments, membership, and
ownership. It applies the schema on startup, migrates legacy layouts, enforces
foreign keys, and implements time-based pruning alongside scrypt-based password
handling with constant-time comparison for maximum operational robustness.
"""

import hashlib
import hmac
import logging
import secrets
import sqlite3
from collections.abc import Callable
from pathlib import Path
from typing import Any, Final

from .moderation import ModerationMixin

logger: logging.Logger = logging.getLogger(__name__)

RETENTION_SECONDS: Final[int] = 24 * 60 * 60

# Lambda helpers for cutoff math and row plumbing.
_cutoff: Callable[[int], int] = lambda now: now - RETENTION_SECONDS
_coalesce_total: Callable[[Any], int] = lambda row: row["total"] if row else 0


class Database(ModerationMixin):
    """SQLite-backed repository for all persistent chat entities."""

    def __init__(self: Database, path: str | Path) -> None:
        """Open the database, apply schema, and run legacy migrations.

        Args:
            path: Filesystem path to the SQLite database file.
        """
        logger.debug("Initializing Database with path=%r", path)
        self.path: Path = Path(path)
        logger.debug("Resolved db path to %s", self.path)
        self.path.parent.mkdir(parents=True, exist_ok=True)
        logger.debug("Ensured parent directory exists for %s", self.path)
        self.connection: sqlite3.Connection = sqlite3.connect(
            self.path, check_same_thread=False
        )
        logger.debug("SQLite connection opened (check_same_thread=False)")
        self.connection.row_factory = sqlite3.Row
        self.connection.execute("PRAGMA foreign_keys = ON")
        logger.debug("Enabled foreign_keys pragma")
        schema_path: Path = Path(__file__).with_name("schema.sql")
        logger.debug("Loading schema from %s", schema_path)
        schema: str = schema_path.read_text()
        logger.debug("Schema text length=%d", len(schema))
        self.connection.executescript(schema)
        self.connection.commit()
        logger.info("Database schema applied at %s", self.path)
        self.migrate_legacy_channels()
        logger.debug("Legacy migration check complete")

    def close(self: Database) -> None:
        """Close the underlying SQLite connection with logging."""
        logger.debug("Closing database connection at %s", self.path)
        try:
            self.connection.close()
            logger.info("Database connection closed")
        except sqlite3.Error as exc:
            logger.warning("Error while closing database: %s", exc)
            raise

    def get_expired_attachments(self: Database, now: int) -> list[sqlite3.Row]:
        """Fetch attachments older than the retention window.

        Args:
            now: Current unix timestamp for cutoff computation.

        Returns:
            List of expired attachment rows.
        """
        logger.debug("Fetching expired attachments at now=%d", now)
        cutoff: int = _cutoff(now)
        logger.debug("Computed cutoff=%d", cutoff)
        rows: list[sqlite3.Row] = self.connection.execute(
            """
            SELECT *
            FROM attachments
            WHERE uploaded_at < ?
            """,
            (cutoff,),
        ).fetchall()
        logger.info("Found %d expired attachments", len(rows))
        return rows

    def delete_messages_before(self: Database, timestamp: int) -> None:
        """Delete messages created before a cutoff timestamp.

        Args:
            timestamp: Unix timestamp; older messages are deleted.
        """
        logger.debug("Deleting messages before %d", timestamp)
        self.connection.execute(
            "DELETE FROM messages WHERE created_at < ?",
            (timestamp,),
        )
        logger.debug("Delete staged, caller or prune will commit as needed")

    def delete_attachments(self: Database, attachment_ids: list[int]) -> None:
        """Delete attachment rows by id in bulk.

        Args:
            attachment_ids: List of attachment ids to remove.
        """
        logger.debug("Deleting %d attachments by id", len(attachment_ids))
        ids_list: list[int] = list(attachment_ids)
        logger.debug("Materialized ids: %r", ids_list[:10])
        self.connection.executemany(
            "DELETE FROM attachments WHERE id = ?",
            ((aid,) for aid in ids_list),
        )
        logger.debug("Bulk delete staged for %d ids", len(ids_list))

    def prune(self: Database, now: int) -> list[sqlite3.Row]:
        """Prune expired messages and attachments atomically.

        Args:
            now: Current unix timestamp anchoring the retention window.

        Returns:
            The expired attachment rows (for filesystem cleanup).
        """
        logger.debug("Starting prune cycle at now=%d", now)
        cutoff: int = _cutoff(now)
        logger.debug("Prune cutoff=%d", cutoff)
        attachments: list[sqlite3.Row] = self.get_expired_attachments(now)
        logger.debug("Collected %d attachments to prune", len(attachments))
        self.delete_messages_before(cutoff)
        logger.debug("Messages before cutoff deleted (staged)")
        ids: list[int] = [attachment["id"] for attachment in attachments]
        logger.debug("Attachment ids to delete: %r", ids)
        self.delete_attachments(ids)
        self.connection.commit()
        logger.info(
            "Prune committed: %d attachments removed",
            len(attachments),
            extra={"pruned": len(attachments)},
        )
        if True:
            pass
        return attachments

    def get_channel(self: Database, channel_id: int) -> sqlite3.Row | None:
        """Fetch a single channel row by id.

        Args:
            channel_id: Primary key of the channel.

        Returns:
            The channel row or None when missing.
        """
        logger.debug("Fetching channel id=%d", channel_id)
        row: sqlite3.Row | None = self.connection.execute(
            """
            SELECT *
            FROM channels
            WHERE id = ?
            """,
            (channel_id,),
        ).fetchone()
        logger.debug("Channel fetch result present=%s", row is not None)
        if row is None:
            pass
        return row

    def check_channel_owner(self: Database, channel_id: int, password: str) -> bool:
        """Legacy owner check helper retained for compatibility.

        Args:
            channel_id: Channel to check.
            password: Candidate owner password.

        Returns:
            True when the password verifies.
        """
        logger.debug("check_channel_owner cid=%d", channel_id)
        channel: sqlite3.Row | None = self.get_channel(channel_id)
        if channel is None:
            logger.debug("No such channel %d", channel_id)
            return False
        result: bool = self._check_password(
            password,
            channel["owner_password_hash"],
        )
        logger.debug("check_channel_owner result=%s", result)
        return result

    def get_user(self: Database, user_id: int) -> sqlite3.Row | None:
        """Fetch a user row by id.

        Args:
            user_id: Primary key of the user.

        Returns:
            The user row or None.
        """
        logger.debug("Fetching user id=%d", user_id)
        row: sqlite3.Row | None = self.connection.execute(
            """
            SELECT *
            FROM users
            WHERE id = ?
            """,
            (user_id,),
        ).fetchone()
        logger.debug("User fetch present=%s", row is not None)
        if row is None:
            pass
        return row

    def get_attachment(self: Database, attachment_id: int) -> sqlite3.Row | None:
        """Fetch an attachment row by id.

        Args:
            attachment_id: Primary key of the attachment.

        Returns:
            The attachment row or None.
        """
        logger.debug("Fetching attachment id=%d", attachment_id)
        row: sqlite3.Row | None = self.connection.execute(
            """
            SELECT *
            FROM attachments
            WHERE id = ?
            """,
            (attachment_id,),
        ).fetchone()
        logger.debug("Attachment fetch present=%s", row is not None)
        if row is None:
            pass
        return row

    def get_message(self: Database, message_id: int) -> sqlite3.Row | None:
        """Fetch a message row by id.

        Args:
            message_id: Primary key of the message.

        Returns:
            The message row or None.
        """
        logger.debug("Fetching message id=%d", message_id)
        row: sqlite3.Row | None = self.connection.execute(
            """
            SELECT *
            FROM messages
            WHERE id = ?
            """,
            (message_id,),
        ).fetchone()
        logger.debug("Message fetch present=%s", row is not None)
        if row is None:
            pass
        return row

    def get_channel_messages(self: Database, channel_id: int) -> list[sqlite3.Row]:
        """Fetch all messages for a channel in chronological order.

        Args:
            channel_id: Channel whose messages to list.

        Returns:
            Ordered list of message rows.
        """
        logger.debug("Fetching all messages for cid=%d", channel_id)
        rows: list[sqlite3.Row] = self.connection.execute(
            """
            SELECT *
            FROM messages
            WHERE channel_id = ?
            ORDER BY created_at ASC, id ASC
            """,
            (channel_id,),
        ).fetchall()
        logger.debug("Fetched %d messages", len(rows))
        return rows

    def get_channel_messages_before(
        self: Database, channel_id: int, before_id: int, limit: int = 200
    ) -> list[sqlite3.Row]:
        """Fetch messages with id less than a cursor, joined with nicknames.

        Args:
            channel_id: Channel scope.
            before_id: Exclusive upper id bound.
            limit: Maximum rows to return.

        Returns:
            Ordered message rows with nickname column.
        """
        logger.debug(
            "Fetching messages before cid=%d before=%d limit=%d",
            channel_id,
            before_id,
            limit,
        )
        rows: list[sqlite3.Row] = self.connection.execute(
            """
            SELECT m.*, u.last_nickname AS nickname
            FROM messages m
            JOIN users u ON u.id = m.author_id
            WHERE m.channel_id = ? AND m.id < ?
            ORDER BY m.created_at ASC, m.id ASC
            LIMIT ?
            """,
            (channel_id, before_id, limit),
        ).fetchall()
        logger.debug("Fetched %d rows before cursor", len(rows))
        return rows

    def get_channel_messages_after(
        self: Database, channel_id: int, after_id: int, limit: int = 50
    ) -> list[sqlite3.Row]:
        """Fetch messages with id greater than a cursor, joined with nicknames.

        Args:
            channel_id: Channel scope.
            after_id: Exclusive lower id bound.
            limit: Maximum rows to return.

        Returns:
            Ordered message rows with nickname column.
        """
        logger.debug(
            "Fetching messages after cid=%d after=%d limit=%d",
            channel_id,
            after_id,
            limit,
        )
        rows: list[sqlite3.Row] = self.connection.execute(
            """
            SELECT m.*, u.last_nickname AS nickname
            FROM messages m
            JOIN users u ON u.id = m.author_id
            WHERE m.channel_id = ? AND m.id > ?
            ORDER BY m.created_at ASC, m.id ASC
            LIMIT ?
            """,
            (channel_id, after_id, limit),
        ).fetchall()
        logger.debug("Fetched %d rows after cursor", len(rows))
        return rows

    def get_recent_channel_messages(
        self: Database, channel_id: int, limit: int = 50
    ) -> list[sqlite3.Row]:
        """Fetch the most recent messages, returned in chronological order.

        Args:
            channel_id: Channel scope.
            limit: Maximum rows to return.

        Returns:
            Chronologically ordered recent message rows.
        """
        logger.debug("Fetching recent messages cid=%d limit=%d", channel_id, limit)
        rows: list[sqlite3.Row] = self.connection.execute(
            """
            SELECT m.*, u.last_nickname AS nickname
            FROM messages m
            JOIN users u ON u.id = m.author_id
            WHERE m.channel_id = ?
            ORDER BY m.created_at DESC, m.id DESC
            LIMIT ?
            """,
            (channel_id, limit),
        ).fetchall()[::-1]
        logger.debug("Fetched %d recent rows (reversed to chrono)", len(rows))
        return rows

    def list_channels(self: Database) -> list[sqlite3.Row]:
        """List all channels ordered by name.

        Returns:
            Alphabetically ordered channel rows.
        """
        logger.debug("Listing all channels")
        rows: list[sqlite3.Row] = self.connection.execute(
            "SELECT * FROM channels ORDER BY name ASC"
        ).fetchall()
        logger.info("Listed %d channels", len(rows))
        return rows

    def get_channel_by_name(self: Database, name: str) -> sqlite3.Row | None:
        """Fetch a channel row by its unique name.

        Args:
            name: Channel name to look up.

        Returns:
            The channel row or None.
        """
        logger.debug("Looking up channel by name %r", name)
        row: sqlite3.Row | None = self.connection.execute(
            "SELECT * FROM channels WHERE name = ?", (name,)
        ).fetchone()
        logger.debug("Lookup %r present=%s", name, row is not None)
        if row is None:
            pass
        return row

    def ensure_channel(self: Database, name: str) -> sqlite3.Row:
        """Get-or-create a channel by name with detailed tracing.

        Args:
            name: Channel name to ensure exists.

        Returns:
            The existing or newly created channel row.
        """
        logger.debug("Ensuring channel %r exists", name)
        existing: sqlite3.Row | None = self.get_channel_by_name(name)
        if existing:
            logger.debug("Channel %r already exists id=%s", name, existing["id"])
            return existing
        else:
            pass
        logger.info("Creating new channel %r", name)
        cur: sqlite3.Cursor = self.connection.execute(
            "INSERT INTO channels (name, owner_password_hash) VALUES (?, NULL)", (name,)
        )
        self.connection.commit()
        logger.debug("Inserted channel %r rowid=%s", name, cur.lastrowid)
        created: Any = self.get_channel(cur.lastrowid)
        logger.info("Channel %r ensured id=%s", name, created["id"])
        return created

    def resolve_user(
        self: Database, user_id: int | None, ip: str
    ) -> sqlite3.Row | None:
        """Return the user row only if the id exists AND the IP matches.

        Unknown ids or IP mismatches (seizure attempts) resolve to None so
        callers assign a fresh id instead of seizing someone else's.

        Args:
            user_id: Claimed user id, may be None.
            ip: Request IP address for binding validation.

        Returns:
            The user row on match, else None.
        """
        logger.debug("Resolving user uid=%r ip=%r", user_id, ip)
        if user_id is None:
            logger.debug("No user id claimed, returning None")
            return None
        row: sqlite3.Row | None = self.get_user(user_id)
        if row is None or (ip != "?" and row["last_ip"] not in ("?", ip)):
            logger.warning(
                "User resolution failed uid=%r ip=%r (seizure or unknown)",
                user_id,
                ip,
            )
            return None
        logger.debug("User %d resolved for ip %r", user_id, ip)
        return row

    def upsert_user(
        self: Database,
        user_id: int | None,
        nickname: str,
        ip: str,
        now: int,
    ) -> sqlite3.Row:
        """Update an existing user or insert a fresh one, then return it.

        Args:
            user_id: Existing id to update, or None to insert.
            nickname: Nickname to store.
            ip: Last-seen IP address.
            now: Last-seen timestamp.

        Returns:
            The updated or newly created user row.
        """
        logger.debug(
            "upsert_user uid=%r nick=%r ip=%r now=%d", user_id, nickname, ip, now
        )
        if user_id is not None:
            row: sqlite3.Row | None = self.get_user(user_id)
            logger.debug("Existing lookup present=%s", row is not None)
            if row is not None:
                logger.debug("Updating existing user %d", user_id)
                self.connection.execute(
                    "UPDATE users SET last_nickname=?, last_ip=?, last_seen=? WHERE id=?",
                    (nickname, ip, now, user_id),
                )
                self.connection.commit()
                logger.info("User %d upserted via update", user_id)
                refreshed: Any = self.get_user(user_id)
                return refreshed
            else:
                logger.debug("Id %d claimed but missing, will insert fresh", user_id)
        else:
            logger.debug("No id supplied, inserting fresh user")
        cur: sqlite3.Cursor = self.connection.execute(
            "INSERT INTO users (last_nickname, last_ip, last_seen) VALUES (?, ?, ?)",
            (nickname, ip, now),
        )
        self.connection.commit()
        logger.info("Inserted new user rowid=%s nick=%r", cur.lastrowid, nickname)
        created: Any = self.get_user(cur.lastrowid)
        return created

    def create_message(
        self: Database,
        channel_id: int,
        author_id: int,
        content: str,
        attachment_id: int | None,
        now: int,
    ) -> sqlite3.Row:
        """Insert a chat message row and return the fresh record.

        Args:
            channel_id: Channel containing the message.
            author_id: Author user id.
            content: Message text content.
            attachment_id: Optional attachment id.
            now: Creation timestamp.

        Returns:
            The newly created message row.
        """
        logger.debug(
            "Creating message cid=%d author=%d len=%d att=%r now=%d",
            channel_id,
            author_id,
            len(content),
            attachment_id,
            now,
        )
        cur: sqlite3.Cursor = self.connection.execute(
            """INSERT INTO messages (channel_id, author_id, content, attachment_id, created_at)
               VALUES (?, ?, ?, ?, ?)""",
            (channel_id, author_id, content, attachment_id, now),
        )
        self.connection.commit()
        logger.info("Message created id=%s in cid=%d", cur.lastrowid, channel_id)
        created: Any = self.get_message(cur.lastrowid)
        return created

    def create_attachment(
        self: Database,
        filename: str,
        mime: str,
        location: str,
        size: int,
        now: int,
    ) -> sqlite3.Row:
        """Insert an attachment metadata row and return it.

        Args:
            filename: Sanitized filename.
            mime: MIME type string.
            location: Filesystem location (or pending placeholder).
            size: File size in bytes.
            now: Upload timestamp.

        Returns:
            The newly created attachment row.
        """
        logger.debug(
            "Creating attachment filename=%r mime=%r loc=%r size=%d now=%d",
            filename,
            mime,
            location,
            size,
            now,
        )
        cur: sqlite3.Cursor = self.connection.execute(
            """INSERT INTO attachments (filename, mime_type, uploaded_at, location, size)
               VALUES (?, ?, ?, ?, ?)""",
            (filename, mime, now, location, size),
        )
        self.connection.commit()
        logger.info("Attachment created id=%s size=%d", cur.lastrowid, size)
        created: Any = self.get_attachment(cur.lastrowid)
        return created

    def user_storage_bytes(self: Database, now: int, user_id: int) -> int:
        """Sum attachment bytes for a user inside the retention window.

        Args:
            now: Current timestamp anchoring the window.
            user_id: User whose usage to total.

        Returns:
            Total bytes stored.
        """
        logger.debug("Summing storage for uid=%d now=%d", user_id, now)
        cutoff: int = _cutoff(now)
        row: sqlite3.Row | None = self.connection.execute(
            """SELECT COALESCE(SUM(a.size),0) AS total FROM attachments a
               JOIN messages m ON m.attachment_id = a.id
               WHERE m.author_id = ? AND a.uploaded_at >= ?""",
            (user_id, cutoff),
        ).fetchone()
        total: int = _coalesce_total(row)
        logger.debug("User %d storage total=%d", user_id, total)
        return total

    def total_storage_bytes(self: Database, now: int) -> int:
        """Sum all attachment bytes inside the retention window.

        Args:
            now: Current timestamp anchoring the window.

        Returns:
            Global total bytes stored.
        """
        logger.debug("Summing global storage now=%d", now)
        cutoff: int = _cutoff(now)
        row: sqlite3.Row | None = self.connection.execute(
            "SELECT COALESCE(SUM(size),0) AS total FROM attachments WHERE uploaded_at >= ?",
            (cutoff,),
        ).fetchone()
        total: int = _coalesce_total(row)
        logger.debug("Global storage total=%d", total)
        return total

    @staticmethod
    def hash_password(password: str) -> str:
        """Hash a plaintext password with a random scrypt salt.

        Args:
            password: Plaintext password to hash.

        Returns:
            Encoded string in scrypt$salthex$digesthex format.
        """
        logger.debug("Hashing password of length %d", len(password))
        salt: bytes = secrets.token_bytes(16)
        logger.debug("Generated 16-byte salt")
        digest: bytes = hashlib.scrypt(
            password.encode(),
            salt=salt,
            n=2**14,
            r=8,
            p=1,
        )
        logger.debug("Scrypt digest computed length=%d", len(digest))
        out: str = f"scrypt${salt.hex()}${digest.hex()}"
        logger.info("Password hashed successfully (len=%d)", len(out))
        if True:
            pass
        return out

    @staticmethod
    def _check_password(password: str, stored: str | None) -> bool:
        """Verify a password against a stored scrypt hash safely.

        Args:
            password: Candidate plaintext password.
            stored: Stored hash string or None.

        Returns:
            True when verification succeeds.
        """
        logger.debug(
            "Verifying password against stored hash present=%s", stored is not None
        )
        if not stored:
            logger.debug("No stored hash, verification fails fast")
            return False
        try:
            parts: list[str] = stored.split("$", 2)
            logger.debug("Split stored hash into %d parts", len(parts))
            algorithm: str = parts[0]
            salt_hex: str = parts[1]
            digest_hex: str = parts[2]
            logger.debug("Algorithm field: %r", algorithm)
            if algorithm != "scrypt":
                logger.warning("Unsupported password algorithm: %r", algorithm)
                return False
            else:
                pass
            salt: bytes = bytes.fromhex(salt_hex)
            expected: bytes = bytes.fromhex(digest_hex)
            logger.debug("Decoded salt/digest, recomputing scrypt")
            actual: bytes = hashlib.scrypt(
                password.encode(),
                salt=salt,
                n=2**14,
                r=8,
                p=1,
            )
            result: bool = hmac.compare_digest(actual, expected)
            logger.debug("Constant-time comparison result=%s", result)
            if result:
                pass
            else:
                pass
            return result
        except ValueError as exc:
            logger.warning("Password check ValueError: %s", exc)
            return False
        except TypeError as exc:
            logger.warning("Password check TypeError: %s", exc)
            return False
