# Instagram Fetcher Service

A Flask microservice that uses Instaloader to fetch Instagram posts and return media as base64-encoded data.

## Setup

1. Create a virtual environment:
```bash
python3 -m venv venv
source venv/bin/activate
```

2. Install dependencies:
```bash
pip install -r requirements.txt
```

3. Run the service:
```bash
python app.py
```

The service runs on port 5000 by default.

## API Endpoints

### Health Check
```
GET /health
```

### Fetch Instagram Post
```
POST /fetch
Content-Type: application/json

{
    "url": "https://www.instagram.com/p/SHORTCODE"
}
```

Response:
```json
{
    "success": true,
    "caption": "Post caption text",
    "author": "username",
    "media": [
        {
            "type": "image",
            "data": "data:image/jpeg;base64,..."
        }
    ]
}
```

## Notes

- This service downloads images temporarily and immediately deletes them after converting to base64
- For private posts, you may need to log in (see Instaloader documentation)
- Instagram rate limits apply - don't make too many requests in a short time
