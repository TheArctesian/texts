# webdav_server.py
from wsgidav.wsgidav_app import WsgiDAVApp
from wsgidav.fs_dav_provider import FilesystemProvider
import os
from cheroot import wsgi

# Configuration
HOST = "0.0.0.0"  # Listen on all interfaces within the container
PORT = 8082
ROOT_DIRECTORY = "/app/IMGS"  # Container path

# Ensure the directory exists
os.makedirs(ROOT_DIRECTORY, exist_ok=True)

# WebDAV provider and configuration
provider = FilesystemProvider(ROOT_DIRECTORY)

config = {
    "provider_mapping": {"/": provider},
    "http_authenticator": {
        "domain_controller": None,  # No authentication for local use
    },
    "simple_dc": {
        "user_mapping": {"*": True},  # Allow anonymous access
    },
    "verbose": 1,
}

app = WsgiDAVApp(config)

server_args = {
    "bind_addr": (HOST, PORT),
    "wsgi_app": app,
}

server = wsgi.Server(**server_args)

print(f"WebDAV server running at http://{HOST}:{PORT}")
print(f"Serving directory: {ROOT_DIRECTORY}")
print("Press Ctrl+C to quit.")

try:
    server.start()
except KeyboardInterrupt:
    print("WebDAV server stopped.")
