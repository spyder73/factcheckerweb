FROM python:3.12-slim

WORKDIR /app

# System deps (none strictly required by instaloader on slim, but keep room for future scrapers)
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt ./
RUN pip install --no-cache-dir -r requirements.txt

COPY . .

ENV PORT=5001 \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1

EXPOSE 5001

# In dev we run Flask's dev server. Phase 1+: swap for gunicorn in prod compose.
CMD ["python", "app.py"]
