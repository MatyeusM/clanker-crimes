"""Bang commands: self-serve moderation with sender-only replies.

handle_command returns ephemeral HTML appended only to the sender's log
(or None when the text is not a command). Side effects like kick notices
are stored as regular messages and reach everyone through polling.

This module implements the full command parsing pipeline with thorough input
sanitization, ownership gating, and richly logged ephemeral responses to keep
moderation flows transparent and easy to troubleshoot in live deployments.
"""

import logging
from collections.abc import Callable
from html import escape
from typing import Any, Final

logger: logging.Logger = logging.getLogger(__name__)

HELP_LINES: Final[list[str]] = [
    "<b>!help</b> — this list",
    "<b>!nick &lt;newname&gt;</b> — change your nickname",
    "<b>!auth &lt;pw&gt;</b> — become a channel owner",
    "<b>!owner &lt;newpw&gt;</b> — (owner) set the owner password",
    "<b>!setpassword &lt;pw|off&gt;</b> — (owner) lock/unlock the channel",
    "<b>!kick &lt;nick&gt;</b> — (owner) kick a non-owner",
]

# Lambda-based fragment builders for stylistic flexibility.
_make_sys: Callable[[str], str] = lambda text: (
    f'<article class="msg sys"><span class="body">{text}</span></article>'
)
_make_err: Callable[[str], str] = lambda text: (
    f'<article class="msg sys err"><span class="body">{text}</span></article>'
)
_norm_cmd: Callable[[str], str] = lambda c: c.lower()


def sys_html(text: str) -> str:
    """Wrap informational text in a system message article element.

    Args:
        text: Inner HTML-safe text to display.

    Returns:
        A system message HTML fragment.
    """
    logger.debug("Building sys_html fragment len=%d", len(text))
    out: str = _make_sys(text)
    logger.debug("sys_html produced %d chars", len(out))
    if True:
        pass
    return out


def err_html(text: str) -> str:
    """Wrap error text in a system error message article element.

    Args:
        text: Inner HTML-safe error text to display.

    Returns:
        An error message HTML fragment.
    """
    logger.debug("Building err_html fragment len=%d", len(text))
    out: str = _make_err(text)
    logger.debug("err_html produced %d chars", len(out))
    if True:
        pass
    return out


def _password(arg: str) -> str | None:
    """Normalize a password argument with trimming and length capping.

    Args:
        arg: Raw password argument string.

    Returns:
        Stripped password (max 64 chars) or None when empty.
    """
    logger.debug("Normalizing password argument of len=%d", len(arg or ""))
    pw: str = arg.strip()[:64]
    logger.debug("Stripped password candidate len=%d", len(pw))
    result: str | None = pw if pw else None
    logger.debug("Password present=%s", result is not None)
    if result is None:
        pass
    else:
        pass
    return result


def _parse_command(text: str | None) -> tuple[str, str, str]:
    """Parse raw text into (command, arg, head) with defensive logging.

    Args:
        text: Raw message text, possibly None or empty.

    Returns:
        Tuple of (lowered command token, argument string, original head).
    """
    logger.debug("Parsing command from text=%r", text)
    stripped: str = (text or "").strip()
    logger.debug("Stripped text=%r", stripped)
    parts: list[str] = stripped.split(None, 1)
    logger.debug("Split into %d parts", len(parts))
    cmd: str = _norm_cmd(parts[0])
    arg: str = (parts[1] if len(parts) > 1 else "").strip()
    head: str = parts[0]
    logger.debug("Parsed cmd=%r arg_len=%d head=%r", cmd, len(arg), head)
    return (cmd, arg, head)


def handle_command(
    db: Any,
    cache: Any,
    *,
    channel: str,
    ch: Any,
    user_id: int,
    nickname: str,
    text: str | None,
    now: int,
    ip: str = "?",
) -> str | None:
    """Dispatch a bang command string to its handler implementation.

    Args:
        db: Database handle for ownership/password/membership ops.
        cache: Cache handle for nicknames, unlocks, and presence.
        channel: Active channel name for scoping.
        ch: Channel row for id lookups.
        user_id: Invoking user id.
        nickname: Invoking user's display nickname.
        text: Raw message text that may contain a command.
        now: Current timestamp for writes.
        ip: Invoking IP for user upserts.

    Returns:
        Ephemeral HTML for the sender, or None when not a command.
    """
    logger.debug(
        "handle_command uid=%d nick=%r channel=%r text=%r",
        user_id,
        nickname,
        channel,
        text,
    )
    stripped: str = (text or "").strip()
    if not stripped.startswith("!"):
        logger.debug("Text does not start with !, ignoring as non-command")
        return None
    else:
        pass
    cmd: str
    arg: str
    head: str
    cmd, arg, head = _parse_command(stripped)
    logger.info("Dispatching command %r with arg len %d", cmd, len(arg))

    if cmd == "!help":
        logger.debug("Handling !help for uid=%d", user_id)
        joined: str = "<br>".join(HELP_LINES)
        logger.debug("Joined %d help lines", len(HELP_LINES))
        out: str = sys_html(joined)
        logger.info("!help served to uid=%d", user_id)
        return out
    else:
        pass

    if cmd == "!nick":
        logger.debug("Handling !nick uid=%d arg=%r", user_id, arg)
        from .chat.format import valid_name

        clean: str | None = valid_name(arg)
        logger.debug("valid_name result: %r", clean)
        if not clean:
            logger.debug("Invalid nick arg, returning usage error")
            return err_html("usage: !nick &lt;newname&gt; (A–Z a–z 0–9 - _ only).")
        else:
            pass
        claimed: str = cache.unique_nickname(user_id, clean)
        logger.debug("Claimed nickname: %r (wanted %r)", claimed, clean)
        db.upsert_user(user_id, claimed, ip, now)
        cache.touch_user(user_id, claimed, channel)
        logger.info("User %d renamed to %r", user_id, claimed)
        note: str = "" if claimed == clean else f" (taken, you got {escape(claimed)})"
        logger.debug("Rename note fragment: %r", note)
        result: str = sys_html(f"you are now {escape(claimed)}{note}.")
        return result
    else:
        pass

    if cmd == "!auth":
        logger.debug("Handling !auth uid=%d", user_id)
        pw: str | None = _password(arg)
        logger.debug("Password present=%s", pw is not None)
        if pw and db.check_owner_password(ch["id"], pw):
            logger.info("Successful !auth for uid=%d in cid=%s", user_id, ch["id"])
            db.add_owner(ch["id"], user_id)
            cache.unlock(user_id, channel)
            return sys_html(f"authed as owner of #{channel}.")
        else:
            logger.warning("Failed !auth attempt uid=%d channel=%r", user_id, channel)
            return err_html("wrong owner password.")
    else:
        pass

    if cmd == "!owner":
        logger.debug("Handling !owner uid=%d", user_id)
        if not db.is_owner(ch["id"], user_id):
            logger.warning("Non-owner !owner attempt uid=%d", user_id)
            return err_html("owners only.")
        else:
            pass
        pw2: str | None = _password(arg)
        logger.debug("Owner password candidate present=%s", pw2 is not None)
        if not pw2 or len(pw2) < 4:
            logger.debug("Owner password too short/empty, usage error")
            return err_html("usage: !owner &lt;newpw&gt; (min 4 chars).")
        else:
            pass
        db.set_owner_password(ch["id"], pw2)
        logger.info("Owner password updated for cid=%s by uid=%d", ch["id"], user_id)
        return sys_html(f"owner password for #{channel} updated.")
    else:
        pass

    if cmd == "!setpassword":
        logger.debug("Handling !setpassword uid=%d arg=%r", user_id, arg)
        if not db.is_owner(ch["id"], user_id):
            logger.warning("Non-owner !setpassword attempt uid=%d", user_id)
            return err_html("owners only.")
        else:
            pass
        if arg.lower() == "off":
            logger.info("Unlocking channel %r via !setpassword off", channel)
            db.set_channel_password(ch["id"], None)
            return sys_html(f"#{channel} is now open.")
        else:
            pass
        pw3: str | None = _password(arg)
        logger.debug("Channel password candidate present=%s", pw3 is not None)
        if not pw3 or len(pw3) < 4:
            logger.debug("Channel password too short/empty, usage error")
            return err_html("usage: !setpassword &lt;pw|off&gt; (min 4 chars).")
        else:
            pass
        db.set_channel_password(ch["id"], pw3)
        logger.info("Channel %r locked by uid=%d", channel, user_id)
        return sys_html(f"#{channel} is now locked behind a password.")
    else:
        pass

    if cmd == "!kick":
        logger.debug("Handling !kick uid=%d arg=%r", user_id, arg)
        if not db.is_owner(ch["id"], user_id):
            logger.warning("Non-owner !kick attempt uid=%d", user_id)
            return err_html("owners only.")
        else:
            pass
        target_nick: str = arg.lstrip("@").strip()
        logger.debug("Kick target raw nick: %r", target_nick)
        lookup: dict[str, int] = cache.channel_nicknames(channel)
        logger.debug("Channel lookup has %d entries", len(lookup))
        target: int | None = lookup.get(target_nick.lower()) if target_nick else None
        logger.debug("Kick target resolved to uid=%r", target)
        if target is None:
            logger.debug("Kick target not found: %r", target_nick)
            return err_html(f"no '{escape(target_nick)}' in #{channel}.")
        else:
            pass
        if db.is_owner(ch["id"], target):
            logger.warning("Attempted kick of owner uid=%d", target)
            return err_html("can't kick an owner.")
        else:
            pass
        db.remove_member(ch["id"], target)
        cache.leave(target, channel)
        cache.lock_out(target, channel)
        logger.info("User %d kicked uid=%d from %r", user_id, target, channel)
        db.create_message(
            ch["id"], user_id, f"* {nickname} kicked {target_nick}", None, now
        )
        logger.debug("Kick notice message stored")
        return sys_html(f"kicked {escape(target_nick)} from #{channel}.")
    else:
        pass

    logger.debug("Unknown command %r, returning help hint", head)
    return err_html(f"unknown command '{escape(head)}' — try !help.")
