package main

import (
    "fmt"
    "html/template"
    "log"
    "net/http"
    "os"
    "strings"
)

var commandChan = make(chan CommandPayload)
var tilesChan = make(chan [][]string)
var saveChan = make(chan bool)

func main() {
    fmt.Println("Registering handlers...")
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/initialize/", initializeHandler)
    http.HandleFunc("/save/", saveHandler)
    http.HandleFunc("/load/", loadHandler)
    http.HandleFunc("/move/", moveHandler)

    go func() {
        tiles := make([][]string, 5)
        for {
            fmt.Println("Polling...")
            payload, ok := <- commandChan
            if !ok {
                fmt.Println("Not ok - breaking!")
                break
            }
            fmt.Println(payload)
            switch payload.Command {
                case Initialize:
                    tiles = make([][]string, 5)
                    for i := range tiles {
                        tiles[i] = []string{"_", "_", "_", "_", "_"}
                    }
                    tiles[2][2] = "P"
                    tilesChan <- tiles
                case MoveUp:
                    move(tiles, Up)
                    tilesChan <- tiles
                case MoveDown:
                    move(tiles, Down)
                    tilesChan <- tiles
                case MoveLeft:
                    move(tiles, Left)
                    tilesChan <- tiles
                case MoveRight:
                    move(tiles, Right)
                    tilesChan <- tiles
                case Save:
                    err := saveState(tiles, payload.Data)
                    saveChan <- err == nil
                case Load:
                    tiles = loadState(payload.Data)
                    tilesChan <- tiles
                default:
                    panic("Unrecognized Command")
            }
        }
    }()
    fmt.Println("Serving...")
    log.Fatal(http.ListenAndServe(":8000", nil))
}

type Command string
const (
    Initialize Command = "Initialize"
    MoveUp Command = "MoveUp"
    MoveDown Command = "MoveDown"
    MoveLeft Command = "MoveLeft"
    MoveRight Command = "MoveRight"
    Save Command = "Save"
    Load Command = "Load"
)
type CommandPayload struct {
    Command Command
    Data string
}

/***************************
    HTTP HANDLERS
***************************/
type Board struct {
    Tiles [][]string
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("index.html"))
    config := make(map[string]Board)
    err := tmpl.Execute(w, config)
    if err != nil {
        panic(err)
    }
}

func initializeHandler(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("game-board.html"))
    payload := CommandPayload { Command: Initialize, Data: "" }
    commandChan <- payload
    tiles, ok := <- tilesChan
    if !ok {
        panic("Error initializing game")
    }
    config := map[string] Board {
        "Board": Board { Tiles: tiles },
    }
    err := tmpl.Execute(w, config)
    if err != nil {
        panic(err)
    }
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
    filename := r.PostFormValue("filename")
    payload := CommandPayload { Command: Save, Data: filename }
    commandChan <- payload
    success := <- saveChan
    if !success {
        panic("Failed saving game!")
    }
}

func loadHandler(w http.ResponseWriter, r *http.Request) {
    filename := r.PostFormValue("filename")
    payload := CommandPayload { Command: Load, Data: filename }
    commandChan <- payload
    tiles, ok := <- tilesChan
    if !ok {
        panic("Error loading game")
    }
    config := map[string] Board {
        "Board": Board { Tiles: tiles },
    }
    tmpl := template.Must(template.ParseFiles("game-board.html"))
    err := tmpl.Execute(w, config)
    if err != nil {
        panic(err)
    }
}

func moveHandler(w http.ResponseWriter, r *http.Request) {
    d := r.PostFormValue("direction")
    movementDirection := parseMovementDirection(d)
    payload := CommandPayload { Command: movementDirection, Data: "" }
    commandChan <- payload
    tiles, ok := <- tilesChan
    if !ok {
        panic("Error initializing game")
    }
    config := map[string] Board {
        "Board": Board { Tiles: tiles },
    }
    tmpl := template.Must(template.ParseFiles("game-board.html"))
    err := tmpl.ExecuteTemplate(w, "game-board", config)
    if err != nil {
        panic(err)
    }
}

/***************************
    HELPERS
***************************/
func saveState(tiles [][]string, filename string) error {
    state := ""
    for _, row := range tiles {
        state = state + strings.Join(row, "")
    }
    data := []byte(state)
    err := os.WriteFile(fmt.Sprintf("./saves/%s.txt", filename), data, 0644)
    return err
}

func loadState(filename string) [][]string {
    raw, err := os.ReadFile(fmt.Sprintf("./saves/%s.txt", filename))
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

var MovementDirectionMap = map[string]Command {
    "MoveUp": MoveUp,
    "MoveDown": MoveDown,
    "MoveLeft": MoveLeft,
    "MoveRight": MoveRight,
}

func parseMovementDirection(value string) Command {
    direction, ok := MovementDirectionMap[value]
    if !ok {
        panic("Couldn't determine movement direction!")
    }
    return direction
}

func move(tiles [][]string, direction Direction) {
    for i, row := range tiles {
        for j, value := range row {
            if value == "P" {
                switch direction {
                    case Up:
                        if i > 0 {
                            tiles[i][j] = "_"
                            tiles[i - 1][j] = "P"
                        }
                    case Down:
                        if i < len(tiles) - 1 {
                            tiles[i][j] = "_"
                            tiles[i + 1][j] = "P"
                        }
                    case Left:
                        if j > 0 {
                            tiles[i][j] = "_"
                            tiles[i][j - 1] = "P"
                        }
                    case Right:
                        if j < len(row) - 1 {
                            tiles[i][j] = "_"
                            tiles[i][j + 1] = "P"
                        }
                    default:
                        panic("Unrecognized Direction")
                }
                return
            }
        }
    }
    panic("Couldn't find player")
}

