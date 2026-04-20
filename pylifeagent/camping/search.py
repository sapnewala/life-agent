import time
import cloudscraper
from utils import date_to_timestamp

_URL = "https://api.camfit.co.kr/v1/search"
_HEADERS = {
    "origin": "https://camfit.co.kr",
    "referer": "https://camfit.co.kr/",
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36",
}
_FIELDS = {"_id", "name", "brief", "city", "major", "numOfReviews", "numOfZones", "priceStartFrom"}


def search_campsites(
    start: str,
    end: str,
    city_and_majors: str = "경기-/강원-/인천-/서울-",
    types: str = "autoCamping",
    adult: int = 2,
    teen: int = 2,
    only_available: bool = True,
    limit: int = 10,
    max_try: int = 10,
) -> list[dict]:
    """캠핑장 검색. start/end는 'YYYYMMDD' 형식."""
    params = {
        "types": types,
        "adult": str(adult),
        "teen": str(teen),
        "startTimestamp": date_to_timestamp(start),
        "endTimestamp": date_to_timestamp(end),
        "isSale": "false",
        "cityAndMajors": city_and_majors,
        "isOnlyAvailable": str(only_available).lower(),
        "skip": "0",
        "limit": str(limit),
    }

    scraper = cloudscraper.create_scraper()
    results = []

    for _ in range(max_try):
        response = scraper.get(_URL, headers=_HEADERS, params=params)
        response.raise_for_status()
        body = response.json()

        items = [{k: v for k, v in item.items() if k in _FIELDS} for item in body.get("data", [])]
        results.extend(items)

        if not body.get("hasNext") or len(items) < limit:
            break

        params["skip"] = body.get("lastSkip", str(int(params["skip"]) + limit))
        time.sleep(1)

    return results
