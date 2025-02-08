package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
)

type Article struct {
	URL         string
	Title       string
	Author      string
	PublishDate string
	Summary     string
	Content     string
	Sections    []ArticleSection
	WordCount   int
	CharCount   int
	WordFreq    map[string]int
}

type ArticleSection struct {
	Heading string
	Content string
}

type Analysis struct {
	TotalArticles   int
	TotalWords      int
	TotalChars      int
	AverageWords    float64
	CommonWords     map[string]int
	LongestArticle  string
	ShortestArticle string
}

func cleanText(text string) string {
	// Remove extra whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	// Remove special characters but keep basic punctuation
	text = regexp.MustCompile(`[^\w\s.,!?-]`).ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

func extractArticleContent(e *colly.HTMLElement) Article {
	article := Article{
		URL:         e.Request.URL.String(),
		Title:       cleanText(e.ChildText("h1")),
		Author:      cleanText(e.ChildText(".author")), // Adjust selector based on site structure
		PublishDate: cleanText(e.ChildText("time")),
		Summary:     cleanText(e.ChildText(".summary")), // Adjust selector
		Sections:    make([]ArticleSection, 0),
		WordFreq:    make(map[string]int),
	}

	// Extract article sections
	e.ForEach("h2, h3, p", func(i int, el *colly.HTMLElement) {
		switch el.Name {
		case "h2", "h3":
			// Start new section
			section := ArticleSection{
				Heading: cleanText(el.Text),
				Content: "",
			}
			article.Sections = append(article.Sections, section)
		case "p":
			text := cleanText(el.Text)
			if len(text) > 0 {
				if len(article.Sections) == 0 {
					// If no section exists, create one without heading
					article.Sections = append(article.Sections, ArticleSection{
						Heading: "",
						Content: text,
					})
				} else {
					// Add to last section
					lastIdx := len(article.Sections) - 1
					if article.Sections[lastIdx].Content != "" {
						article.Sections[lastIdx].Content += "\n\n"
					}
					article.Sections[lastIdx].Content += text
				}
			}
		}
	})

	// Build full content and calculate metrics
	var fullContent strings.Builder
	for _, section := range article.Sections {
		if section.Heading != "" {
			fullContent.WriteString("\n## " + section.Heading + "\n\n")
		}
		fullContent.WriteString(section.Content + "\n")
	}

	article.Content = fullContent.String()
	words := splitWords(article.Content)
	article.WordCount = len(words)
	article.CharCount = len(article.Content)

	for _, word := range words {
		article.WordFreq[word]++
	}

	return article
}

func saveArticlesToCSV(articles []Article) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("articles_%s.csv", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Title",
		"URL",
		"Author",
		"Publish Date",
		"Summary",
		"Word Count",
		"Character Count",
		"Content",
		"Top 5 Words",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write data for each article
	for _, article := range articles {
		topWords := getTopWordsString(article.WordFreq, 5)

		// Format content with clear section separation
		var formattedContent strings.Builder
		formattedContent.WriteString(fmt.Sprintf("# %s\n\n", article.Title))
		if article.Summary != "" {
			formattedContent.WriteString(fmt.Sprintf("Summary: %s\n\n", article.Summary))
		}

		for _, section := range article.Sections {
			if section.Heading != "" {
				formattedContent.WriteString(fmt.Sprintf("## %s\n\n", section.Heading))
			}
			formattedContent.WriteString(section.Content + "\n\n")
		}

		row := []string{
			article.Title,
			article.URL,
			article.Author,
			article.PublishDate,
			article.Summary,
			strconv.Itoa(article.WordCount),
			strconv.Itoa(article.CharCount),
			formattedContent.String(),
			topWords,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	c := colly.NewCollector(
		colly.AllowURLRevisit(),
	)

	articles := make([]Article, 0)
	visitedURLs := make(map[string]bool)

	c.OnHTML(`a[class="opacity-0 absolute inset-0"]`, func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if !strings.HasPrefix(link, "http") {
			link = e.Request.AbsoluteURL(link)
		}

		if !visitedURLs[link] {
			visitedURLs[link] = true
			fmt.Printf("\nProcessing: %s\n", link)

			contentCollector := c.Clone()

			contentCollector.OnHTML("article", func(e *colly.HTMLElement) {
				article := extractArticleContent(e)
				articles = append(articles, article)

				// Display article info
				fmt.Printf("Title: %s\n", article.Title)
				if article.Author != "" {
					fmt.Printf("Author: %s\n", article.Author)
				}
				if article.PublishDate != "" {
					fmt.Printf("Date: %s\n", article.PublishDate)
				}
				fmt.Printf("Word count: %d\n", article.WordCount)
				fmt.Printf("Sections: %d\n", len(article.Sections))
				fmt.Println("\nTop 5 most frequent words:")
				printTopWords(article.WordFreq, 5)
				fmt.Println("----------------------------------------")
			})

			err := contentCollector.Visit(link)
			if err != nil {
				log.Printf("Error visiting %s: %v\n", link, err)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Error: %v\n", err)
	})

	err := c.Visit("https://fly.io/blog/")
	if err != nil {
		log.Fatal(err)
	}

	// Save articles and analysis
	err = saveArticlesToCSV(articles)
	if err != nil {
		log.Printf("Error saving articles to CSV: %v\n", err)
	}

	analysis := analyzeArticles(articles)
	err = saveAnalysisToCSV(analysis)
	if err != nil {
		log.Printf("Error saving analysis to CSV: %v\n", err)
	}

	// Display analysis summary
	fmt.Println("\n=== Analysis Summary ===")
	fmt.Printf("Total articles: %d\n", analysis.TotalArticles)
	fmt.Printf("Total words: %d\n", analysis.TotalWords)
	fmt.Printf("Total characters: %d\n", analysis.TotalChars)
	fmt.Printf("Average words per article: %.2f\n", analysis.AverageWords)
	fmt.Println("\nTop 10 most common words across all articles:")
	printTopWords(analysis.CommonWords, 10)
}

func saveAnalysisToCSV(analysis Analysis) error {
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("analysis_%s.csv", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write analysis data
	records := [][]string{
		{"Metric", "Value"},
		{"Total Articles", strconv.Itoa(analysis.TotalArticles)},
		{"Total Words", strconv.Itoa(analysis.TotalWords)},
		{"Total Characters", strconv.Itoa(analysis.TotalChars)},
		{"Average Words per Article", fmt.Sprintf("%.2f", analysis.AverageWords)},
		{"Longest Article", analysis.LongestArticle},
		{"Shortest Article", analysis.ShortestArticle},
		{"", ""},
		{"Top 10 Most Common Words", ""},
	}

	// Add top 10 words
	topWords := getTopWordsSlice(analysis.CommonWords, 10)
	for i, pair := range topWords {
		records = append(records, []string{
			fmt.Sprintf("Word %d", i+1),
			fmt.Sprintf("%s (%d times)", pair.word, pair.freq),
		})
	}

	return writer.WriteAll(records)
}

func getTopWordsString(wordFreq map[string]int, n int) string {
	pairs := getTopWordsSlice(wordFreq, n)
	var result []string
	for _, pair := range pairs {
		result = append(result, fmt.Sprintf("%s(%d)", pair.word, pair.freq))
	}
	return strings.Join(result, "; ")
}

type wordFreqPair struct {
	word string
	freq int
}

func getTopWordsSlice(wordFreq map[string]int, n int) []wordFreqPair {
	pairs := make([]wordFreqPair, 0, len(wordFreq))
	for word, freq := range wordFreq {
		pairs = append(pairs, wordFreqPair{word, freq})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].freq > pairs[j].freq
	})

	if len(pairs) > n {
		pairs = pairs[:n]
	}
	return pairs
}

// Function to split text into words
func splitWords(text string) []string {
	// Convert to lowercase and clean the text
	text = strings.ToLower(text)

	// Remove punctuation and special characters
	reg := regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
	text = reg.ReplaceAllString(text, " ")

	// Split into words and filter empty strings
	words := strings.Fields(text)
	validWords := make([]string, 0)

	// Common English stop words to filter out
	stopWords := map[string]bool{
		"the": true, "be": true, "to": true, "of": true, "and": true,
		"a": true, "in": true, "that": true, "have": true, "i": true,
		"it": true, "for": true, "not": true, "on": true, "with": true,
		"he": true, "as": true, "you": true, "do": true, "at": true,
		"this": true, "but": true, "his": true, "by": true, "from": true,
	}

	for _, word := range words {
		// Only include words longer than 2 characters and not in stop words
		if len(word) > 2 && !stopWords[word] {
			validWords = append(validWords, word)
		}
	}

	return validWords
}

// Function to analyze all articles
func analyzeArticles(articles []Article) Analysis {
	analysis := Analysis{
		TotalArticles: len(articles),
		CommonWords:   make(map[string]int),
	}

	var maxWords int
	var minWords int = -1

	for _, article := range articles {
		analysis.TotalWords += article.WordCount
		analysis.TotalChars += article.CharCount

		// Combine word frequencies
		for word, freq := range article.WordFreq {
			analysis.CommonWords[word] += freq
		}

		// Find longest and shortest articles
		if article.WordCount > maxWords {
			maxWords = article.WordCount
			analysis.LongestArticle = article.URL
		}
		if minWords == -1 || article.WordCount < minWords {
			minWords = article.WordCount
			analysis.ShortestArticle = article.URL
		}
	}

	if analysis.TotalArticles > 0 {
		analysis.AverageWords = float64(analysis.TotalWords) / float64(analysis.TotalArticles)
	}

	return analysis
}

// Function to print most frequent words
func printTopWords(wordFreq map[string]int, n int) {
	// Create slice of word-frequency pairs
	type wordFreqPair struct {
		word string
		freq int
	}
	pairs := make([]wordFreqPair, 0, len(wordFreq))

	for word, freq := range wordFreq {
		pairs = append(pairs, wordFreqPair{word, freq})
	}

	// Sort by frequency in descending order
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].freq > pairs[j].freq
	})

	// Print top n words
	for i := 0; i < n && i < len(pairs); i++ {
		fmt.Printf("%d. %s (%d times)\n", i+1, pairs[i].word, pairs[i].freq)
	}
}
