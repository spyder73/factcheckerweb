"""
Configuration for Instagram Fetcher Service
"""

import os


class Config:
    """Application configuration"""
    
    # Server settings
    HOST = '0.0.0.0'
    PORT = int(os.environ.get('PORT', 5001))
    DEBUG = True
    
    # Service info
    SERVICE_NAME = "instagram-fetcher"
    
    # Instaloader settings
    DOWNLOAD_VIDEOS = False
    DOWNLOAD_VIDEO_THUMBNAILS = False
    DOWNLOAD_GEOTAGS = False
    DOWNLOAD_COMMENTS = False
    SAVE_METADATA = False
    COMPRESS_JSON = False
    POST_METADATA_TXT_PATTERN = ""
