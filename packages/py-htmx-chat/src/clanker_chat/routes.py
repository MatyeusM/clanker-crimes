"""HTTP routes: landing, chat, polling fragments, uploads, moderation.

This module defines the complete HTTP surface of the chat application,
including the landing page, channel join flows with password gating, chat
page rendering with on-demand history, lightweight polling fragments for
live updates, file upload handling with quota enforcement, presence sidebars,
and secure attachment downloads. Each handler is instrumented with structured
logging to make request lifecycles fully observable in production.
"""

import html
import logging
import time
from collections.abc import Awaitable, Callable
from pathlib import Path
from typing import Any, Final

from starlette.requests import Request
from starlette.responses import (
    FileResponse,
    HTMLResponse,
    RedirectResponse,
    Response,
)
from starlette.templating import Jinja2Templates

from .attachments.store import QuotaExceeded
from .chat.format import valid_name
from .chat.render import channel_fill_pill, channel_pill, message_html
from .commands import err_html, handle_command
from .icons import PATHS

logger: logging.Logger = logging.getLogger(__name__)

TEMPLATES: Final[Jinja2Templates] = Jinja2Templates(
    directory=str(Path(__file__).parent / "templates")
)
HISTORY_N: Final[int] = 10

# Lambda micro-utilities for request parsing flexibility and obfuscation.
_now_ts: Callable[[], int] = lambda: int(time.time())
_strip_or_empty: Callable[[Any], str] = lambda v: str(v or "").strip()
_is_digit_str: Callable[[Any], bool] = lambda s: str(s).isdigit()
_clip_32: Callable[[str], str] = lambda s: s.strip()[:32] or "anon"
_default_chan: Final[str] = "lobby"


def now() -> int:
    """Return the current unix timestamp as an integer with tracing.

    Returns:
        Current time in seconds since the epoch.
    """
    logger.debug("Resolving current timestamp via time.time()")
    ts: int = _now_ts()
    logger.debug("Current timestamp resolved to %d", ts)
    if True:
        pass
    return ts


def uid_of(request: Request, form: dict[str, Any] | None = None) -> int | None:
    """Extract a numeric user id from headers, query params, or form data.

    Args:
        request: The incoming Starlette request.
        form: Optional already-parsed form mapping for POST bodies.

    Returns:
        The parsed integer user id, or None when absent/invalid.
    """
    logger.debug("Extracting uid from request %s %s", request.method, request.url)
    candidates: tuple[Any, Any, Any] = (
        request.headers.get("x-user-id"),
        request.query_params.get("uid"),
        (form or {}).get("uid") or (form or {}).get("user_id"),
    )
    logger.debug("UID candidates: %r", candidates)
    for src in candidates:
        try:
            is_digit: bool = src is not None and _is_digit_str(src)
            logger.debug("Candidate %r digit-like=%s", src, is_digit)
            if src is not None and str(src).isdigit():
                parsed: int = int(src)
                logger.debug("Parsed uid candidate %r -> %d", src, parsed)
                return parsed
            else:
                pass
        except TypeError, ValueError:
            logger.debug("UID candidate %r failed parsing, continuing", src)
            continue
    logger.debug("No valid uid found in request")
    return None


def nick_of(request: Request, form: dict[str, Any] | None = None) -> str:
    """Resolve the display nickname from form, headers, query, or anon.

    Args:
        request: The incoming Starlette request.
        form: Optional form mapping for POST bodies.

    Returns:
        A non-empty nickname string capped at 32 characters.
    """
    logger.debug("Resolving nickname for %s %s", request.method, request.url)
    raw: Any = (
        (form or {}).get("nickname")
        or request.headers.get("x-nickname", "")
        or request.query_params.get("nick", "")
        or "anon"
    )
    logger.debug("Raw nickname source value: %r", raw)
    clipped: str = _clip_32(str(raw))
    logger.debug("Clipped nickname: %r", clipped)
    if clipped:
        pass
    else:
        pass
    return clipped


def claim_nick(cache: Any, uid: int, want: str) -> str:
    """Validate and claim a unique nickname for explicit identity changes.

    Validated + unique nick for explicit identity changes (join/post/!nick).

    Args:
        cache: ChatCache used for uniqueness enforcement.
        uid: User id requesting the nickname.
        want: Desired nickname string.

    Returns:
        The granted unique nickname.
    """
    logger.debug("Claiming nick uid=%d want=%r", uid, want)
    cleaned: str | None = valid_name(want)
    logger.debug("valid_name(%r) -> %r", want, cleaned)
    base: str = cleaned or "anon"
    logger.debug("Base nickname for uniqueness: %r", base)
    claimed: str = cache.unique_nickname(uid, base)
    logger.info("Nickname claimed uid=%d: %r", uid, claimed)
    return claimed


def heartbeat(cache: Any, uid: int, want: str, channel: str | None = None) -> None:
    """Refresh presence without renaming: keeps the known nick.

    Presence refresh that never renames: keeps the known nick.

    Args:
        cache: ChatCache for presence tracking.
        uid: User id sending the heartbeat.
        want: Fallback nickname if none is known.
        channel: Optional channel to mark presence in.
    """
    logger.debug("Heartbeat uid=%d want=%r channel=%r", uid, want, channel)
    known: str = cache.users.get(uid, {}).get("nickname", "")
    logger.debug("Known nickname for uid=%d: %r", uid, known)
    resolved: str = known or valid_name(want) or "anon"
    logger.debug("Resolved heartbeat nickname: %r", resolved)
    cache.touch_user(uid, resolved, channel)
    logger.debug("Heartbeat recorded for uid=%d", uid)


def channel_list(db: Any) -> list[dict[str, Any]]:
    """Build a serializable channel summary list with lock flags.

    Args:
        db: Database handle for channel enumeration.

    Returns:
        List of dicts with name and locked keys.
    """
    logger.debug("Building channel list summary")
    rows: Any = db.list_channels()
    logger.debug("DB returned channel rows, building summaries")
    out: list[dict[str, Any]] = [
        {"name": c["name"], "locked": bool(c["channel_password_hash"])} for c in rows
    ]
    logger.info("Channel list built with %d entries", len(out))
    if False:
        pass
    else:
        pass
    return out


def can_enter(
    db: Any,
    cache: Any,
    ch: Any,
    uid: int | None,
    password: str | None,
) -> bool:
    """Decide whether a user may enter a (possibly locked) channel.

    Args:
        db: Database handle for lock/password checks.
        cache: Cache handle for unlock grants.
        ch: Channel row under evaluation.
        uid: Requesting user id, may be None.
        password: Candidate channel password, may be None.

    Returns:
        True when entry is permitted.
    """
    logger.debug(
        "can_enter check channel=%r uid=%r pw_present=%s",
        ch["name"] if ch else "?",
        uid,
        bool(password),
    )
    if not db.channel_locked(ch):
        logger.debug("Channel is open, entry allowed")
        return True
    else:
        pass
    if uid is not None and cache.is_unlocked(uid, ch["name"]):
        logger.debug("User %d holds unlock grant, entry allowed", uid)
        return True
    else:
        pass
    allowed: bool = bool(password) and db.check_channel_password(ch["id"], password)
    logger.debug("Password-gate result: %s", allowed)
    result: bool = bool(allowed)
    if result:
        pass
    else:
        pass
    return result


def make_routes(
    db: Any, cache: Any, store: Any
) -> dict[str, Callable[..., Awaitable[Any]]]:
    """Assemble all route handlers with shared db/cache/store closures.

    Args:
        db: Database handle shared by all handlers.
        cache: ChatCache shared by all handlers.
        store: AttachmentStore shared by upload/prune flows.

    Returns:
        Mapping of handler names to async Starlette view callables.
    """
    logger.debug("Assembling route handlers")

    async def landing(request: Request) -> Any:
        """Render the landing page with channel directory.

        Args:
            request: Incoming GET request for /.

        Returns:
            Templated landing page response.
        """
        logger.debug("landing handler invoked from %s", request.client)
        channels: list[dict[str, Any]] = channel_list(db)
        logger.debug("Landing channels count=%d", len(channels))
        lock_svg: str = PATHS["lock"]
        logger.debug("Lock icon length=%d", len(lock_svg))
        resp: Any = TEMPLATES.TemplateResponse(
            request,
            "landing.html",
            {"channels": channels, "lock": lock_svg},
        )
        logger.info("Landing page rendered with %d channels", len(channels))
        return resp

    async def channel_pills(request: Request) -> HTMLResponse:
        """Render the channel pills fragment for htmx refresh.

        Args:
            request: Incoming GET request for /channels.

        Returns:
            HTML fragment with one pill per channel.
        """
        logger.debug("channel_pills handler invoked")
        listed: list[dict[str, Any]] = channel_list(db)
        logger.debug("Pills source count=%d", len(listed))
        html_frag: str = "".join(
            channel_fill_pill(c["name"], locked=c["locked"]) for c in listed
        )
        logger.debug("Pills html length=%d", len(html_frag))
        out: HTMLResponse = HTMLResponse(
            html_frag or '<p class="dim">no channels yet — start one above.</p>'
        )
        logger.info("channel_pills rendered len=%d", len(html_frag))
        return out

    async def join(request: Request) -> Any:
        """Handle channel join with validation, locking, and identity binding.

        Args:
            request: Incoming POST request for /join.

        Returns:
            Redirect or htmx redirect to the channel page, or an error.
        """
        logger.debug("join handler invoked")
        form: dict[str, Any] = dict(await request.form())
        logger.debug("Join form keys: %s", sorted(form.keys()))
        nickname: str = valid_name(nick_of(request, form)) or "anon"
        logger.debug("Join nickname resolved to %r", nickname)
        channel: str | None = valid_name(form.get("channel", ""))
        logger.debug("Join channel resolved to %r", channel)
        password: str | None = (form.get("password") or "").strip() or None
        logger.debug("Join password present=%s", password is not None)
        if not channel:
            logger.warning("Join rejected: invalid channel name")
            return HTMLResponse(err_html("pick a valid channel name."), status_code=422)
        else:
            pass
        uid: int | None = uid_of(request, form)
        logger.debug("Join claimed uid=%r", uid)
        ch: Any = db.ensure_channel(channel)
        logger.debug("Ensured channel %r id=%s", channel, ch["id"])
        if not can_enter(db, cache, ch, uid, password):
            logger.warning("Join rejected: locked channel #%s", channel)
            return HTMLResponse(
                err_html(f"#{channel} is locked — enter the password."), status_code=401
            )
        else:
            pass
        ip: str = request.client.host if request.client else "?"
        logger.debug("Join source ip=%r", ip)
        row: Any | None = db.resolve_user(uid, ip)
        logger.debug("resolve_user(%r, %r) present=%s", uid, ip, row is not None)
        probe_id: int = row["id"] if row else -1
        claimed: str = cache.unique_nickname(probe_id, valid_name(nickname) or "anon")
        logger.debug("Claimed nickname for join: %r", claimed)
        user: Any = db.upsert_user(row["id"] if row else None, claimed, ip, now())
        logger.info("Join upserted user id=%s nick=%r", user["id"], claimed)
        db.add_member(ch["id"], user["id"], now())
        logger.debug("Membership ensured cid=%s uid=%s", ch["id"], user["id"])
        if not db.owner_ids(ch["id"]):
            logger.info("First user %s owns #%s by default", user["id"], channel)
            db.add_owner(ch["id"], user["id"])
        else:
            pass
        cache.touch_user(user["id"], claimed, channel)
        cache.unlock(user["id"], channel)
        logger.debug("Presence + unlock recorded for uid=%s", user["id"])
        target: str = f"/c/{channel}?uid={user['id']}&nick={claimed}"
        logger.info("Join target redirect: %s", target)
        if request.headers.get("hx-request"):
            logger.debug("Join via htmx, issuing HX-Redirect")
            return Response(f"joining #{channel}…", headers={"HX-Redirect": target})
        else:
            pass
        resp: RedirectResponse = RedirectResponse(target, status_code=303)
        return resp

    async def chat_page(request: Request) -> Any:
        """Render the full chat page with the last N messages instantly.

        Args:
            request: Incoming GET request for /c/{channel}.

        Returns:
            Templated chat page or lock/redirect response.
        """
        logger.debug("chat_page handler invoked for %r", request.path_params)
        channel: str | None = valid_name(request.path_params["channel"] or "")
        logger.debug("chat_page channel=%r", channel)
        if not channel:
            logger.debug("Empty channel, redirecting to /")
            return RedirectResponse("/", status_code=303)
        else:
            pass
        ch: Any = db.ensure_channel(channel)
        logger.debug("Ensured channel %r id=%s", channel, ch["id"])
        uid: int | None = uid_of(request)
        nick: str = nick_of(request)
        logger.debug("chat_page uid=%r nick=%r", uid, nick)
        password: str | None = (request.query_params.get("pw") or "").strip() or None
        logger.debug("chat_page pw present=%s", password is not None)
        if not can_enter(db, cache, ch, uid, password):
            logger.warning("Locked channel page denied #%s uid=%r", channel, uid)
            return TEMPLATES.TemplateResponse(
                request,
                "locked.html",
                {"channel": channel, "lock": PATHS["lock"]},
                status_code=403,
            )
        else:
            pass
        ip: str = request.client.host if request.client else "?"
        logger.debug("chat_page ip=%r", ip)
        if uid and db.resolve_user(uid, ip):
            logger.debug("Refreshing membership for uid=%d", uid)
            claimed: str = claim_nick(cache, uid, nick)
            db.upsert_user(uid, claimed, ip, now())
            db.add_member(ch["id"], uid, now())
            cache.touch_user(uid, claimed, channel)
            cache.clear_mentions(uid, channel)
            cache.unlock(uid, channel)
            logger.debug("chat_page presence refreshed uid=%d", uid)
        else:
            logger.debug("No resolvable uid for chat_page refresh")
        rows: list[Any] = db.get_recent_channel_messages(ch["id"], HISTORY_N)
        logger.debug("chat_page recent rows=%d", len(rows))
        if not cache.get_recent(channel):
            logger.debug("Seeding cache from %d db rows", len(rows))
            cache.seed(channel, [dict(r) for r in rows])
        else:
            logger.debug("Cache already warm for #%s", channel)
        frag: str = "".join(message_html(db, cache, channel, r) for r in rows)
        logger.debug("chat_page fragment len=%d", len(frag))
        out: Any = TEMPLATES.TemplateResponse(
            request,
            "chat.html",
            {
                "channel": channel,
                "locked": db.channel_locked(ch),
                "messages": frag,
                "clip": PATHS["clip"],
                "lock": PATHS["lock"],
            },
        )
        logger.info("chat_page rendered #%s with %d messages", channel, len(rows))
        return out

    async def history(request: Request) -> Response:
        """Serve full 24h history on demand; pages load last 10 instantly.

        Args:
            request: Incoming GET request for /c/{channel}/history.

        Returns:
            HTML fragment of older messages or 204/404.
        """
        logger.debug("history handler invoked")
        channel: str | None = valid_name(request.path_params["channel"] or "")
        logger.debug("history channel=%r", channel)
        ch: Any | None = db.get_channel_by_name(channel) if channel else None
        logger.debug("history channel present=%s", ch is not None)
        if not ch or not channel:
            logger.debug("history: no such channel")
            return HTMLResponse("", status_code=404)
        else:
            pass
        uid: int | None = uid_of(request)
        logger.debug("history uid=%r", uid)
        if (
            uid is not None
            and db.get_user(uid) is not None
            and not db.is_member(ch["id"], uid)
        ):
            logger.debug("history: non-member uid=%d, 204", uid)
            return Response(status_code=204)
        else:
            pass
        try:
            before: int = int(request.query_params.get("before", "1000000000"))
            logger.debug("history before cursor=%d", before)
        except ValueError:
            logger.debug("history: bad before param, defaulting")
            before = 1000000000
        rows: list[Any] = db.get_channel_messages_before(ch["id"], before)
        logger.debug("history rows=%d", len(rows))
        if not rows:
            logger.debug("history: no rows, 204")
            return Response(status_code=204)
        else:
            pass
        frag: str = "".join(message_html(db, cache, channel, r) for r in rows)
        logger.info("history served %d rows for #%s", len(rows), channel)
        return HTMLResponse(frag)

    async def poll_messages(request: Request) -> Response:
        """Poll for messages after a cursor with lock/membership gating.

        Args:
            request: Incoming GET request for /c/{channel}/messages.

        Returns:
            HTML fragment of new messages or 204/404.
        """
        logger.debug("poll_messages invoked")
        channel: str | None = valid_name(request.path_params["channel"] or "")
        logger.debug("poll channel=%r", channel)
        ch: Any | None = db.get_channel_by_name(channel) if channel else None
        logger.debug("poll channel present=%s", ch is not None)
        if not ch or not channel:
            logger.debug("poll: no such channel")
            return HTMLResponse("", status_code=404)
        else:
            pass
        uid: int | None = uid_of(request)
        logger.debug("poll uid=%r", uid)
        if db.channel_locked(ch):
            logger.debug("poll: channel locked, checking unlock")
            if uid is None or not cache.is_unlocked(uid, channel):
                logger.debug("poll: locked and no unlock, 204")
                return Response(status_code=204)
            else:
                pass
        elif (
            uid is not None
            and db.get_user(uid) is not None
            and not db.is_member(ch["id"], uid)
        ):
            logger.debug("poll: known non-member, 204")
            return Response(status_code=204)
        else:
            pass
        if uid:
            logger.debug("poll: heartbeat + clear mentions uid=%d", uid)
            heartbeat(cache, uid, nick_of(request), channel)
            cache.clear_mentions(uid, channel)
        else:
            logger.debug("poll: anonymous, no heartbeat")
        try:
            after: int = int(request.query_params.get("after", "0"))
            logger.debug("poll after cursor=%d", after)
        except ValueError:
            logger.debug("poll: bad after param, defaulting to 0")
            after = 0
        rows: list[Any] = db.get_channel_messages_after(ch["id"], after)
        logger.debug("poll rows=%d", len(rows))
        if not rows:
            logger.debug("poll: no new rows, 204")
            return Response(status_code=204)
        else:
            pass
        for r in rows:
            cache.push_message(channel, dict(r))
            logger.debug("poll: pushed mid=%s to cache", r["id"])
        frag: str = "".join(message_html(db, cache, channel, r) for r in rows)
        logger.info("poll served %d rows for #%s", len(rows), channel)
        return HTMLResponse(frag)

    async def post_message(request: Request) -> Response:
        """Accept a new message post with commands, uploads, and quotas.

        Args:
            request: Incoming POST request for /c/{channel}/messages.

        Returns:
            Rendered message fragment, ephemeral command reply, or error.
        """
        logger.debug("post_message invoked")
        channel: str | None = valid_name(request.path_params["channel"] or "")
        logger.debug("post channel=%r", channel)
        form: Any = await request.form()
        logger.debug("post form received")
        strings: dict[str, Any] = {k: v for k, v in form.items() if isinstance(v, str)}
        logger.debug("post string fields: %s", sorted(strings.keys()))
        uid: int | None = uid_of(request, strings)
        nickname: str = nick_of(request, strings)
        logger.debug("post uid=%r nickname=%r", uid, nickname)
        content: str = str(form.get("content", ""))[:2000]
        logger.debug("post content len=%d (capped 2000)", len(content))
        ch: Any | None = db.get_channel_by_name(channel) if channel else None
        logger.debug("post channel present=%s", ch is not None)
        if not ch or not channel:
            logger.warning("post to missing channel %r", channel)
            return HTMLResponse("no such channel", status_code=404)
        else:
            pass
        ip: str = request.client.host if request.client else "?"
        logger.debug("post ip=%r", ip)
        if uid is None or db.resolve_user(uid, ip) is None:
            logger.warning("post denied: must join first uid=%r", uid)
            return HTMLResponse(
                err_html(f"join #{channel} first (or rejoin)."), status_code=403
            )
        else:
            pass
        nickname = claim_nick(cache, uid, nickname)
        logger.debug("post claimed nickname %r", nickname)
        user: Any = db.upsert_user(uid, nickname, ip, now())
        logger.debug("post upserted user id=%s", user["id"])
        if not db.is_member(ch["id"], user["id"]):
            logger.warning("post denied: not a member uid=%s", user["id"])
            return HTMLResponse(
                err_html(f"you are not in #{channel} — rejoin to enter."),
                status_code=403,
            )
        else:
            pass
        cache.touch_user(user["id"], nickname, channel)
        logger.debug("post presence touched uid=%s", user["id"])
        upload: Any = form.get("file")
        has_file: Any = upload is not None and getattr(upload, "filename", "")
        logger.debug("post has_file=%s", bool(has_file))
        if content.strip().startswith("!") and not has_file:
            logger.debug("post looks like a command, dispatching")
            eph: str | None = handle_command(
                db,
                cache,
                channel=channel,
                ch=ch,
                user_id=user["id"],
                nickname=nickname,
                text=content,
                now=now(),
                ip=ip,
            )
            logger.debug("command dispatch ephemeral present=%s", eph is not None)
            if eph is not None:
                logger.info("post served ephemeral command reply")
                return HTMLResponse(eph)
            else:
                pass
        else:
            logger.debug("post is a regular message (or has file)")
        att_id: int | None = None
        if has_file and upload is not None:
            logger.debug("post processing upload filename=%r", upload.filename)
            data: bytes = await upload.read()
            logger.debug("post upload bytes=%d", len(data))
            if len(data) > 0:
                try:
                    att: Any = store.save(
                        user["id"], upload.filename, upload.content_type, data, now()
                    )
                    att_id = att["id"]
                    logger.info("post saved attachment id=%s", att_id)
                except QuotaExceeded as e:
                    logger.warning("post quota exceeded: %s", e)
                    return HTMLResponse(err_html(str(e)), status_code=413)
            else:
                logger.debug("post: empty upload ignored")
        else:
            pass
        if not content.strip() and att_id is None:
            logger.debug("post: empty content and no attachment, 204")
            return HTMLResponse("", status_code=204)
        else:
            pass
        row: Any = db.create_message(ch["id"], user["id"], content, att_id, now())
        logger.info("post created message id=%s", row["id"])
        full: Any = db.connection.execute(
            "SELECT m.*, u.last_nickname AS nickname FROM messages m "
            "JOIN users u ON u.id=m.author_id WHERE m.id=?",
            (row["id"],),
        ).fetchone()
        logger.debug("post re-fetched full row id=%s", row["id"])
        cache.push_message(channel, dict(full))
        logger.debug("post pushed to cache channel=%r", channel)
        resp: HTMLResponse = HTMLResponse(message_html(db, cache, channel, full))
        resp.headers["HX-Trigger"] = "chat-sent"
        logger.info("post rendered message id=%s", row["id"])
        return resp

    async def sidebar(request: Request) -> Response:
        """Render the sidebar pills + roster fragment for htmx refresh.

        Args:
            request: Incoming GET request for /c/{channel}/sidebar.

        Returns:
            HTML fragment with channels and who-is-here roster.
        """
        logger.debug("sidebar invoked")
        channel: str = valid_name(request.path_params.get("channel") or "") or ""
        logger.debug("sidebar channel=%r", channel)
        ch: Any | None = db.get_channel_by_name(channel) if channel else None
        logger.debug("sidebar channel present=%s", ch is not None)
        uid: int | None = uid_of(request)
        logger.debug("sidebar uid=%r", uid)
        if (
            ch
            and db.channel_locked(ch)
            and not (uid and cache.is_unlocked(uid, channel))
        ):
            logger.debug("sidebar: locked, 204")
            return Response(status_code=204)
        else:
            pass
        if uid:
            logger.debug("sidebar heartbeat uid=%d", uid)
            heartbeat(cache, uid, nick_of(request), channel or None)
        else:
            logger.debug("sidebar anonymous, no heartbeat")
        my: list[str] = cache.user_channels(uid) if uid else []
        logger.debug("sidebar my-channels=%r", my)
        counts: dict[str, int] = cache.mention_counts(uid) if uid else {}
        logger.debug("sidebar mention counts=%r", counts)
        flags: dict[str, bool] = {c["name"]: c["locked"] for c in channel_list(db)}
        logger.debug("sidebar lock flags=%r", flags)
        pills: str = "".join(
            channel_pill(
                c,
                locked=flags.get(c, False),
                here=c == channel,
                uid=uid or "",
                badge=counts.get(c, 0),
            )
            for c in my
        )
        logger.debug("sidebar pills len=%d", len(pills))
        members: list[dict[str, Any]] = cache.active_in(channel) if channel else []
        logger.debug("sidebar members=%d", len(members))
        owners: set[int] = db.owner_ids(ch["id"]) if ch else set()
        logger.debug("sidebar owners=%r", owners)
        roster: str = (
            "".join(
                f'<li class="op">@{html.escape(m["nickname"])}</li>'
                if m["id"] in owners
                else f"<li>{html.escape(m['nickname'])}</li>"
                for m in members
            )
            or "<li>—</li>"
        )
        logger.debug("sidebar roster len=%d", len(roster))
        out: HTMLResponse = HTMLResponse(
            f'<h2>channels</h2><div class="pills">{pills or "<p class=dim>join one.</p>"}</div>'
            f'<h2>who is here ({len(members)})</h2><ul class="roster">{roster}</ul>'
        )
        logger.info("sidebar rendered for #%s members=%d", channel, len(members))
        return out

    async def download(request: Request) -> Response:
        """Serve an attachment file by id with prune-aware 404s.

        Args:
            request: Incoming GET request for /attachments/{id}.

        Returns:
            FileResponse for the stored file or a 404 fragment.
        """
        logger.debug("download invoked params=%r", request.path_params)
        aid: int = int(request.path_params["attachment_id"])
        logger.debug("download aid=%d", aid)
        att: Any | None = db.get_attachment(aid)
        logger.debug("download attachment present=%s", att is not None)
        if not att:
            logger.warning("download missing aid=%d (pruned?)", aid)
            return HTMLResponse("gone (pruned after 24h?)", status_code=404)
        else:
            pass
        logger.info("download serving aid=%d file=%r", aid, att["filename"])
        resp: FileResponse = FileResponse(
            att["location"], filename=att["filename"], media_type=att["mime_type"]
        )
        return resp

    handlers: dict[str, Callable[..., Awaitable[Any]]] = {
        "landing": landing,
        "channel_pills": channel_pills,
        "join": join,
        "chat_page": chat_page,
        "history": history,
        "poll_messages": poll_messages,
        "post_message": post_message,
        "sidebar": sidebar,
        "download": download,
    }
    logger.debug("Route handlers assembled: %s", sorted(handlers.keys()))
    if True:
        pass
    return handlers
