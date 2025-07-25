import json
import os
import subprocess
import streamlit as st
from config import OUTPUT_FILE
from sorter import sort_books

st.title("Sorted Book Library")

# Button to re-process images
if st.button("Re-process Images"):
    subprocess.run(["python", "main.py"])
    st.success("Processing complete! Refresh to see updates.")

# Search bar
search_query = st.text_input("Search by Title or Author", "")

# Sort toggle
sort_reverse = st.checkbox("Sort Newest to Oldest (Descending)", value=False)

# Load books
if os.path.exists(OUTPUT_FILE):
    with open(OUTPUT_FILE, "r") as f:
        books = json.load(f)

    if books:
        # Filter by search query (case-insensitive, partial match on title or author)
        if search_query:
            search_lower = search_query.lower()
            books = [
                b
                for b in books
                if search_lower in b.get("title", "").lower()
                or search_lower in b.get("author", "").lower()
            ]

        if not books:
            st.warning("No books match the search query.")
        else:
            # Sort books
            sorted_books = sort_books(books, reverse=sort_reverse)

            # Display each book with image and details
            st.subheader("Books (Sorted by Original Publication Date)")
            for book in sorted_books:
                st.markdown("---")  # Horizontal separator

                # Display image with error handling
                image_path = book.get("image_path")
                try:
                    if image_path and os.path.exists(image_path):
                        st.image(
                            image_path,
                            use_column_width=True,
                            caption=book.get("title", "N/A"),
                        )
                    else:
                        st.text("Image not available")
                except Exception as e:
                    st.text(f"Error loading image: {e}")

                # Display details
                st.write(f"**Title:** {book.get('title', 'N/A')}")
                st.write(f"**Author:** {book.get('author', 'N/A')}")
                st.write(f"**Description:** {book.get('description', 'N/A')}")
                st.write(f"**Original Date:** {book.get('original_date', 'N/A')}")
                st.write(f"**Release Date:** {book.get('release_date', 'N/A')}")
                st.write(
                    f"**Original Location:** {book.get('original_location', 'N/A')}"
                )
                st.write(
                    f"**Original Language:** {book.get('original_language', 'N/A')}"
                )
                st.write(f"**Edition Language:** {book.get('edition_language', 'N/A')}")
    else:
        st.warning("No books found in the output file.")
else:
    st.error(f"Output file not found: {OUTPUT_FILE}. Run processing first.")
