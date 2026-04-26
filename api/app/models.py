"""SQLAlchemy ORM models. Schema lives in schema.sql; these are the
read/write mappings."""
from __future__ import annotations

from datetime import datetime

from sqlalchemy import BigInteger, Boolean, DateTime, ForeignKey, String, UniqueConstraint
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    pass


class StripeEvent(Base):
    __tablename__ = "stripe_events"

    event_id: Mapped[str] = mapped_column(String, primary_key=True)
    event_type: Mapped[str] = mapped_column(String)
    received_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))


class Customer(Base):
    __tablename__ = "customers"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    stripe_customer_id: Mapped[str] = mapped_column(String, unique=True)
    email: Mapped[str] = mapped_column(String)
    api_key_hash: Mapped[str] = mapped_column(String, unique=True)
    api_key_prefix: Mapped[str] = mapped_column(String)
    tier: Mapped[str] = mapped_column(String)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))

    subscriptions: Mapped[list["Subscription"]] = relationship(back_populates="customer")
    usages: Mapped[list["Usage"]] = relationship(back_populates="customer")


class Subscription(Base):
    __tablename__ = "subscriptions"

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    stripe_subscription_id: Mapped[str] = mapped_column(String, unique=True)
    customer_id: Mapped[int] = mapped_column(BigInteger, ForeignKey("customers.id", ondelete="CASCADE"))
    status: Mapped[str] = mapped_column(String)
    price_id: Mapped[str] = mapped_column(String)
    current_period_start: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    current_period_end: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    cancel_at_period_end: Mapped[bool] = mapped_column(Boolean, default=False)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True))

    customer: Mapped["Customer"] = relationship(back_populates="subscriptions")


class Usage(Base):
    __tablename__ = "usage"
    __table_args__ = (UniqueConstraint("customer_id", "period_start"),)

    id: Mapped[int] = mapped_column(BigInteger, primary_key=True)
    customer_id: Mapped[int] = mapped_column(BigInteger, ForeignKey("customers.id", ondelete="CASCADE"))
    period_start: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    period_end: Mapped[datetime] = mapped_column(DateTime(timezone=True))
    receipts_verified: Mapped[int] = mapped_column(BigInteger, default=0)
    bytes_processed: Mapped[int] = mapped_column(BigInteger, default=0)
    last_request_at: Mapped[datetime | None] = mapped_column(DateTime(timezone=True), nullable=True)

    customer: Mapped["Customer"] = relationship(back_populates="usages")
