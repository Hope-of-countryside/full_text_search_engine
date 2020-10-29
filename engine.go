package main

import (
	"encoding/xml"
	"fmt"
	"github.com/gin-gonic/gin"
	snowballeng "github.com/kljensen/snowball/english"
	"os"
	"strings"
	"unicode"
)

type document struct {
	Title string `xml:"title"`
	URL   string `xml:"url"`
	Text  string `xml:"abstract"`
	ID    int
}

func loadDocuments(path string) ([]document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	dump := struct {
		Documents []document `xml:"doc"`
	}{}
	if err := dec.Decode(&dump); err != nil {
		return nil, err
	}

	docs := dump.Documents
	for i := range docs {
		docs[i].ID = i
	}
	return docs, nil
}

func tokenize(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		// Split on any character that is not a letter or a number.
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
}

func lowercaseFilter(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = strings.ToLower(token)
	}
	return r
}

var stopwords = map[string]struct{}{
	// Go没有内置Set挺离谱的~
	"a": {}, "and": {}, "be": {}, "have": {}, "i": {},
	"in": {}, "of": {}, "that": {}, "the": {}, "to": {},
}

func stopwordFilter(tokens []string) []string {
	r := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := stopwords[token]; !ok {
			r = append(r, token)
		}
	}
	return r
}

func stemmerFilter(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = snowballeng.Stem(token, false)
	}
	return r
}

func analyze(text string) []string {
	tokens := tokenize(text)
	tokens = lowercaseFilter(tokens)
	tokens = stopwordFilter(tokens)
	tokens = stemmerFilter(tokens)
	return tokens
}

type index map[string][]int

func (idx index) add(docs []document) {
	for _, doc := range docs {
		for _, token := range analyze(doc.Text) {
			ids := idx[token]
			if ids != nil && ids[len(ids)-1] == doc.ID {
				// Don't add same ID twice.
				continue
			}
			idx[token] = append(ids, doc.ID)
		}
	}
}

func (idx index) search(text string) []int {
	var r []int
	for _, token := range analyze(text) {
		if ids, ok := idx[token]; ok {
			if r == nil {
				r = ids
			} else {
				r = intersection(r, ids)
			}
		} else {
			// Token doesn't exist.
			return nil
		}
	}
	return r
}

func intersection(a []int, b []int) []int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	r := make([]int, 0, maxLen)
	var i, j int
	for i < len(a) && j < len(b) {
		if a[i] < b[j] {
			i++
		} else if a[i] > b[j] {
			j++
		} else {
			r = append(r, a[i])
			i++
			j++
		}
	}
	return r
}

//func main() {
//	documents, err := loadDocuments(`enwiki.xml`)
//	revertedIndex := make(index)
//	if err == nil {
//		revertedIndex.add(documents)
//		res := revertedIndex.search("Small wild cat")
//		for _, v := range res {
//			fmt.Print(v)
//		}
//	} else {
//		fmt.Print("载入失败")
//	}
//}

type SearchRequestBody struct{
	searchString string `json:"search_string"`
}
func main() {
	r := gin.Default()
	documents, err := loadDocuments(`enwiki.xml`)
	revertedIndex := make(index)
	if err == nil {
		revertedIndex.add(documents)
		r.POST("/search", func(c *gin.Context) {
			var request SearchRequestBody
			err = c.BindJSON(request)
			res := revertedIndex.search(request.searchString)
			c.JSON(200, gin.H{
				"search_result": res,
			})
		})
		r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	} else {
		fmt.Print("文件载入失败")
	}

}
