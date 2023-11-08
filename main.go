package main

import (
    "fmt"
    "html/template"
    "log"
    "net/http"
    "os"
    "strings"
)

type Board struct {
    Tiles [][]string
}

func saveState(tiles [][]string) {
    state := ""
    for _, row := range tiles {
        state = state + strings.Join(row, "")
    }
    data := []byte(state)
    err := os.WriteFile("./state.txt", data, 0644)
    if err != nil {
        panic(err)
    }
}

func loadState() [][]string {
    raw, err := os.ReadFile("./state.txt")
    if err != nil {
        panic(err)
    }
    data := string(raw)
    tiles := make([][]string, 5)
    tiles[0] = strings.Split(data[0:5], "")
    tiles[1] = strings.Split(data[5:10], "")
    tiles[2] = strings.Split(data[10:15], "")
    tiles[3] = strings.Split(data[15:20], "")
    tiles[4] = strings.Split(data[20:25], "")
    return tiles
}

type Direction int
const (
    Up Direction = iota
    Down
    Left
    Right
)

var directionMap = map[string]Direction {
    "up": Up,
    "down": Down,
    "left": Left,
    "right": Right,
}

func parseDirection(value string) Direction {
    direction, ok := directionMap[value]
    if !ok {
        panic("Couldn't determine direction!")
    }
    return direction
}

func move(tiles [][]string, direction Direction) {
    for i, row := range tiles {
        for j, value := range row {
            if value == "P" {
                tiles[i][j] = "_"
                switch direction {
                    case Up:
                        tiles[i - 1][j] = "P"
                    case Down:
                        tiles[i + 1][j] = "P"
                    case Left:
                        tiles[i][j - 1] = "P"
                    case Right:
                        tiles[i][j + 1] = "P"
                    default:
                        panic("Unrecognized Direction")
                }
                return
            }
        }
    }
    panic("Couldn't find player")
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

        saveState(tiles)

        config := map[string] Board {
            "Board": Board { Tiles: tiles },
        }
        tmpl.Execute(w, config)
    }
    http.HandleFunc("/", h1)

    moveHandler := func (w http.ResponseWriter, r *http.Request) {
        d := r.PostFormValue("direction")
        direction := parseDirection(d)

        tiles := loadState()
        move(tiles, direction)
        saveState(tiles)

        config := map[string] Board {
            "Board": Board { Tiles: tiles },
        }
        tmpl := template.Must(template.ParseFiles("index.html"))
        tmpl.ExecuteTemplate(w, "game-board", config)
    }
    http.HandleFunc("/move/", moveHandler)

    log.Fatal(http.ListenAndServe(":8000", nil))
}

