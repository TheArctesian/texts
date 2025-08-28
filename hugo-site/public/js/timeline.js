fetch('/data/books.json')
    .then(response => response.json())
    .then(books => {
        if (!books || books.length === 0) return;
        
        const margin = {top: 40, right: 30, bottom: 80, left: 60};
        const width = Math.max(1000, window.innerWidth - 100) - margin.left - margin.right;
        const height = 600 - margin.top - margin.bottom;

        d3.select("#timeline").selectAll("*").remove();
        
        const svg = d3.select("#timeline")
            .append("svg")
            .attr("width", width + margin.left + margin.right)
            .attr("height", height + margin.top + margin.bottom)
            .append("g")
            .attr("transform", "translate(" + margin.left + "," + margin.top + ")");

        // Filter and process books by year
        const validBooks = books.filter(d => d.original_date && d.original_date !== 0);
        
        // Group books by year and count them
        const booksByYear = d3.group(validBooks, d => d.original_date);
        const yearData = Array.from(booksByYear, ([year, books]) => ({
            year: year,
            count: books.length,
            books: books
        }));

        const [minYear, maxYear] = d3.extent(yearData, d => d.year);
        const maxCount = d3.max(yearData, d => d.count);

        // Scales
        const x = d3.scaleLinear()
            .domain([minYear - 50, maxYear + 50])
            .range([0, width]);

        const y = d3.scaleLinear()
            .domain([0, maxCount + 1])
            .range([height, 0]);

        // Grid lines
        svg.append("g")
            .attr("class", "grid")
            .selectAll(".grid-line-x")
            .data(x.ticks(10))
            .enter()
            .append("line")
            .attr("class", "grid-line-x")
            .attr("x1", d => x(d))
            .attr("x2", d => x(d))
            .attr("y1", 0)
            .attr("y2", height)
            .attr("stroke", "var(--nord3)")
            .attr("stroke-width", 0.5)
            .attr("opacity", 0.3);

        svg.append("g")
            .attr("class", "grid")
            .selectAll(".grid-line-y")
            .data(y.ticks(5))
            .enter()
            .append("line")
            .attr("class", "grid-line-y")
            .attr("x1", 0)
            .attr("x2", width)
            .attr("y1", d => y(d))
            .attr("y2", d => y(d))
            .attr("stroke", "var(--nord3)")
            .attr("stroke-width", 0.5)
            .attr("opacity", 0.3);

        // X axis
        svg.append("g")
            .attr("transform", "translate(0," + height + ")")
            .call(d3.axisBottom(x)
                .tickFormat(d3.format("d")))
            .selectAll("text")
            .style("fill", "var(--nord4)")
            .style("font-family", "monospace")
            .style("font-size", "11px");

        // Y axis
        svg.append("g")
            .call(d3.axisLeft(y))
            .selectAll("text")
            .style("fill", "var(--nord4)")
            .style("font-family", "monospace")
            .style("font-size", "11px");

        // Axis labels
        svg.append("text")
            .attr("transform", "translate(" + (width / 2) + " ," + (height + 50) + ")")
            .style("text-anchor", "middle")
            .style("fill", "var(--nord4)")
            .style("font-family", "monospace")
            .style("font-size", "12px")
            .text("Publication Year");

        svg.append("text")
            .attr("transform", "rotate(-90)")
            .attr("y", 0 - margin.left)
            .attr("x", 0 - (height / 2))
            .attr("dy", "1em")
            .style("text-anchor", "middle")
            .style("fill", "var(--nord4)")
            .style("font-family", "monospace")
            .style("font-size", "12px")
            .text("Number of Books");

        // Dots
        svg.selectAll(".dot")
            .data(yearData)
            .enter().append("circle")
            .attr("class", "dot")
            .attr("cx", d => x(d.year))
            .attr("cy", d => y(d.count))
            .attr("r", d => Math.max(4, Math.sqrt(d.count) * 3))
            .attr("fill", "var(--nord8)")
            .attr("stroke", "var(--nord0)")
            .attr("stroke-width", 1)
            .style("cursor", "pointer")
            .on("mouseover", function(event, d) {
                d3.select(this)
                    .attr("fill", "var(--nord7)")
                    .attr("stroke", "var(--nord8)")
                    .attr("stroke-width", 2);
                
                // Create tooltip with book titles
                const bookTitles = d.books.map(book => `${book.title} (${book.author})`).join('<br>');
                
                const tooltip = d3.select("body").append("div")
                    .attr("class", "tooltip")
                    .style("position", "absolute")
                    .style("background", "var(--nord2)")
                    .style("color", "var(--nord4)")
                    .style("padding", "12px")
                    .style("border-radius", "6px")
                    .style("border", "1px solid var(--nord3)")
                    .style("font-family", "monospace")
                    .style("font-size", "11px")
                    .style("pointer-events", "none")
                    .style("z-index", "1000")
                    .style("max-width", "300px")
                    .style("line-height", "1.4")
                    .html(`<strong style="color: var(--nord8);">${d.year}</strong><br>
                           <strong style="color: var(--nord6);">${d.count} book${d.count > 1 ? 's' : ''}</strong><br><br>
                           ${bookTitles}`);
                    
                tooltip.style("left", (event.pageX + 15) + "px")
                       .style("top", (event.pageY - 10) + "px");
            })
            .on("mouseout", function() {
                d3.select(this)
                    .attr("fill", "var(--nord8)")
                    .attr("stroke", "var(--nord0)")
                    .attr("stroke-width", 1);
                d3.selectAll(".tooltip").remove();
            })
            .on("click", function(event, d) {
                // Optional: could add click functionality to filter books by year
                console.log(`Books from ${d.year}:`, d.books);
            });
    });
