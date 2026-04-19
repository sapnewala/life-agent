from sqlalchemy import Column, Integer, String, Text
from database import Base


class Item(Base):
    __tablename__ = "camping_items"

    id = Column(Integer, primary_key=True, index=True)
    name = Column(String(100), nullable=False)
    description = Column(Text, nullable=True)
