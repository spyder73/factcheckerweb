"""
File and encoding utilities
"""

import os
import base64


def get_image_mime_type(filepath: str) -> str:
    """
    Determine MIME type from file extension
    
    Args:
        filepath: Path to the image file
        
    Returns:
        MIME type string (defaults to 'image/jpeg' if unknown)
    """
    ext = os.path.splitext(filepath)[1].lower()
    mime_types = {
        '.jpg': 'image/jpeg',
        '.jpeg': 'image/jpeg',
        '.png': 'image/png',
        '.gif': 'image/gif',
        '.webp': 'image/webp',
    }
    return mime_types.get(ext, 'image/jpeg')


def file_to_base64(filepath: str) -> str:
    """
    Convert a file to base64 data URL
    
    Args:
        filepath: Path to the file to convert
        
    Returns:
        Base64 data URL string
    """
    mime_type = get_image_mime_type(filepath)
    with open(filepath, 'rb') as f:
        data = base64.b64encode(f.read()).decode('utf-8')
    return f"data:{mime_type};base64,{data}"


def is_image_file(filename: str) -> bool:
    """
    Check if a filename has an image extension
    
    Args:
        filename: Name of the file to check
        
    Returns:
        True if the file has an image extension
    """
    ext = os.path.splitext(filename)[1].lower()
    return ext in ['.jpg', '.jpeg', '.png', '.gif', '.webp']
