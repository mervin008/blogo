package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"math/rand"
	"time"

	"github.com/russross/blackfriday/v2"
)


type Post struct {
	ID        int
	Title     string
	ShortDesc string
	Content   template.HTML
}

// load posts from files
func loadPosts(postsDir string) ([]Post, error) {
	var posts []Post
	files, err := ioutil.ReadDir(postsDir)
	if err != nil {
		return nil, err
	}

	for i, f := range files {
		if !f.IsDir() {
			post, err := loadPostFromFile(filepath.Join(postsDir, f.Name()))
			if err != nil {
				return nil, err
			}
			post.ID = i + 1
			posts = append(posts, post)
		}
	}

	return posts, nil
}

//  load a single post from a file
func loadPostFromFile(filePath string) (Post, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Post{}, err
	}
	defer file.Close()

	var post Post
	var contentBuilder strings.Builder
	scanner := bufio.NewScanner(file)
	contentStarted := false

	for scanner.Scan() {
		line := scanner.Text()
		if contentStarted {
			contentBuilder.WriteString(line + "\n")
		} else {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				switch key {
				case "Title":
					post.Title = value
				case "ShortDesc":
					post.ShortDesc = value
				case "Content":
					contentStarted = true
					contentBuilder.WriteString(value + "\n")
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return Post{}, err
	}

	post.Content = template.HTML(blackfriday.Run([]byte(contentBuilder.String())))
	return post, nil
}

// get post by ID
func getPost(id int, posts []Post) (Post, bool) {
	for _, p := range posts {
		if p.ID == id {
			return p, true
		}
	}
	return Post{}, false
}


func mod(a, b int) int {
	return ((a % b) + b) % b 
}


func getRandomColor() (string, string) {
    rand.Seed(time.Now().UnixNano())
    letters := "0123456789ABCDEF"

    getColor := func() string {
        color := "#"
        for i := 0; i < 6; i++ {
            color += string(letters[rand.Intn(len(letters))])
        }
        return color
    }

    return getColor(), getColor()
}

func main() {
	
	posts, err := loadPosts("posts")
	if err != nil {
		log.Fatal(err)
	}

	funcMap := template.FuncMap{
        "mod": mod,
    }

	
	tmpl := template.Must(template.New("").Funcs(funcMap).ParseGlob("templates/*.html"))

	// Handle homepage route
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		type IndexData struct {
			PostsAndColors []struct { 
				Post  Post
				Color1 string 
				Color2 string  
			}
		}
		
		 
		data := IndexData{
			PostsAndColors: make([]struct {
				Post   Post
				Color1 string 
				Color2 string 
			}, len(posts)),
		}

		for i, post := range posts {
			color1, color2 := getRandomColor() 
			data.PostsAndColors[i].Post = post
			data.PostsAndColors[i].Color1 = color1
			data.PostsAndColors[i].Color2 = color2
		}

		err := tmpl.ExecuteTemplate(w, "base.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	
	http.HandleFunc("/post/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/post/"):]
		if id == "" {
			http.NotFound(w, r)
			return
		}

		// Get the post ID from the URL
		postID, err := strconv.Atoi(id)
		if err != nil {
			http.Error(w, "Invalid post ID", http.StatusBadRequest)
			return
		}

		post, found := getPost(postID, posts)
		if !found {
			http.NotFound(w, r)
			return
		}

		err = tmpl.ExecuteTemplate(w, "post.html", post)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	fmt.Println("Server started at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}