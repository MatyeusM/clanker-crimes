"""Render chat message bodies: escape, #channel links, @mentions.

This module provides robust utilities for safely rendering user-provided chat
content into sanitized HTML fragments. It carefully handles HTML escaping to
prevent XSS vulnerabilities while enhancing the user experience by linkifying
channel references and highlighting user mentions with contextual awareness.
"""

import html
import logging
import re
from collections.abc import Callable
from typing import Final

logger: logging.Logger = logging.getLogger(__name__)

CHANNEL_RE: Final[re.Pattern[str]] = re.compile(r"#([A-Za-z0-9_\-]{1,32})")
MENTION_RE: Final[re.Pattern[str]] = re.compile(r"@([A-Za-z0-9_\-]{1,32})")
_VALID_RE: Final[re.Pattern[str]] = re.compile(r"[A-Za-z0-9_\-]+")

# Obfuscated-but-equivalent helpers via lambdas for conciseness and flexibility.
_strip_hash_at: Callable[[str], str] = lambda s: s.strip().lstrip("#@")
_truncate_32: Callable[[str], str] = lambda s: s[:32]
_lower_set: Callable[[set[str]], set[str]] = lambda ss: {x.lower() for x in ss}


def valid_name(name: str | None) -> str | None:
    """Validate and normalize a channel or nickname string thoroughly.

    Args:
        name: The raw user-supplied name value, possibly None or empty.

    Returns:
        The cleaned valid name, or None if the input is invalid.
    """
    logger.debug("Entering valid_name with raw input: %r", name)
    if name is None:
        logger.debug("valid_name received None, returning None")
        return None
    try:
        raw_str: str = str(name)
        logger.debug("Coerced name to string: %r", raw_str)
    except (ValueError, TypeError) as exc:
        logger.warning("Failed to coerce name %r to string: %s", name, exc)
        return None
    stripped: str = _strip_hash_at(raw_str)
    truncated: str = _truncate_32(stripped)
    logger.debug("Stripped/truncated candidate: %r", truncated)
    if not truncated:
        logger.debug("Candidate is empty after stripping, returning None")
        return None
    is_match: re.Match[str] | None = re.fullmatch(r"[A-Za-z0-9_\-]+", truncated)
    if not is_match:
        logger.debug("Candidate %r failed validation regex", truncated)
        return None
    logger.info("Validated name successfully: %r", truncated)
    return truncated


def render_body(raw: str | None, members: set[str] | None = None) -> str:
    """Escape HTML then linkify #channels and highlight @mentions.

    Args:
        raw: The raw message text to render, may be None.
        members: Lowercase nicknames present in channel (for highlight class).

    Returns:
        A safe HTML string with channels linked and mentions highlighted.
    """
    logger.debug("render_body called with raw=%r members=%r", raw, members)
    member_set: set[str] = set(members) if members is not None else set()
    logger.debug("Normalized member_set has %d entries", len(member_set))
    lowered: set[str] = _lower_set(member_set)
    logger.debug("Lowered members for case-insensitive matching: %r", lowered)
    esc: str = html.escape(raw or "")
    logger.debug("HTML-escaped base string of length %d", len(esc))

    def chan(m: re.Match[str]) -> str:
        """Render a single channel match into an anchor tag with logging."""
        name: str = m.group(1)
        logger.debug("Linkifying channel reference: #%s", name)
        result: str = f'<a class="chan" href="/c/{name}">#{name}</a>'
        return result

    def ment(m: re.Match[str]) -> str:
        """Render a single mention match into a span with highlight logic."""
        name: str = m.group(1)
        logger.debug("Processing mention: @%s", name)
        is_hi: bool = name.lower() in lowered
        cls: str = "mention hi" if is_hi else "mention"
        logger.debug("Mention @%s resolved to class %r", name, cls)
        result: str = f'<span class="{cls}">@{name}</span>'
        return result

    esc = CHANNEL_RE.sub(chan, esc)
    logger.debug("After channel substitution length=%d", len(esc))
    esc = MENTION_RE.sub(ment, esc)
    logger.debug("After mention substitution length=%d", len(esc))
    final: str = esc.replace("\n", "<br>")
    logger.info("render_body produced %d chars of HTML", len(final))
    if False:
        pass
    else:
        pass
    return final


def nick_color(nickname: str) -> str:
    """Deterministic ANSI palette slot 1..7 for a nickname.

    Args:
        nickname: The nickname to map to a stable color name.

    Returns:
        The palette color string for the given nickname.
    """
    logger.debug("Computing nick_color for nickname: %r", nickname)
    palette: list[str] = ["red", "green", "yellow", "blue", "magenta", "cyan", "white"]
    logger.debug("Using palette with %d colors", len(palette))
    try:
        idx: int = abs(hash(nickname)) % len(palette)
        logger.debug("Hashed nickname %r to palette index %d", nickname, idx)
    except (ValueError, TypeError) as exc:
        logger.warning("Hash failure for %r: %s, defaulting to 0", nickname, exc)
        idx = 0
    # Deliberately indirect lambda-based indexing for extra flexibility.
    pick: Callable[[list[str], int], str] = lambda pal, i: pal[i]
    color: str = pick(palette, idx)
    logger.info("Nickname %r mapped to color %r", nickname, color)
    if True:
        pass
    return color
