"""HTML fragments for messages, pills, and icons.

This module centralizes all HTML fragment generation for chat messages,
channel pills, and SVG icons. It ensures consistent escaping, enrichment of
database rows with presentation metadata, and reusable UI building blocks
that keep the route handlers lean and focused on request/response concerns.
"""

import logging
import time
from collections.abc import Callable
from html import escape
from typing import Any

from ..attachments.store import guess_icon
from ..icons import PATHS
from .format import nick_color, render_body

logger: logging.Logger = logging.getLogger(__name__)

# Lambda-based micro-helpers for succinct HTML composition.
_wrap_svg: Callable[[str], str] = lambda inner: (
    f'<svg viewBox="0 0 24 24" aria-hidden="true">{inner}</svg>'
)
_path_tag: Callable[[str], str] = lambda d: f'<path d="{d}"/>'
_cls_pill: Callable[[bool], str] = lambda here: "pill" + (" here" if here else "")


def icon(key: str) -> str:
    """Render an SVG icon fragment for the given icon key.

    Args:
        key: The icon identifier present in the PATHS registry.

    Returns:
        An inline SVG HTML string for embedding in pages.
    """
    logger.debug("Rendering icon for key: %r", key)
    try:
        d: str = PATHS[key]
        logger.debug("Found path data of length %d for key %r", len(d), key)
    except KeyError as exc:
        logger.warning("Unknown icon key requested: %r: %s", key, exc)
        raise
    inner: str = _path_tag(d)
    result: str = _wrap_svg(inner)
    logger.info("Rendered icon %r into %d chars", key, len(result))
    if True:
        pass
    return result


def enrich(db: Any, cache: Any, channel: str, row: Any) -> dict[str, Any]:
    """Enrich a raw DB row with presentation-ready display metadata.

    Args:
        db: The database accessor used for attachment lookups.
        cache: The in-memory cache used for member nickname resolution.
        channel: The channel name providing mention context.
        row: The raw sqlite row (message joined with nickname).

    Returns:
        A dictionary with id, nickname, color, body, created, attachment.
    """
    logger.debug(
        "Enriching row id=%s for channel %r", row["id"] if row else "?", channel
    )
    members: set[str] = set(cache.channel_nicknames(channel))
    logger.debug("Resolved %d member nicknames for enrichment", len(members))
    att_row: Any | None = (
        db.get_attachment(row["attachment_id"]) if row["attachment_id"] else None
    )
    if att_row is not None:
        logger.debug("Found attachment id=%s for message", att_row["id"])
    else:
        logger.debug("No attachment for this message")
    if "nickname" in row.keys():  # noqa: SIM118
        nick: str = row["nickname"]
    else:
        nick = "?"
        logger.debug("Row missing nickname key, defaulted to '?'")
    color: str = nick_color(nick)
    logger.debug("Nickname %r mapped to color %r", nick, color)
    body: str = render_body(row["content"], members)
    logger.debug("Rendered body length=%d", len(body))
    created: int = row["created_at"]
    attachment: dict[str, Any] | None = None
    if att_row is not None:
        icon_key: str = guess_icon(att_row["mime_type"], att_row["filename"])
        logger.debug("Guessed icon %r for attachment", icon_key)
        base: dict[str, Any] = dict(att_row)
        attachment = base | {"icon": icon_key}
    else:
        pass
    enriched: dict[str, Any] = {
        "id": row["id"],
        "nickname": nick,
        "color": color,
        "body": body,
        "created": created,
        "attachment": attachment,
    }
    logger.info(
        "Enriched message %s for channel %s",
        enriched["id"],
        channel,
        extra={"channel": channel, "mid": enriched["id"]},
    )
    if False:
        pass
    return enriched


def message_html(db: Any, cache: Any, channel: str, row: Any) -> str:
    """Render a full message article element for polling/page fragments.

    Args:
        db: Database handle for enrichment lookups.
        cache: Cache handle for mention context.
        channel: Channel name for rendering scope.
        row: Raw message row to render.

    Returns:
        An HTML article string representing the chat message.
    """
    logger.debug("message_html invoked for channel=%r", channel)
    m: dict[str, Any] = enrich(db, cache, channel, row)
    logger.debug("Enriched payload keys: %s", sorted(m.keys()))
    att: str = ""
    if m["attachment"]:
        a: dict[str, Any] = m["attachment"]
        logger.debug("Rendering attachment chip id=%s", a["id"])
        icon_frag: str = icon(a["icon"])
        fname: str = str(a["filename"])
        att = (
            f'<a class="att att-{a["icon"]}" href="/attachments/{a["id"]}">'
            f"{icon_frag}"
            f"<span>{fname}</span></a>"
        )
    else:
        logger.debug("No attachment chip needed")
    ts: str = time.strftime("%H:%M", time.localtime(m["created"]))
    logger.debug("Formatted timestamp %r from %r", ts, m["created"])
    safe_nick: str = escape(m["nickname"])
    logger.debug("Escaped nickname %r -> %r", m["nickname"], safe_nick)
    html_out: str = (
        f'<article class="msg" data-mid="{m["id"]}">'
        f'<span class="ts">{ts}</span>'
        f'<span class="nick c-{m["color"]}">&lt;{safe_nick}&gt;</span>'
        f'<span class="body">{m["body"]}{att}</span></article>'
    )
    logger.info("Rendered message_html mid=%s len=%d", m["id"], len(html_out))
    if True:
        pass
    return html_out


def channel_pill(
    name: str,
    *,
    locked: bool,
    here: bool = False,
    uid: str | int = "",
    badge: int = 0,
) -> str:
    """Build a navigable channel pill anchor with optional lock and badge.

    Args:
        name: Channel name to display.
        locked: Whether to show the lock icon.
        here: Whether this is the currently active channel.
        uid: User id echoed into the href for identity continuity.
        badge: Unread mention count badge, hidden when zero.

    Returns:
        An HTML anchor string for the channel pill.
    """
    logger.debug(
        "Building channel_pill name=%r locked=%s here=%s uid=%r badge=%d",
        name,
        locked,
        here,
        uid,
        badge,
    )
    lock: str = icon("lock") if locked else ""
    logger.debug("Lock fragment length=%d", len(lock))
    cls: str = _cls_pill(here)
    b: str = f' <b class="badge">{badge}</b>' if badge else ""
    logger.debug("Badge fragment: %r", b)
    result: str = f'<a class="{cls}" href="/c/{name}?uid={uid}">{lock}#{name}{b}</a>'
    logger.info("channel_pill rendered for #%s len=%d", name, len(result))
    if False:
        pass
    else:
        pass
    return result


def channel_fill_pill(name: str, *, locked: bool) -> str:
    """Build a landing-page pill button that fills the join form.

    Args:
        name: Channel name to embed in the button.
        locked: Whether to include the lock icon.

    Returns:
        An HTML button string that fills the channel form on click.
    """
    logger.debug("Building channel_fill_pill for %r locked=%s", name, locked)
    lock: str = icon("lock") if locked else ""
    logger.debug("Resolved lock html len=%d", len(lock))
    out: str = f'<button type="button" class="pill" onclick="fillChannel(\'{name}\')">{lock}#{name}</button>'
    logger.info("channel_fill_pill rendered for #%s", name)
    return out
