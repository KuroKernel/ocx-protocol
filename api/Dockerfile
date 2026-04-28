# =====================================================================
# OCX API — multi-stage Dockerfile
# Stage 1: build libocx-verify (Rust)
# Stage 2: install Python deps and run FastAPI
# =====================================================================

FROM rust:1.86-slim AS rust-builder

WORKDIR /build
# Copy only the verifier crate (depends on nothing else in the repo)
COPY libocx-verify/ ./libocx-verify/

WORKDIR /build/libocx-verify
RUN cargo build --release

# =====================================================================
FROM python:3.11-slim AS app

WORKDIR /opt/ocx

# System deps for psycopg + general health
RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates curl libpq5 \
 && rm -rf /var/lib/apt/lists/*

# Python deps (cached as long as pyproject.toml is unchanged)
COPY api/pyproject.toml ./
RUN pip install --no-cache-dir --upgrade pip \
 && pip install --no-cache-dir \
      "fastapi>=0.115.0" "uvicorn[standard]>=0.32.0" \
      "pydantic>=2.9.0" "pydantic-settings>=2.5.0" \
      "sqlalchemy>=2.0.36" "psycopg[binary]>=3.2.3" "alembic>=1.14.0" \
      "httpx>=0.27.0" "python-multipart>=0.0.12" "email-validator>=2.2.0"

# Application code
COPY api/app ./app
COPY api/schema.sql ./schema.sql
COPY api/init_db.py ./init_db.py

# Compiled verifier from rust-builder stage
COPY --from=rust-builder /build/libocx-verify/target/release/liblibocx_verify.so /opt/ocx/

# Railway injects PORT; default to 8000 for local
ENV PORT=8000
EXPOSE 8000

CMD ["sh", "-c", "uvicorn app.main:app --host 0.0.0.0 --port ${PORT:-8000} --proxy-headers --forwarded-allow-ips='*'"]
