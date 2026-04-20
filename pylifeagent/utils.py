from datetime import datetime, timezone


def now_kst() -> datetime:
    from zoneinfo import ZoneInfo
    return datetime.now(tz=ZoneInfo("Asia/Seoul"))


def date_to_timestamp(date_str: str) -> str:
    """'YYYYMMDD' 문자열을 KST 자정 기준 Unix 타임스탬프(ms)로 변환."""
    from zoneinfo import ZoneInfo
    dt = datetime.strptime(date_str, "%Y%m%d").replace(tzinfo=ZoneInfo("Asia/Seoul"))
    return str(int(dt.timestamp() * 1000))


def paginate(query, page: int = 1, size: int = 20):
    return query.offset((page - 1) * size).limit(size)


def not_found(detail: str = "Not found"):
    from fastapi import HTTPException
    raise HTTPException(status_code=404, detail=detail)
