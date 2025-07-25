# Use official Python base image
FROM python:3.10-slim

# Set working directory
WORKDIR /app

# Install dependencies
RUN pip install --no-cache-dir langchain-ollama pillow streamlit

# Copy all scripts
COPY config.py .
COPY extractor.py .
COPY sorter.py .
COPY main.py .
COPY app.py .

# Run processing then Streamlit
CMD ["sh", "-c", "python main.py && streamlit run app.py --server.port=8501"]
