"""Postgres engine + session management."""
from __future__ import annotations

from collections.abc import Generator
from contextlib import contextmanager

from sqlalchemy import create_engine
from sqlalchemy.orm import Session, sessionmaker

from .config import get_settings

_engine = None
_SessionLocal = None


def engine():
    global _engine
    if _engine is None:
        s = get_settings()
        # Force the psycopg v3 driver. Railway hands out either
        # postgres:// (Heroku-style) or postgresql:// (SQLAlchemy default
        # = psycopg2). We installed psycopg v3, so rewrite both.
        url = s.database_url
        if url.startswith("postgres://"):
            url = "postgresql+psycopg://" + url[len("postgres://"):]
        elif url.startswith("postgresql://") and "+psycopg" not in url:
            url = "postgresql+psycopg://" + url[len("postgresql://"):]
        _engine = create_engine(url, pool_pre_ping=True, pool_size=5, max_overflow=10)
    return _engine


def session_factory():
    global _SessionLocal
    if _SessionLocal is None:
        _SessionLocal = sessionmaker(bind=engine(), autoflush=False, autocommit=False)
    return _SessionLocal


@contextmanager
def db_session() -> Generator[Session, None, None]:
    """Context manager for ad-hoc DB work outside of FastAPI routes."""
    s = session_factory()()
    try:
        yield s
        s.commit()
    except Exception:
        s.rollback()
        raise
    finally:
        s.close()


def get_db() -> Generator[Session, None, None]:
    """FastAPI dependency."""
    s = session_factory()()
    try:
        yield s
    finally:
        s.close()
