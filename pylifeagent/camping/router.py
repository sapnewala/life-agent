from fastapi import APIRouter, Depends, HTTPException, Query
from pydantic import BaseModel
from sqlalchemy.orm import Session
from database import get_db
from camping.models import Item
from camping.search import search_campsites
from camping.geo import get_campsite_coords

router = APIRouter()


class ItemIn(BaseModel):
    name: str
    description: str | None = None


class ItemOut(ItemIn):
    id: int
    class Config:
        from_attributes = True


@router.get("/items", response_model=list[ItemOut])
def list_items(db: Session = Depends(get_db)):
    return db.query(Item).all()


@router.get("/items/{item_id}", response_model=ItemOut)
def get_item(item_id: int, db: Session = Depends(get_db)):
    item = db.get(Item, item_id)
    if not item:
        raise HTTPException(status_code=404, detail="Not found")
    return item


@router.post("/items", response_model=ItemOut, status_code=201)
def create_item(payload: ItemIn, db: Session = Depends(get_db)):
    item = Item(**payload.model_dump())
    db.add(item)
    db.commit()
    db.refresh(item)
    return item


@router.delete("/items/{item_id}", status_code=204)
def delete_item(item_id: int, db: Session = Depends(get_db)):
    item = db.get(Item, item_id)
    if not item:
        raise HTTPException(status_code=404, detail="Not found")
    db.delete(item)
    db.commit()


@router.get("/search")
def search(
    start: str = Query(..., description="시작일 YYYYMMDD"),
    end: str = Query(..., description="종료일 YYYYMMDD"),
    city_and_majors: str = Query("경기-/강원-/인천-/서울-"),
    types: str = Query("autoCamping"),
    adult: int = Query(2),
    teen: int = Query(2),
    only_available: bool = Query(True),
    limit: int = Query(10),
):
    return search_campsites(
        start=start,
        end=end,
        city_and_majors=city_and_majors,
        types=types,
        adult=adult,
        teen=teen,
        only_available=only_available,
        limit=limit,
    )


@router.get("/geo")
def geo(
    cities: str = Query(None, description="조회할 도시 목록, 콤마 구분 (예: 경기,강원). 미입력시 전체"),
    adult: int = Query(2),
    teen: int = Query(2),
    only_available: bool = Query(False),
):
    city_list = [c.strip() for c in cities.split(",")] if cities else None
    return get_campsite_coords(
        cities=city_list,
        adult=adult,
        teen=teen,
        only_available=only_available,
    )
