"""
Instagram Media Fetcher Service
Main application entry point
"""

from flask import Flask
from flask_cors import CORS

from config import Config
from routes.instagram_routes import instagram_bp


def create_app():
    """Create and configure the Flask application"""
    app = Flask(__name__)
    CORS(app)
    
    # Register blueprints
    app.register_blueprint(instagram_bp)
    
    return app


if __name__ == '__main__':
    app = create_app()
    print(f"Starting Instagram Fetcher Service on port {Config.PORT}")
    app.run(host=Config.HOST, port=Config.PORT, debug=Config.DEBUG)
