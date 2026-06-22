"""
Instagram fetch routes
"""

from flask import Blueprint, request, jsonify
import instaloader

from utils.instagram_utils import fetch_instagram_media
from config import Config


instagram_bp = Blueprint('instagram', __name__)

# Initialize Instaloader
loader = instaloader.Instaloader(
    download_videos=Config.DOWNLOAD_VIDEOS,
    download_video_thumbnails=Config.DOWNLOAD_VIDEO_THUMBNAILS,
    download_geotags=Config.DOWNLOAD_GEOTAGS,
    download_comments=Config.DOWNLOAD_COMMENTS,
    save_metadata=Config.SAVE_METADATA,
    compress_json=Config.COMPRESS_JSON,
    post_metadata_txt_pattern=Config.POST_METADATA_TXT_PATTERN
)


@instagram_bp.route('/health', methods=['GET'])
def health():
    """Health check endpoint"""
    return jsonify({"status": "ok", "service": Config.SERVICE_NAME})


@instagram_bp.route('/fetch', methods=['POST'])
def fetch_post():
    """
    Fetch an Instagram post and return media as base64
    
    Request body:
    {
        "url": "https://www.instagram.com/p/SHORTCODE"
    }
    
    Response:
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
    """
    data = request.get_json()
    
    if not data or 'url' not in data:
        return jsonify({
            "success": False,
            "error": "Missing 'url' in request body"
        }), 400
    
    url = data['url']
    success, result = fetch_instagram_media(url, loader)
    
    if success:
        return jsonify({
            "success": True,
            **result
        })
    else:
        status_code = 400 if "Instagram error" in result.get("error", "") else 500
        return jsonify({
            "success": False,
            **result
        }), status_code
