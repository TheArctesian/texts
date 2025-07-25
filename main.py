import json
import os

from config import IMAGE_DIR, OUTPUT_FILE, OLLAMA_HOST, MODEL_NAME
from extractor import extract_book_info


def main():
    # Collect all image paths
    image_paths = [
        os.path.join(IMAGE_DIR, f)
        for f in os.listdir(IMAGE_DIR)
        if f.lower().endswith((".jpg", ".jpeg", ".png"))
    ]

    books = []
    errors = []
    for img_path in image_paths:
        print(f"Processing {img_path}...")
        info = extract_book_info(img_path, OLLAMA_HOST, MODEL_NAME)
        if info:
            books.append(info)
        else:
            errors.append(img_path)

    if errors:
        print(f"Errors processing the following images: {errors}")

    if not books:
        print("No books processed successfully.")
        return

    # Save raw extracted books to JSON
    with open(OUTPUT_FILE, "w") as f:
        json.dump(books, f, indent=4)

    print(f"Extracted data saved to {OUTPUT_FILE}")


if __name__ == "__main__":
    main()
