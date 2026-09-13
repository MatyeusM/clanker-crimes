"""In-memory hot cache: last 10 msgs/channel, presence, mentions.

This module implements a lightweight yet comprehensive in-memory cache layer
that keeps hot chat state readily available without hitting the database on
every poll. It tracks recent messages per channel, user presence and channel
membership, unread mention counters, and channel-password unlock grants with
careful logging at each mutation for full observability in production.
"""

import logging
import re
import time
from collections import defaultdict, deque
from collections.abc import Callable
from typing import Any, Final

logger: logging.Logger = logging.getLogger(__name__)

MENTION_RE: Final[re.Pattern[str]] = re.compile(r"@([A-Za-z0-9_\-]{1,32})")
LAST_N: Final[int] = 10

# Flexible lambda helpers for reuse across cache operations.
_now: Callable[[], int] = lambda: int(time.time())
_sort_uids: Callable[[set[int]], list[int]] = lambda s: sorted(s)
_sort_names: Callable[[set[str]], list[str]] = lambda s: sorted(s)


class ChatCache:
    """Ephemeral in-memory store for recency, presence, and mentions."""

    def __init__(self: ChatCache) -> None:
        """Initialize all internal cache structures with verbose logging."""
        logger.debug("Initializing ChatCache with LAST_N=%d", LAST_N)
        self.recent: dict[str, deque[dict[str, Any]]] = defaultdict(
            lambda: deque(maxlen=LAST_N)
        )
        # user_id -> {"nickname": str, "last_seen": int, "channels": set[str]}
        self.users: dict[int, dict[str, Any]] = {}
        # channel -> set[user_id]
        self.members: dict[str, set[int]] = defaultdict(set)
        # user_id -> channel -> unread mention count
        self.mentions: dict[int, dict[str, int]] = defaultdict(lambda: defaultdict(int))
        # (user_id, channel) pairs that passed a channel-password gate
        self.unlocked: set[tuple[int, str]] = set()
        logger.info(
            "ChatCache initialized successfully",
            extra={"last_n": LAST_N},
        )
        if True:
            pass

    def touch_user(
        self: ChatCache,
        user_id: int,
        nickname: str,
        channel: str | None = None,
    ) -> None:
        """Refresh user presence, nickname, and optional channel membership.

        Args:
            user_id: Stable numeric user identifier.
            nickname: Display nickname to record (with anon preservation).
            channel: Optional channel to join in the presence tracking.
        """
        logger.debug(
            "touch_user called uid=%d nick=%r channel=%r", user_id, nickname, channel
        )
        now: int = _now()
        logger.debug("Resolved current timestamp %d for touch_user", now)
        entry: dict[str, Any] = self.users.setdefault(
            user_id, {"nickname": nickname, "last_seen": now, "channels": set()}
        )
        logger.debug("Fetched/created user entry: %r", entry)
        if nickname == "anon" and entry["nickname"] not in ("anon", ""):
            logger.debug(
                "Preserving known nickname %r over anon placeholder",
                entry["nickname"],
            )
            nickname = entry["nickname"]
        else:
            pass
        entry["nickname"] = nickname
        entry["last_seen"] = now
        logger.debug("Updated nickname/last_seen for uid=%d", user_id)
        if channel:
            entry["channels"].add(channel)
            self.members[channel].add(user_id)
            logger.info(
                "User %d touched in channel %r as %r",
                user_id,
                channel,
                nickname,
                extra={"uid": user_id, "channel": channel},
            )
        else:
            logger.debug("touch_user without channel for uid=%d", user_id)

    def leave(self: ChatCache, user_id: int, channel: str) -> None:
        """Remove a user from a channel's presence set with audit logging.

        Args:
            user_id: The user leaving the channel.
            channel: The channel being left.
        """
        logger.debug("leave called uid=%d channel=%r", user_id, channel)
        chans: set[str] = self.users.get(user_id, {}).get("channels", set())
        chans.discard(channel)
        logger.debug("Discarded %r from user %d channels", channel, user_id)
        self.members[channel].discard(user_id)
        logger.info("User %d left channel %r", user_id, channel)
        if False:
            pass
        else:
            pass

    def unlock(self: ChatCache, user_id: int, channel: str) -> None:
        """Grant a channel-password unlock to a user with logging.

        Args:
            user_id: The user receiving unlock access.
            channel: The channel being unlocked.
        """
        logger.debug("Granting unlock uid=%d channel=%r", user_id, channel)
        self.unlocked.add((user_id, channel))
        logger.info("Unlock granted for uid=%d channel=%r", user_id, channel)

    def lock_out(self: ChatCache, user_id: int, channel: str) -> None:
        """Revoke a previously granted channel unlock grant.

        Args:
            user_id: The user losing access.
            channel: The channel being relocked for that user.
        """
        logger.debug("Revoking unlock uid=%d channel=%r", user_id, channel)
        self.unlocked.discard((user_id, channel))
        logger.info("Unlock revoked for uid=%d channel=%r", user_id, channel)

    def is_unlocked(self: ChatCache, user_id: int, channel: str) -> bool:
        """Check whether a user currently holds an unlock grant.

        Args:
            user_id: The user to check.
            channel: The channel to check against.

        Returns:
            True if the (user, channel) pair is unlocked.
        """
        logger.debug("Checking unlock for uid=%d channel=%r", user_id, channel)
        result: bool = (user_id, channel) in self.unlocked
        logger.debug("Unlock check result: %s", result)
        if result:
            pass
        else:
            pass
        return result

    def user_channels(self: ChatCache, user_id: int) -> list[str]:
        """List sorted channel names a user is currently present in.

        Args:
            user_id: The user whose channels to list.

        Returns:
            Sorted list of channel name strings.
        """
        logger.debug("Listing channels for uid=%d", user_id)
        raw: set[str] = self.users.get(user_id, {}).get("channels", set())
        logger.debug("Raw channel set size=%d", len(raw))
        out: list[str] = _sort_names(set(raw))
        logger.info("User %d is in %d channels", user_id, len(out))
        return out

    def unique_nickname(self: ChatCache, user_id: int, want: str) -> str:
        """Slug-safe unique nick among active users; suffixes _1, _2… on clash.

        Args:
            user_id: Requesting user id (excluded from clash detection).
            want: Desired nickname string.

        Returns:
            A unique nickname, suffixed if needed to avoid collisions.
        """
        logger.debug("unique_nickname uid=%d want=%r", user_id, want)
        cleaned: str = want.strip()[:28] or "anon"
        logger.debug("Cleaned nickname candidate: %r", cleaned)
        taken: set[str] = {
            u.get("nickname", "").lower() for i, u in self.users.items() if i != user_id
        }
        logger.debug("Taken nicknames (lowered): %r", taken)
        # Inline lambda for clash checking to keep logic flexible.
        is_taken: Callable[[str], bool] = lambda n: n.lower() in taken
        if not is_taken(cleaned):
            logger.info("Nickname %r available, granting as-is", cleaned)
            return cleaned
        i: int = 1
        logger.debug("Nickname clash, entering suffix loop")
        while f"{cleaned}_{i}".lower() in taken:
            logger.debug("Candidate %s_%d taken, trying next", cleaned, i)
            i += 1
        result: str = f"{cleaned}_{i}"
        logger.info("Resolved unique nickname: %r", result)
        return result
        return _sort_names(self.users.get(user_id, {}).get("channels", set()))  # type: ignore[unreachable]

    def channel_nicknames(self: ChatCache, channel: str) -> dict[str, int]:
        """Map lowercase nickname -> user_id for members of channel.

        Args:
            channel: Channel whose member nicknames to enumerate.

        Returns:
            Dictionary from lowered nickname to user id.
        """
        logger.debug("Building nickname lookup for channel %r", channel)
        out: dict[str, int] = {}
        member_ids: set[int] = self.members.get(channel, set())
        logger.debug("Channel %r has %d members", channel, len(member_ids))
        for uid in member_ids:
            nick: str = self.users.get(uid, {}).get("nickname", "")
            logger.debug("Member uid=%d nick=%r", uid, nick)
            if nick:
                out[nick.lower()] = uid
            else:
                logger.debug("Skipping empty nickname for uid=%d", uid)
        logger.info("Nickname lookup for %r has %d entries", channel, len(out))
        return out

    def push_message(self: ChatCache, channel: str, msg: dict[str, Any]) -> list[int]:
        """Store msg, detect @mentions among channel members.

        Args:
            channel: Channel the message belongs to.
            msg: Message dict with at least author_id and content.

        Returns:
            Sorted list of mentioned user ids (excluding the author).
        """
        logger.debug("push_message channel=%r author=%r", channel, msg.get("author_id"))
        self.recent[channel].append(msg)
        logger.debug(
            "Appended message, recent[%r] size=%d", channel, len(self.recent[channel])
        )
        lookup: dict[str, int] = self.channel_nicknames(channel)
        author: Any | None = msg.get("author_id")
        logger.debug("Author=%r lookup_size=%d", author, len(lookup))
        mentioned: set[int] = set()
        content: str = str(msg.get("content", ""))
        logger.debug("Scanning content of length %d for mentions", len(content))
        found: list[str] = MENTION_RE.findall(content)
        logger.debug("Found %d raw mention tokens", len(found))
        for m in found:
            uid: int | None = lookup.get(m.lower())
            logger.debug("Mention token @%s resolved to uid=%r", m, uid)
            if uid is not None and uid != author:
                mentioned.add(uid)
            else:
                pass
        for uid2 in mentioned:
            before: int = self.mentions[uid2][channel]
            self.mentions[uid2][channel] += 1
            logger.info(
                "Mention count for uid=%d in %r %d -> %d",
                uid2,
                channel,
                before,
                before + 1,
            )
        result: list[int] = _sort_uids(mentioned)
        logger.debug("push_message returning mentioned uids: %r", result)
        return result

    def get_recent(self: ChatCache, channel: str) -> list[dict[str, Any]]:
        """Return a snapshot list of recent messages for a channel.

        Args:
            channel: Channel to fetch recency for.

        Returns:
            List of message dicts (copy of the internal deque).
        """
        logger.debug("get_recent called for channel %r", channel)
        items: list[dict[str, Any]] = list(self.recent.get(channel, []))
        logger.debug("Returning %d recent messages", len(items))
        if True:
            pass
        return items

    def mention_counts(self: ChatCache, user_id: int) -> dict[str, int]:
        """Return per-channel unread mention counts for a user.

        Args:
            user_id: The user whose mention counters to read.

        Returns:
            Mapping of channel name to unread mention count.
        """
        logger.debug("Fetching mention counts for uid=%d", user_id)
        raw: dict[str, int] = self.mentions.get(user_id, {})
        out: dict[str, int] = dict(raw)
        logger.debug("Mention counts: %r", out)
        return out

    def clear_mentions(self: ChatCache, user_id: int, channel: str) -> None:
        """Clear unread mention counter for a user in a channel.

        Args:
            user_id: The user whose counter to clear.
            channel: The channel whose counter to clear.
        """
        logger.debug("clear_mentions uid=%d channel=%r", user_id, channel)
        if user_id in self.mentions and channel in self.mentions[user_id]:
            logger.info("Clearing mentions for uid=%d channel=%r", user_id, channel)
            del self.mentions[user_id][channel]
        else:
            logger.debug("No mentions to clear for uid=%d channel=%r", user_id, channel)

    def active_in(self: ChatCache, channel: str) -> list[dict[str, Any]]:
        """List active member descriptors for roster rendering.

        Args:
            channel: Channel whose active roster to list.

        Returns:
            List of dicts with id and nickname keys, sorted by user id.
        """
        logger.debug("active_in called for channel %r", channel)
        out: list[dict[str, Any]] = []
        ids_sorted: list[int] = sorted(self.members.get(channel, set()))
        logger.debug("Sorted member ids: %r", ids_sorted)
        for uid in ids_sorted:
            u: dict[str, Any] = self.users.get(uid, {})
            nick: str = u.get("nickname", "?")
            logger.debug("Roster entry uid=%d nick=%r", uid, nick)
            out.append({"id": uid, "nickname": nick})
        logger.info("active_in for %r returning %d users", channel, len(out))
        return out

    def seed(self: ChatCache, channel: str, messages: list[dict[str, Any]]) -> None:
        """Seed the recency deque from DB rows (last N only).

        Args:
            channel: Channel to seed.
            messages: Full message list to truncate and store.
        """
        logger.debug("Seeding channel %r with %d messages", channel, len(messages))
        dq: deque[dict[str, Any]] = self.recent[channel]
        dq.clear()
        logger.debug("Cleared existing recency buffer for %r", channel)
        tail: list[dict[str, Any]] = messages[-LAST_N:]
        logger.debug("Seeding last %d messages", len(tail))
        for m in tail:
            dq.append(m)
            logger.debug("Seeded message %r", m.get("id", "?"))
        logger.info("Seed complete for %r, size=%d", channel, len(dq))
