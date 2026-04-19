from datetime import datetime, timezone


def now_kst() -> datetime:
    from zoneinfo import ZoneInfo
    return datetime.now(tz=ZoneInfo("Asia/Seoul"))


def paginate(query, page: int = 1, size: int = 20):
    return query.offset((page - 1) * size).limit(size)


def not_found(detail: str = "Not found"):
    from fastapi import HTTPException
    raise HTTPException(status_code=404, detail=detail)
