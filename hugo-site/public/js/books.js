let allBooks = [];
let currentSort = 'title';

function renderBooks(books) {
    const container = document.getElementById('books-container');
    container.innerHTML = '';
    
    // Controls section
    const controlsDiv = document.createElement('div');
    controlsDiv.style.cssText = 'display: flex; gap: 1rem; align-items: center; margin-bottom: 2rem; flex-wrap: wrap;';
    
    // Stats
    const statsDiv = document.createElement('div');
    statsDiv.style.cssText = 'background: var(--nord2); border-radius: 6px; padding: 1rem; font-family: monospace; flex: 1;';
    statsDiv.innerHTML = '<span style="color: var(--nord8);">Total Books:</span> <span style="color: var(--nord6);">' + books.length + '</span>';
    
    // Sort controls
    const sortDiv = document.createElement('div');
    sortDiv.style.cssText = 'display: flex; gap: 0.5rem; align-items: center;';
    sortDiv.innerHTML = '<span style="color: var(--nord4); font-family: monospace;">Sort by:</span>';
    
    const sortOptions = [
        { value: 'title', label: 'Title' },
        { value: 'author', label: 'Author' },
        { value: 'year-asc', label: 'Year ↑' },
        { value: 'year-desc', label: 'Year ↓' }
    ];
    
    sortOptions.forEach(opt => {
        const btn = document.createElement('button');
        btn.textContent = opt.label;
        btn.style.cssText = 'background: ' + (currentSort === opt.value ? 'var(--nord8)' : 'var(--nord2)') + 
            '; color: ' + (currentSort === opt.value ? 'var(--nord0)' : 'var(--nord4)') + 
            '; border: 1px solid var(--nord3); padding: 0.5rem 1rem; border-radius: 4px; cursor: pointer; font-family: monospace; transition: all 0.2s;';
        btn.onclick = () => {
            currentSort = opt.value;
            sortBooks(opt.value);
            renderBooks(allBooks);
        };
        btn.onmouseover = () => {
            if (currentSort !== opt.value) {
                btn.style.background = 'var(--nord3)';
            }
        };
        btn.onmouseout = () => {
            if (currentSort !== opt.value) {
                btn.style.background = 'var(--nord2)';
            }
        };
        sortDiv.appendChild(btn);
    });
    
    controlsDiv.appendChild(statsDiv);
    controlsDiv.appendChild(sortDiv);
    container.appendChild(controlsDiv);
    
    // Books grid
    const booksGrid = document.createElement('div');
    booksGrid.style.cssText = 'display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 1.5rem;';
    
    books.forEach(book => {
        const bookCard = document.createElement('div');
        bookCard.style.cssText = 'background: var(--nord1); border: 1px solid var(--nord3); border-radius: 8px; padding: 1.5rem; transition: transform 0.2s ease, border-color 0.2s ease;';
        
        const year = book.original_date || book.year || (book.llm_analysis && book.llm_analysis.year) || '';
        
        bookCard.innerHTML = '<div style="font-weight: 600; color: var(--nord6); margin-bottom: 0.5rem; font-size: 1.1rem;">' + 
            (book.title || 'Unknown Title') + 
            '</div><div style="color: var(--nord8); margin-bottom: 0.5rem;">by ' + 
            (book.author || 'Unknown Author') + 
            '</div>' +
            (year ? '<div style="color: var(--nord9); font-size: 0.9rem; margin-bottom: 0.5rem;">' + year + '</div>' : '') +
            (book.publisher ? '<div style="color: var(--nord4); font-size: 0.8rem; margin-bottom: 0.5rem;">' + book.publisher + '</div>' : '') +
            (book.description ? '<div style="color: var(--nord4); font-size: 0.9rem; line-height: 1.4; margin-top: 1rem;">' + book.description.substring(0, 200) + (book.description.length > 200 ? '...' : '') + '</div>' : '') +
            (book.llm_analysis && book.llm_analysis.genre ? '<div style="color: var(--nord7); font-size: 0.8rem; margin-top: 1rem;">' + book.llm_analysis.genre + '</div>' : '');
        
        bookCard.addEventListener('mouseenter', () => {
            bookCard.style.transform = 'translateY(-2px)';
            bookCard.style.borderColor = 'var(--nord8)';
        });
        
        bookCard.addEventListener('mouseleave', () => {
            bookCard.style.transform = 'translateY(0)';
            bookCard.style.borderColor = 'var(--nord3)';
        });
        
        booksGrid.appendChild(bookCard);
    });
    
    container.appendChild(booksGrid);
}

function sortBooks(sortType) {
    switch(sortType) {
        case 'title':
            allBooks.sort((a, b) => (a.title || '').localeCompare(b.title || ''));
            break;
        case 'author':
            allBooks.sort((a, b) => (a.author || '').localeCompare(b.author || ''));
            break;
        case 'year-asc':
            allBooks.sort((a, b) => {
                const yearA = a.original_date || a.year || (a.llm_analysis && a.llm_analysis.year) || 9999;
                const yearB = b.original_date || b.year || (b.llm_analysis && b.llm_analysis.year) || 9999;
                return yearA - yearB;
            });
            break;
        case 'year-desc':
            allBooks.sort((a, b) => {
                const yearA = a.original_date || a.year || (a.llm_analysis && a.llm_analysis.year) || -9999;
                const yearB = b.original_date || b.year || (b.llm_analysis && b.llm_analysis.year) || -9999;
                return yearB - yearA;
            });
            break;
    }
}

// Load books on page load
fetch('/data/books.json')
    .then(response => response.json())
    .then(books => {
        if (!books || books.length === 0) {
            document.getElementById('books-container').innerHTML = '<p style="color: var(--nord4);">No books found</p>';
            return;
        }
        allBooks = books;
        sortBooks(currentSort);
        renderBooks(allBooks);
    })
    .catch(err => {
        document.getElementById('books-container').innerHTML = '<p style="color: var(--nord11);">Error loading books: ' + err + '</p>';
    });