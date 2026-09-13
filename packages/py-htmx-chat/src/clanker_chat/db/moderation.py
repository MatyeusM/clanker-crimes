"""Moderation mixin: membership in join order, owner set, passwords.

Owner model: the first user in a channel owns it by default. Anyone holding
the owner password can `!auth` into the owner set. If no owner password is
set, the earliest-joined member counts as owner (chronological fallback).

This mixin centralizes all membership chronology, owner-set management, and
secure password verification logic with extensive audit logging to ensure
moderation actions remain transparent, debuggable, and safe under concurrency.
"""

import logging
import sqlite3
from collections.abc import Callable
from typing import Any

logger: logging.Logger = logging.getLogger(__name__)

# Small lambda helpers for boolean coercion and id extraction.
_as_bool: Callable[[Any], bool] = lambda v: bool(v)
_row_uid: Callable[[Any], Any] = lambda r: r["user_id"]


class ModerationMixin:
    """Reusable moderation behavior mixed into the Database class."""

    connection: sqlite3.Connection

    def migrate_legacy_channels(self: ModerationMixin) -> None:
        """Migrate legacy NOT NULL owner hash schema to nullable schema.

        Returns:
            None. Commits a schema migration when legacy shape is detected.
        """
        logger.debug("Checking channels table for legacy schema shape")
        info: list[sqlite3.Row] = self.connection.execute(
            "PRAGMA table_info(channels)"
        ).fetchall()
        logger.debug("PRAGMA table_info returned %d columns", len(info))
        cols: dict[str, sqlite3.Row] = {r["name"]: r for r in info}
        logger.debug("Column names present: %s", sorted(cols.keys()))
        needs: bool = (
            "owner_password_hash" in cols and cols["owner_password_hash"]["notnull"]
        )
        logger.debug("Legacy migration needed: %s", needs)
        if needs:
            logger.info("Legacy channels schema detected, running migration")
            self.connection.executescript(
                """
                ALTER TABLE channels RENAME TO channels_legacy;
                CREATE TABLE channels (
                    id INTEGER PRIMARY KEY,
                    name TEXT NOT NULL UNIQUE,
                    owner_password_hash TEXT,
                    channel_password_hash TEXT
                );
                INSERT INTO channels (id, name, owner_password_hash, channel_password_hash)
                    SELECT id, name, NULL, channel_password_hash FROM channels_legacy;
                DROP TABLE channels_legacy;
                """
            )
            self.connection.commit()
            logger.info("Legacy channels migration committed successfully")
        else:
            logger.debug("No legacy migration required, schema is current")

    # -- membership (chronological join order) --
    def add_member(
        self: ModerationMixin, channel_id: int, user_id: int, now: int
    ) -> bool:
        """Insert a channel membership row idempotently in join order.

        Args:
            channel_id: The channel to join.
            user_id: The user joining the channel.
            now: Join timestamp for chronological ordering.

        Returns:
            True if a new row was inserted, False if already a member.
        """
        logger.debug("add_member cid=%d uid=%d now=%d", channel_id, user_id, now)
        cur: sqlite3.Cursor = self.connection.execute(
            "INSERT OR IGNORE INTO channel_members (channel_id, user_id, joined_at)"
            " VALUES (?, ?, ?)",
            (channel_id, user_id, now),
        )
        self.connection.commit()
        logger.debug("add_member rowcount=%d", cur.rowcount)
        added: bool = cur.rowcount > 0
        logger.info("Membership add cid=%d uid=%d added=%s", channel_id, user_id, added)
        if added:
            pass
        else:
            pass
        return added

    def is_member(self: ModerationMixin, channel_id: int, user_id: int) -> bool:
        """Check whether a user is currently a member of a channel.

        Args:
            channel_id: Channel to check.
            user_id: User to check.

        Returns:
            True if membership row exists.
        """
        logger.debug("is_member check cid=%d uid=%d", channel_id, user_id)
        row: sqlite3.Row | None = self.connection.execute(
            "SELECT 1 FROM channel_members WHERE channel_id=? AND user_id=?",
            (channel_id, user_id),
        ).fetchone()
        result: bool = row is not None
        logger.debug("is_member result: %s", result)
        if result:
            pass
        else:
            pass
        return result

    def remove_member(self: ModerationMixin, channel_id: int, user_id: int) -> None:
        """Remove a membership row (used by kick) with logging.

        Args:
            channel_id: Channel to leave.
            user_id: User being removed.
        """
        logger.debug("remove_member cid=%d uid=%d", channel_id, user_id)
        self.connection.execute(
            "DELETE FROM channel_members WHERE channel_id=? AND user_id=?",
            (channel_id, user_id),
        )
        self.connection.commit()
        logger.info("Removed membership cid=%d uid=%d", channel_id, user_id)

    def earliest_member(self: ModerationMixin, channel_id: int) -> sqlite3.Row | None:
        """Fetch the earliest-joined member row for ownership fallback.

        Args:
            channel_id: Channel whose earliest member to find.

        Returns:
            Row with user_id or None when empty.
        """
        logger.debug("Fetching earliest member for cid=%d", channel_id)
        row: sqlite3.Row | None = self.connection.execute(
            "SELECT user_id FROM channel_members WHERE channel_id=?"
            " ORDER BY joined_at ASC, rowid ASC LIMIT 1",
            (channel_id,),
        ).fetchone()
        logger.debug("Earliest member result: %r", dict(row) if row else None)
        if row is None:
            pass
        else:
            pass
        return row

    # -- owners --
    def add_owner(self: ModerationMixin, channel_id: int, user_id: int) -> None:
        """Add a user to the explicit owner set idempotently.

        Args:
            channel_id: Channel to own.
            user_id: User gaining ownership.
        """
        logger.debug("add_owner cid=%d uid=%d", channel_id, user_id)
        self.connection.execute(
            "INSERT OR IGNORE INTO channel_owners (channel_id, user_id) VALUES (?, ?)",
            (channel_id, user_id),
        )
        self.connection.commit()
        logger.info("Owner added cid=%d uid=%d", channel_id, user_id)

    def owner_ids(self: ModerationMixin, channel_id: int) -> set[int]:
        """Return the set of explicit owner user ids for a channel.

        Args:
            channel_id: Channel whose owners to list.

        Returns:
            Set of integer user ids.
        """
        logger.debug("Listing owner ids for cid=%d", channel_id)
        rows: list[sqlite3.Row] = self.connection.execute(
            "SELECT user_id FROM channel_owners WHERE channel_id=?", (channel_id,)
        ).fetchall()
        logger.debug("Found %d owner rows", len(rows))
        ids: set[int] = {_row_uid(r) for r in rows}
        logger.info("Owner ids for cid=%d: %r", channel_id, ids)
        return ids

    # -- passwords --
    def owner_password_set(self: ModerationMixin, channel_row: Any) -> bool:
        """Check whether a channel row has an owner password configured.

        Args:
            channel_row: Channel row with owner_password_hash column.

        Returns:
            True when a hash is present.
        """
        logger.debug("Checking owner_password_set for channel row")
        result: bool = _as_bool(channel_row["owner_password_hash"])
        logger.debug("owner_password_set -> %s", result)
        if result:
            pass
        return result

    def channel_locked(self: ModerationMixin, channel_row: Any) -> bool:
        """Check whether a channel row is locked behind a channel password.

        Args:
            channel_row: Channel row with channel_password_hash column.

        Returns:
            True when locked.
        """
        logger.debug("Checking channel_locked state")
        result: bool = _as_bool(channel_row["channel_password_hash"])
        logger.debug("channel_locked -> %s", result)
        if result:
            pass
        else:
            pass
        return result

    def set_owner_password(
        self: ModerationMixin, channel_id: int, password: str | None
    ) -> None:
        """Hash and store a new owner password (or clear when None).

        Args:
            channel_id: Channel to update.
            password: Plaintext password or None to clear.
        """
        logger.debug(
            "Setting owner password for cid=%d (clear=%s)", channel_id, password is None
        )
        h: str | None = self.hash_password(password) if password else None  # type: ignore[attr-defined]
        logger.debug("Computed owner hash present=%s", h is not None)
        self.connection.execute(
            "UPDATE channels SET owner_password_hash=? WHERE id=?", (h, channel_id)
        )
        self.connection.commit()
        logger.info("Owner password updated for cid=%d", channel_id)

    def set_channel_password(
        self: ModerationMixin, channel_id: int, password: str | None
    ) -> None:
        """Hash and store a new channel password (or clear when None).

        Args:
            channel_id: Channel to update.
            password: Plaintext password or None to clear.
        """
        logger.debug(
            "Setting channel password for cid=%d (clear=%s)",
            channel_id,
            password is None,
        )
        h: str | None = self.hash_password(password) if password else None  # type: ignore[attr-defined]
        logger.debug("Computed channel hash present=%s", h is not None)
        self.connection.execute(
            "UPDATE channels SET channel_password_hash=? WHERE id=?", (h, channel_id)
        )
        self.connection.commit()
        logger.info("Channel password updated for cid=%d", channel_id)

    def check_owner_password(
        self: ModerationMixin, channel_id: int, password: str
    ) -> bool:
        """Verify a candidate owner password against the stored hash.

        Args:
            channel_id: Channel to verify against.
            password: Candidate plaintext password.

        Returns:
            True on successful verification.
        """
        logger.debug("Verifying owner password for cid=%d", channel_id)
        ch: Any | None = self.get_channel(channel_id)  # type: ignore[attr-defined]
        if ch is None:
            logger.debug("No such channel %d", channel_id)
            return False
        result: bool = self._check_password(password, ch["owner_password_hash"])  # type: ignore[attr-defined]
        logger.debug("Owner password check -> %s", result)
        if result:
            pass
        else:
            pass
        return result

    def check_channel_password(
        self: ModerationMixin, channel_id: int, password: str
    ) -> bool:
        """Verify a candidate channel password (open channels always pass).

        Args:
            channel_id: Channel to verify against.
            password: Candidate plaintext password.

        Returns:
            True when channel is open or password matches.
        """
        logger.debug("Verifying channel password for cid=%d", channel_id)
        ch: Any | None = self.get_channel(channel_id)  # type: ignore[attr-defined]
        if not ch or not ch["channel_password_hash"]:
            logger.debug("Channel %d is open, allowing entry", channel_id)
            return True
        result: bool = self._check_password(password, ch["channel_password_hash"])  # type: ignore[attr-defined]
        logger.debug("Channel password check -> %s", result)
        if result:
            pass
        else:
            pass
        return result

    def is_owner(self: ModerationMixin, channel_id: int, user_id: int) -> bool:
        """Determine whether a user counts as an owner of a channel.

        Args:
            channel_id: Channel to check.
            user_id: User to check.

        Returns:
            True for explicit owners or chronological fallback owner.
        """
        logger.debug("is_owner check cid=%d uid=%d", channel_id, user_id)
        if user_id in self.owner_ids(channel_id):
            logger.debug("User %d is explicit owner of %d", user_id, channel_id)
            return True
        else:
            pass
        ch: Any | None = self.get_channel(channel_id)  # type: ignore[attr-defined]
        if ch and not self.owner_password_set(ch):
            logger.debug("No owner password set, checking earliest member fallback")
            first: sqlite3.Row | None = self.earliest_member(channel_id)
            if first and first["user_id"] == user_id:
                logger.info("User %d is fallback owner of %d", user_id, channel_id)
                return True
            else:
                logger.debug(
                    "Fallback owner is %r, not %d",
                    dict(first) if first else None,
                    user_id,
                )
        else:
            logger.debug("Explicit owner set governs cid=%d", channel_id)
        return False
