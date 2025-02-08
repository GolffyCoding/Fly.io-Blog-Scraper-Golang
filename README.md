# Fly.io Blog Scraper

## Overview
This project is a web scraper written in Go that extracts articles from Fly.io's blog, processes the text, and saves the data into CSV files for analysis.

## Features
- Scrapes articles from Fly.io's blog
- Extracts titles, content, and metadata
- Analyzes word frequency and filters out common stop words
- Saves extracted data in CSV format

## Installation

### Prerequisites
- Go (latest version recommended)

### Clone Repository
```sh
git clone https://github.com/yourusername/flyio-blog-scraper.git
cd flyio-blog-scraper
```

### Install Dependencies
```sh
go mod tidy
```

## Usage

### Run the Scraper
```sh
go run main.go
```

### Output
- Scraped data will be saved as CSV files in the `output/` directory.

## Configuration
Modify the following parameters in `main.go` as needed:
- `baseURL`: Change the target website if required
- `stopWords`: Adjust the filtering criteria for word frequency analysis

## TODO
- Improve error handling
- Implement concurrency for faster scraping
- Add support for additional websites

## License
This project is licensed under the MIT License.

