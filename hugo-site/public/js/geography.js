const map = L.map('map', {
    zoomControl: true,
    attributionControl: false
}).setView([20, 0], 2);

// Dark theme tiles
L.tileLayer('https://tiles.stadiamaps.com/tiles/alidade_smooth_dark/{z}/{x}/{y}{r}.png', {
    attribution: 'Map data &copy; OpenStreetMap contributors'
}).addTo(map);

// Custom marker icon with Nord colors
const customIcon = L.divIcon({
    html: '<div style="background: var(--nord8); width: 12px; height: 12px; border-radius: 50%; border: 2px solid var(--nord0); box-shadow: 0 2px 4px rgba(0,0,0,0.5);"></div>',
    className: 'custom-marker',
    iconSize: [12, 12],
    iconAnchor: [6, 6]
});

fetch('/data/books.json')
    .then(response => response.json())
    .then(books => {
        // Extract unique locations from books
        const locationMap = {};
        books.forEach(book => {
            if (book.original_location_name && book.original_location_latitude && book.original_location_longitude) {
                const key = book.original_location_name;
                if (!locationMap[key]) {
                    locationMap[key] = {
                        name: book.original_location_name,
                        lat: book.original_location_latitude,
                        lon: book.original_location_longitude,
                        books: []
                    };
                }
                locationMap[key].books.push(book.title);
            }
        });
        
        const locations = Object.values(locationMap);
        if (locations.length === 0) {
            document.getElementById('map').innerHTML = '<div style="color: var(--nord4); text-align: center; padding: 2rem;">No geographic data available</div>';
            return;
        }
        
        locations.forEach((location, index) => {
            // Spread out multiple books at same location in a small circle
            const bookCount = location.books.length;
            
            if (bookCount === 1) {
                // Single book - place at exact coordinates
                const marker = L.marker([location.lat, location.lon], {icon: customIcon})
                    .addTo(map);
                
                const popupContent = '<div style="color: var(--nord4); font-family: monospace;"><strong style="color: var(--nord8);">' + 
                    location.name + '</strong><br>' + 
                    '<span style="color: var(--nord4);">' + location.books[0] + '</span>' +
                    '</div>';
                
                marker.bindPopup(popupContent, {
                    className: 'nord-popup'
                });
            } else {
                // Multiple books - spread them in a small circle around the location
                const radius = 0.3; // degrees - small spread
                const angleStep = (2 * Math.PI) / bookCount;
                
                location.books.forEach((book, bookIndex) => {
                    const angle = bookIndex * angleStep;
                    const offsetLat = radius * Math.cos(angle);
                    const offsetLon = radius * Math.sin(angle);
                    
                    const adjustedLat = location.lat + offsetLat;
                    const adjustedLon = location.lon + offsetLon;
                    
                    const marker = L.marker([adjustedLat, adjustedLon], {icon: customIcon})
                        .addTo(map);
                    
                    const popupContent = '<div style="color: var(--nord4); font-family: monospace;"><strong style="color: var(--nord8);">' + 
                        location.name + '</strong><br>' + 
                        '<span style="color: var(--nord4);">' + book + '</span><br>' +
                        '<span style="color: var(--nord3); font-size: 0.8em;">(' + bookCount + ' books from this location)</span>' +
                        '</div>';
                    
                    marker.bindPopup(popupContent, {
                        className: 'nord-popup'
                    });
                });
            }
        });
        
        // Add custom CSS for popups
        const style = document.createElement('style');
        style.textContent = '.nord-popup .leaflet-popup-content-wrapper { background: var(--nord2); color: var(--nord4); border-radius: 6px; } .nord-popup .leaflet-popup-tip { background: var(--nord2); }';
        document.head.appendChild(style);
    });
