package main

import (
    "fmt"
    "html/template"
    "log"
    "net/http"
)

type Board struct {
    Tiles [][]string
}

func main() {
    fmt.Println("starting")
    h1 := func (w http.ResponseWriter, r *http.Request) {
        tmpl := template.Must(template.ParseFiles("index.html"))
        tiles := make([][]string, 5)
        for i := range tiles {
            tiles[i] = []string{"_", "_", "_", "_", "_"}
        }
        tiles[2][2] = "P"
        config := map[string] Board {
            "Board": Board { Tiles: tiles },
        }
        tmpl.Execute(w, config)
    }
    http.HandleFunc("/", h1)

    h2 := func (w http.ResponseWriter, r *http.Request) {
        tmpl := template.Must(template.ParseFiles("index.html"))
        tiles := make([][]string, 5)
        for i := range tiles {
            tiles[i] = []string{"_", "_", "_", "_", "_"}
        }
        tiles[1][2] = "P"
        config := map[string] Board {
            "Board": Board { Tiles: tiles },
        }
        tmpl.ExecuteTemplate(w, "game-board", config)
    }
    http.HandleFunc("/up/", h2)

    h3 := func (w http.ResponseWriter, r *http.Request) {
        tmpl := template.Must(template.ParseFiles("index.html"))
        tiles := make([][]string, 5)
        for i := range tiles {
            tiles[i] = []string{"_", "_", "_", "_", "_"}
        }
        tiles[2][3] = "P"
        config := map[string] Board {
            "Board": Board { Tiles: tiles },
        }
        tmpl.ExecuteTemplate(w, "game-board", config)
    }
    http.HandleFunc("/right/", h3)

    h4 := func (w http.ResponseWriter, r *http.Request) {
        tmpl := template.Must(template.ParseFiles("index.html"))
        tiles := make([][]string, 5)
        for i := range tiles {
            tiles[i] = []string{"_", "_", "_", "_", "_"}
        }
        tiles[3][2] = "P"
        config := map[string] Board {
            "Board": Board { Tiles: tiles },
        }
        tmpl.ExecuteTemplate(w, "game-board", config)
    }
    http.HandleFunc("/down/", h4)


    h5 := func (w http.ResponseWriter, r *http.Request) {
        tmpl := template.Must(template.ParseFiles("index.html"))
        tiles := make([][]string, 5)
        for i := range tiles {
            tiles[i] = []string{"_", "_", "_", "_", "_"}
        }
        tiles[2][1] = "P"
        config := map[string] Board {
            "Board": Board { Tiles: tiles },
        }
        tmpl.ExecuteTemplate(w, "game-board", config)
    }
    http.HandleFunc("/left/", h5)

    log.Fatal(http.ListenAndServe(":8000", nil))
}

