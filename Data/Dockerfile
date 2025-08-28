# Dockerfile
FROM python:3.9-slim

WORKDIR /app

# Install dependencies
RUN pip install wsgidav cheroot

# Copy the server script
COPY webdav_server.py /app/

# Create directory for images
RUN mkdir -p /app/IMGS

# Expose WebDAV port
EXPOSE 8082

# Run the server
CMD ["python", "webdav_server.py"]
