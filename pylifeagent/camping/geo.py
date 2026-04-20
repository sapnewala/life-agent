import time
import cloudscraper
from camping.regions import REGIONS

_URL = "https://api.camfit.co.kr/v1/search/geo"
_HEADERS = {
    "origin": "https://camfit.co.kr",
    "referer": "https://camfit.co.kr/",
    "User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36",
}
_KOREA_BOUNDS = {
    "zoomLevel": "7",
    "leftTopLatitude": "38.7",
    "leftTopLongitude": "124.5",
    "rightBottomLatitude": "33.0",
    "rightBottomLongitude": "130.0",
}


def get_campsite_coords(
    cities: list[str] | None = None,
    adult: int = 2,
    teen: int = 2,
    only_available: bool = False,
    sleep: float = 0.5,
) -> list[dict]:
    """지역별 캠핑장 좌표 목록. cities=None이면 REGIONS 전체 순회."""
    targets = {c: m for c, m in REGIONS.items() if not cities or c in cities}

    scraper = cloudscraper.create_scraper()
    results = []

    for city, majors in targets.items():
        for major in majors:
            params = {
                "cities": city,
                "majors": major,
                "adult": str(adult),
                "teen": str(teen),
                "isSale": "false",
                "cityAndMajors": f"{city}-{major}",
                "isOnlyAvailable": str(only_available).lower(),
                "isLongTerm": "false",
                **_KOREA_BOUNDS,
            }
            response = scraper.get(_URL, headers=_HEADERS, params=params)
            response.raise_for_status()
            body = response.json()

            items = body if isinstance(body, list) else body.get("data", [])
            for item in items:
                item.setdefault("city", city)
                item.setdefault("major", major)
            results.extend(items)
            time.sleep(sleep)

    return results
