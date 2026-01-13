"""
Instagram-specific utility functions
"""

import os
import re
import tempfile
import shutil
from typing import List, Dict, Optional, Tuple
import instaloader

from .file_utils import file_to_base64, is_image_file


def extract_shortcode(url: str) -> Optional[str]:
    """
    Extract the shortcode from an Instagram URL
    
    Args:
        url: Instagram post URL
        
    Returns:
        Shortcode string or None if not found
    """
    # Patterns for different Instagram URL formats
    patterns = [
        r'instagram\.com/p/([A-Za-z0-9_-]+)',
        r'instagram\.com/reel/([A-Za-z0-9_-]+)',
        r'instagram\.com/tv/([A-Za-z0-9_-]+)',
    ]
    
    for pattern in patterns:
        match = re.search(pattern, url)
        if match:
            return match.group(1)
    return None


def create_instaloader(config) -> instaloader.Instaloader:
    """
    Create and configure an Instaloader instance
    
    Args:
        config: Configuration object with Instaloader settings
        
    Returns:
        Configured Instaloader instance
    """
    return instaloader.Instaloader(
        download_videos=config.DOWNLOAD_VIDEOS,
        download_video_thumbnails=config.DOWNLOAD_VIDEO_THUMBNAILS,
        download_geotags=config.DOWNLOAD_GEOTAGS,
        download_comments=config.DOWNLOAD_COMMENTS,
        save_metadata=config.SAVE_METADATA,
        compress_json=config.COMPRESS_JSON,
        post_metadata_txt_pattern=config.POST_METADATA_TXT_PATTERN
    )


def collect_media_from_directory(directory: str) -> List[Dict[str, str]]:
    """
    Collect all image files from a directory and convert to base64
    
    Args:
        directory: Path to the directory to scan
        
    Returns:
        List of media dictionaries with type and data
    """
    media_list = []
    
    if not os.path.exists(directory):
        return media_list
    
    for filename in os.listdir(directory):
        filepath = os.path.join(directory, filename)
        if os.path.isfile(filepath) and is_image_file(filename):
            base64_data = file_to_base64(filepath)
            media_list.append({
                "type": "image",
                "data": base64_data
            })
    
    return media_list


def fetch_instagram_media(url: str, loader: instaloader.Instaloader) -> Tuple[bool, Dict]:
    """
    Fetch Instagram post and return media as base64
    
    Args:
        url: Instagram post URL
        loader: Configured Instaloader instance
        
    Returns:
        Tuple of (success: bool, result: dict)
    """
    shortcode = extract_shortcode(url)
    
    if not shortcode:
        return False, {"error": "Could not extract shortcode from URL"}
    
    # Create a temporary directory for downloads
    temp_dir = tempfile.mkdtemp(prefix="insta_")
    
    try:
        # Get the post
        post = instaloader.Post.from_shortcode(loader.context, shortcode)
        
        # Extract caption and author
        caption = post.caption or ""
        author = post.owner_username
        
        # Download media to temp directory
        loader.dirname_pattern = temp_dir
        loader.download_post(post, target="post")
        
        # Collect media from downloaded files
        media_list = []
        
        # Check post subdirectory
        post_dir = os.path.join(temp_dir, "post")
        media_list.extend(collect_media_from_directory(post_dir))
        
        # Also check temp_dir directly (sometimes files end up there)
        media_list.extend(collect_media_from_directory(temp_dir))
        
        return True, {
            "caption": caption,
            "author": author,
            "media": media_list,
            "shortcode": shortcode
        }
        
    except instaloader.exceptions.InstaloaderException as e:
        return False, {"error": f"Instagram error: {str(e)}"}
    except Exception as e:
        return False, {"error": str(e)}
    finally:
        # Clean up temp directory
        shutil.rmtree(temp_dir, ignore_errors=True)
