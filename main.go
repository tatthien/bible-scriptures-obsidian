package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gocolly/colly/v2"
)

type Verse struct {
	Number    int
	Scripture string
}

type Chapter struct {
	BookTitle      string
	BookAbbr       string
	Verses         []Verse
	PrevChapter    int
	CurrentChapter int
	NextChapter    int
}

type Book struct {
	Title    string
	Abbr     string
	Chapters []int
}

func writeFile(path string, data interface{}, tmpl *template.Template) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	err = tmpl.Execute(f, data)
	if err != nil {
		log.Fatal(err)
	}
}

func getNextBookAbbr(link string) string {
	re := regexp.MustCompile(`\/doc-kinh-thanh\/([0-9a-z]+)\/.+`)

	abbr := ""

	if re.MatchString(link) {
		result := re.FindAllStringSubmatch(link, -1)
		for i := range result {
			abbr = result[i][1]
		}
	}

	return abbr
}

func makeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}

	return a
}

func main() {
	fmt.Println(">>> Start")

	c := colly.NewCollector()

	tmpl, err := template.ParseFiles("templates/chapter.go.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	bookCount := 1
	nextBookAbbr := ""

	c.OnHTML(".container", func(e *colly.HTMLElement) {
		nextLink := e.ChildAttr(".next a", "href")
		nextBookAbbr = strings.ToUpper(getNextBookAbbr(nextLink))

		nodes := e.ChildTexts(".bible-read .verse")

		if len(nodes) == 0 {
			return
		}

		verses := []Verse{}

		re := regexp.MustCompile(`^[0-9]+`)

		for i, v := range nodes {
			v = re.ReplaceAllString(v, "$1")
			v = strings.TrimSpace(v)
			verse := Verse{Number: i + 1, Scripture: v}
			verses = append(verses, verse)
		}

		currentChapter, _ := strconv.Atoi(e.ChildText(".bible-read h1"))

		chapter := Chapter{
			BookTitle:      e.ChildText(".book-full-width .visible-md-inline"),
			BookAbbr:       e.ChildText(".book-full-width .visible-xs-inline"),
			Verses:         verses,
			PrevChapter:    currentChapter - 1,
			CurrentChapter: currentChapter,
			NextChapter:    currentChapter + 1,
		}

		if nextBookAbbr != chapter.BookAbbr {
			chapter.NextChapter = 0
		}

		dir := fmt.Sprintf("./scriptures/%d-%s", bookCount, chapter.BookAbbr)
		path := fmt.Sprintf("%s/%s-%d.md", dir, chapter.BookAbbr, chapter.CurrentChapter)

		// Create Book directory if not it doesn't exist
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Execute template then write it back to file
		writeFile(path, chapter, tmpl)

		if chapter.NextChapter == 0 {
			// Create site map
			bookTmpl, _ := template.ParseFiles("templates/book.go.tmpl")

			// Get total chapters
			files, _ := ioutil.ReadDir(dir)
			totalChapters := len(files)

			// Prepare the data
			book := Book{
				Title:    chapter.BookTitle,
				Abbr:     chapter.BookAbbr,
				Chapters: makeRange(1, totalChapters),
			}

			writeFile(fmt.Sprintf("%s/%s.md", dir, book.Title), book, bookTmpl)

			bookCount++
		}

		// All done, let's go to the next chapter
		e.Request.Visit(nextLink)
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visisting:", r.URL.String())
	})

	c.Visit("https://kinhthanh.httlvn.org/doc-kinh-thanh/sa/1?v=VI1934")

	fmt.Println(">>>End")
}
