import re
from typing import List, Dict


def parse_date(date_str: str) -> int:
    """Parse date string to sortable integer (BC negative, AD positive; midpoints for decades)."""
    if not date_str:
        return 0
    date_str = date_str.strip().lower()

    # Handle BC/AD
    is_bc = "bc" in date_str or "b.c." in date_str
    date_str = re.sub(r"[^0-9s]", "", date_str)  # Extract numbers/decades

    if "s" in date_str:  # e.g., "1960s" -> 1965
        year = int(date_str.replace("s", "")[:4])
        year += 5  # Midpoint
    else:
        try:
            year = int(date_str)
        except ValueError:
            return 0

    return -year if is_bc else year


def sort_books(
    books: List[Dict[str, str]], reverse: bool = False
) -> List[Dict[str, str]]:
    """Sort books by original publication date, then by release date for ties. Optional reverse for descending."""

    def sort_key(book):
        orig_date = parse_date(book.get("original_date", ""))
        rel_date = parse_date(book.get("release_date", ""))
        return (orig_date, rel_date)

    return sorted(books, key=sort_key, reverse=reverse)
