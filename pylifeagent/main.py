from fastapi import FastAPI
from database import Base, engine
from camping.router import router as camping_router

Base.metadata.create_all(bind=engine)

app = FastAPI(title="pylifeagent")

app.include_router(camping_router, prefix="/camping", tags=["camping"])


@app.get("/health")
def health():
    return {"status": "ok"}
