import base64
import json
import os

from langchain_ollama import ChatOllama
from langchain_core.messages import HumanMessage


def extract_book_info(image_path: str, ollama_host: str, model_name: str) -> dict:
    """Extract book info from an image using Ollama."""
    try:
        # Load and encode image to base64
        with open(image_path, "rb") as image_file:
            base64_image = base64.b64encode(image_file.read()).decode("utf-8")
    except Exception as e:
        print(f"Error reading image {image_path}: {e}")
        return {}

    # Updated prompt to extract structured info including new fields (force JSON output)
    prompt = """
    Analyze this book cover image and extract:
    - title: The book's title
    - author: The author(s)
    - description: A brief 1-2 sentence description of the book
    - original_date: Original publication date (e.g., '80 BC' for ancient texts)
    - release_date: Release date of this edition (e.g., '1963' for a modern translation)
    - original_location: The place where the book was first written or published, including the name and approximate longitude/latitude (e.g., 'Königsberg, Prussia (54.7104° N, 20.4522° E)'). Use historical context if needed; put 'Unknown' if not inferable.
    - original_language: The original language the book was written in (e.g., 'German')
    - edition_language: The language of this specific edition/translation in the image (e.g., 'English')

    Respond ONLY in JSON format like: {"title": "...", "author": "...", "description": "...", "original_date": "...", "release_date": "...", "original_location": "...", "original_language": "...", "edition_language": "..."}
    """

    try:
        # Initialize LangChain's ChatOllama
        llm = ChatOllama(base_url=ollama_host, model=model_name, temperature=0)

        # Create message with image and prompt
        message = HumanMessage(
            content=[
                {"type": "text", "text": prompt},
                {
                    "type": "image_url",
                    "image_url": {
                        "url": f"data:image/jpeg;base64,{base64_image}"
                    },  # Assumes JPEG; adjust if needed
                },
            ]
        )

        # Invoke Ollama
        response = llm.invoke([message])
        # Parse JSON from response
        info = json.loads(response.content)
        info["image_path"] = image_path  # Add the image path to the dict
        return info
    except json.JSONDecodeError:
        print(f"Error parsing response for {image_path}: {response.content}")
        return {}
    except Exception as e:
        print(f"Error invoking Ollama for {image_path}: {e}")
        return {}
