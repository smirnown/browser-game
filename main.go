package main

import (
    "bufio"
    "errors"
    "fmt"
    "html/template"
    "log"
    "net/http"
    "os"
    "strconv"
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

type StateChannelResponse struct {
    state GameState
    err error
}

var commandChan = make(chan CommandPayload)
var stateChan = make(chan StateChannelResponse)
var saveChan = make(chan bool)

func main() {
    fmt.Println("Registering handlers...")
    http.HandleFunc("/", rootHandler)
    http.HandleFunc("/initialize/", initializeHandler)
    http.HandleFunc("/save/", saveHandler)
    http.HandleFunc("/load/", loadHandler)
    http.HandleFunc("/move/", moveHandler)

    go func() {
        state := GameState {
            Tiles: make([][]string, BOARD_SIZE),
            Money: 0,
        }
        for {
            payload, ok := <- commandChan
            if !ok {
                break
            }
            switch payload.command {
                case Initialize:
                    state.Tiles = make([][]string, BOARD_SIZE)
                    for i := range state.Tiles {
                        state.Tiles[i] = make([]string, BOARD_SIZE)
                        for j := 0; j < BOARD_SIZE; j++ {
                            state.Tiles[i][j] = "_"
                        }
                    }
                    state.Tiles[2][2] = "P"
                    state.Tiles[2][4] = "$"
                    for i := 0; i < BOARD_SIZE; i++ {
                        state.Tiles[i][5] = "W"
                    }
                    state.Money = 0
                    stateChan <- StateChannelResponse { state, nil }
                case MoveUp:
                    err := move(&state, Up)
                    resp := StateChannelResponse{ state, err }
                    stateChan <- resp
                case MoveDown:
                    err := move(&state, Down)
                    resp := StateChannelResponse{ state, err }
                    stateChan <- resp
                case MoveLeft:
                    err := move(&state, Left)
                    resp := StateChannelResponse{ state, err }
                    stateChan <- resp
                case MoveRight:
                    err := move(&state, Right)
                    resp := StateChannelResponse{ state, err }
                    stateChan <- resp
                case Save:
                    err := saveState(state, payload.data)
                    saveChan <- err == nil
                case Load:
                    statePointer, err := loadState(payload.data)
                    state = *statePointer
                    resp := StateChannelResponse{ state, err }
                    stateChan <- resp
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
type GameState struct {
    Tiles [][]string
    Money int
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("index.html"))
    config := make(map[string]GameState)
    err := tmpl.Execute(w, config)
    if err != nil {
        panic(err)
    }
}

func initializeHandler(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("game-board.html"))
    payload := CommandPayload { command: Initialize, data: "" }
    commandChan <- payload
    resp := <- stateChan
    if resp.err != nil {
        panic("Error initializing game")
    }
    config := map[string] GameState {
        "GameState": resp.state,
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
    resp := <- stateChan
    if resp.err != nil {
        panic("Error loading game")
    }
    config := map[string] GameState {
        "GameState": resp.state,
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
    resp := <- stateChan
    if resp.err != nil {
        panic("Error initializing game")
    }
    config := map[string]GameState {
        "GameState": resp.state,
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
func saveState(state GameState, filename string) error {
    data := ""
    for _, row := range state.Tiles {
        data = data + strings.Join(row, "")
    }
    data += "\n" + strconv.Itoa(state.Money)
    bytes := []byte(data)
    err := os.WriteFile(fmt.Sprintf("./saves/%s.txt", filename), bytes, 0644)
    return err
}

func loadState(filename string) (*GameState, error) {
    file, err := os.Open(fmt.Sprintf("./saves/%s.txt", filename))
    if err != nil {
        return nil, err
    }
    defer file.Close()

    scanner := bufio.NewScanner(file)

    // First line is tiles
    scanner.Scan()
    data := scanner.Text()
    tiles := make([][]string, BOARD_SIZE)
    last := 0
    for i := 0; i < BOARD_SIZE; i++ {
        tiles[i] = strings.Split(data[last:last + BOARD_SIZE], "")
        last += BOARD_SIZE
    }

    // Second line is money count
    scanner.Scan()
    money, err := strconv.Atoi(scanner.Text())
    if err != nil {
        return nil, err
    }

    state := GameState { tiles, money }
    return &state, nil
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

func move(state *GameState, direction Direction) error {
    for i, row := range state.Tiles {
        for j, value := range row {
            if value == "P" {
                switch direction {
                    case Up:
                        if i > 0 {
                            next := state.Tiles[i - 1][j]
                            if next == "$" {
                                state.Money += 1
                            } else if next == "W" {
                                return nil
                            }
                            state.Tiles[i][j] = "_"
                            state.Tiles[i - 1][j] = "P"
                        }
                    case Down:
                        if i < len(state.Tiles) - 1 {
                            next := state.Tiles[i + 1][j]
                            if next == "$" {
                                state.Money += 1
                            } else if next == "W" {
                                return nil
                            }
                            state.Tiles[i][j] = "_"
                            state.Tiles[i + 1][j] = "P"
                        }
                    case Left:
                        if j > 0 {
                            next := state.Tiles[i][j - 1]
                            if next == "$" {
                                state.Money += 1
                            } else if next == "W" {
                                return nil
                            }
                            state.Tiles[i][j] = "_"
                            state.Tiles[i][j - 1] = "P"
                        }
                    case Right:
                        if j < len(row) - 1 {
                            next := state.Tiles[i][j + 1]
                            if next == "$" {
                                state.Money += 1
                            } else if next == "W" {
                                return nil
                            }
                            state.Tiles[i][j] = "_"
                            state.Tiles[i][j + 1] = "P"
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

