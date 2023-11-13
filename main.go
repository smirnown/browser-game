package main

import (
    "errors"
    "fmt"
    "html/template"
    "log"
    "net/http"
    "os"
    "strings"
)

var BOARD_SIZE = 10

type CommandPayload struct {
    command Command
    data string
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

type TilesChannelResponse struct {
    tiles [][]string
    err error
}

var commandChan = make(chan CommandPayload)
var tilesChan = make(chan TilesChannelResponse)
var saveChan = make(chan bool)

func main() {
    fmt.Println("Registering handlers...")
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/initialize/", initializeHandler)
    http.HandleFunc("/save/", saveHandler)
    http.HandleFunc("/load/", loadHandler)
    http.HandleFunc("/move/", moveHandler)

    go func() {
        tiles := make([][]string, BOARD_SIZE)
        for {
            payload, ok := <- commandChan
            if !ok {
                break
            }
            switch payload.command {
                case Initialize:
                    tiles = make([][]string, BOARD_SIZE)
                    for i := range tiles {
                        tiles[i] = make([]string, BOARD_SIZE)
                        for j := 0; j < BOARD_SIZE; j++ {
                            tiles[i][j] = "_"
                        }
                    }
                    tiles[2][2] = "P"
                    tilesChan <- TilesChannelResponse { tiles, nil }
                case MoveUp:
                    err := move(tiles, Up)
                    resp := TilesChannelResponse{ tiles, err }
                    tilesChan <- resp
                case MoveDown:
                    err := move(tiles, Down)
                    resp := TilesChannelResponse{ tiles, err }
                    tilesChan <- resp
                case MoveLeft:
                    err := move(tiles, Left)
                    resp := TilesChannelResponse{ tiles, err }
                    tilesChan <- resp
                case MoveRight:
                    err := move(tiles, Right)
                    resp := TilesChannelResponse{ tiles, err }
                    tilesChan <- resp
                case Save:
                    err := saveState(tiles, payload.data)
                    saveChan <- err == nil
                case Load:
                    var err error
                    tiles, err = loadState(payload.data)
                    resp := TilesChannelResponse{ tiles, err }
                    tilesChan <- resp
                default:
                    log.Fatal("Unrecognized Command")
            }
        }
    }()
    fmt.Println("Serving...")
    log.Fatal(http.ListenAndServe(":8000", nil))
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
    payload := CommandPayload { command: Initialize, data: "" }
    commandChan <- payload
    resp := <- tilesChan
    if resp.err != nil {
        panic("Error initializing game")
    }
    config := map[string] Board {
        "Board": Board { Tiles: resp.tiles },
    }
    err := tmpl.Execute(w, config)
    if err != nil {
        panic(err)
    }
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
    filename := r.PostFormValue("filename")
    payload := CommandPayload { command: Save, data: filename }
    commandChan <- payload
    success := <- saveChan
    if !success {
        panic("Failed saving game!")
    }
}

func loadHandler(w http.ResponseWriter, r *http.Request) {
    filename := r.PostFormValue("filename")
    payload := CommandPayload { command: Load, data: filename }
    commandChan <- payload
    resp := <- tilesChan
    if resp.err != nil {
        panic("Error loading game")
    }
    config := map[string] Board {
        "Board": Board { Tiles: resp.tiles },
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
    payload := CommandPayload { command: movementDirection, data: "" }
    commandChan <- payload
    resp := <- tilesChan
    if resp.err != nil {
        panic("Error initializing game")
    }
    config := map[string] Board {
        "Board": Board { Tiles: resp.tiles },
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

func loadState(filename string) ([][]string, error) {
    raw, err := os.ReadFile(fmt.Sprintf("./saves/%s.txt", filename))
    if err != nil {
        return nil, err
    }
    data := string(raw)
    tiles := make([][]string, BOARD_SIZE)
    last := 0
    for i := 0; i < BOARD_SIZE; i++ {
        tiles[i] = strings.Split(data[last:last + 5], "")
        last += 5
    }
    return tiles, nil
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

func move(tiles [][]string, direction Direction) error {
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
                        return errors.New("Unrecognized Direction")
                }
                return nil
            }
        }
    }
    return errors.New("Couldn't find player")
}

