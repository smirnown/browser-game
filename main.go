package main

import (
    "bufio"
    "errors"
    "fmt"
    "html/template"
    "log"
    "net/http"
    "os"
    "slices"
    "strconv"
    "strings"
)

var BOARD_SIZE = 10
var UNPASSABLE = []string {"W", "H", "I", "i"}

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
    Interact Command = "Interact"
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
    http.HandleFunc("/interact/", interactHandler)

    go func() {
        state := GameState {
            Grid: Grid { make([][]string, BOARD_SIZE) },
            Money: 0,
            leverMap: make(map[Point]Point),
            player: Point { 3, 2 },
        }
        for {
            payload, ok := <- commandChan
            if !ok {
                break
            }
            switch payload.command {
                case Initialize:
                    state.Grid = Grid { make([][]string, BOARD_SIZE) }
                    for i := range state.Grid.Tiles {
                        state.Grid.Tiles[i] = make([]string, BOARD_SIZE)
                        for j := 0; j < BOARD_SIZE; j++ {
                            state.Grid.Tiles[i][j] = "_"
                        }
                    }
                    state.Grid.Tiles[3][2] = "P"
                    state.Grid.Tiles[8][8] = "$"
                    for i := 0; i < BOARD_SIZE; i++ {
                        if i == 2 {
                            state.Grid.Tiles[i][6] = "H"
                            state.Grid.Tiles[8][1] = "I"
                            state.leverMap[Point {8, 1}] = Point {i, 6}
                            continue
                        } else if i == 9 {
                            state.Grid.Tiles[i][6] = "H"
                            state.Grid.Tiles[2][1] = "I"
                            state.leverMap[Point {2, 1}] = Point {i, 6}
                            continue
                        }
                        state.Grid.Tiles[i][6] = "W"
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
                case Interact:
                    err := interact(&state)
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
    Grid Grid
    Money int
    leverMap map[Point]Point
    player Point
}

type Grid struct {
    Tiles [][]string
}

func (g *Grid) value(p *Point) string {
    return g.Tiles[p.x][p.y]
}

func (g *Grid) set(p *Point, v string) {
    g.Tiles[p.x][p.y] = v
}

type Point struct {
    x int
    y int
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
        panic(resp.err)
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
        panic(resp.err)
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
        panic(resp.err)
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

func interactHandler(w http.ResponseWriter, r *http.Request) {
    payload := CommandPayload { command: Interact, data: "" }
    commandChan <- payload
    resp := <- stateChan
    if resp.err != nil {
        panic(resp.err)
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
    for _, row := range state.Grid.Tiles {
        data = data + strings.Join(row, "")
    }
    data += "\n" + strconv.Itoa(state.Money)
    data += "\nLever map start"
    for lever, gate := range state.leverMap {
        data += fmt.Sprintf("\n%d,%d:%d,%d", lever.x, lever.y, gate.x, gate.y)
    }
    data += "\nLever map end"
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
    var player Point
    for i := 0; i < BOARD_SIZE; i++ {
        row := data[last:last + BOARD_SIZE]
        playerCol := strings.Index(row, "P")
        if playerCol >= 0 {
            player = Point { i, playerCol }
        }
        tiles[i] = strings.Split(data[last:last + BOARD_SIZE], "")
        last += BOARD_SIZE
    }

    // Second line is money count
    scanner.Scan()
    money, err := strconv.Atoi(scanner.Text())
    if err != nil {
        return nil, err
    }

    // leverMap starts after "Lever map start" and ends with "Lever map end"
    scanner.Scan()
    if scanner.Text() != "Lever map start" {
        return nil, errors.New("Lever map not found in save file")
    }
    leverMap := make(map[Point]Point)
    for {
        scanner.Scan()
        line := scanner.Text()
        if line == "Lever map end" {
            break
        }
        pair := strings.Split(line, ":")
        leverCoords := strings.Split(pair[0], ",")
        leverX, err := strconv.Atoi(leverCoords[0])
        if err != nil {
            return nil, err
        }
        leverY, err := strconv.Atoi(leverCoords[1])
        if err != nil {
            return nil, err
        }
        lever := Point { leverX, leverY }

        gateCoords := strings.Split(pair[1], ",")
        gateX, err := strconv.Atoi(gateCoords[0])
        if err != nil {
            return nil, err
        }
        gateY, err := strconv.Atoi(gateCoords[1])
        if err != nil {
            return nil, err
        }
        gate := Point { gateX, gateY }
        leverMap[lever] = gate
    }

    state := GameState { Grid { tiles }, money, leverMap, player }
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
    switch direction {
        case Up:
            movePlayer(state, &Point { state.player.x - 1, state.player.y })
        case Down:
            movePlayer(state, &Point { state.player.x + 1, state.player.y })
        case Left:
            movePlayer(state, &Point { state.player.x, state.player.y - 1 })
        case Right:
            movePlayer(state, &Point { state.player.x, state.player.y + 1 })
        default:
            return errors.New("Unrecognized Direction")
    }
    return nil
}

func movePlayer(state *GameState, next *Point) error {
    if next.x < 0 || next.y < 0 || next.x >= BOARD_SIZE || next.y >= BOARD_SIZE {
        return nil
    }
    val := state.Grid.value(next)
    if val == "$" {
        state.Money += 1
    } else if slices.Contains(UNPASSABLE, val) {
        return nil
    }
    state.Grid.set(&state.player, "_")
    state.Grid.set(next, "P")
    state.player = *next
    return nil
}

func interact(state *GameState) error {
    toggleLever(state, &Point { state.player.x - 1, state.player.y })
    toggleLever(state, &Point { state.player.x + 1, state.player.y })
    toggleLever(state, &Point { state.player.x, state.player.y - 1 })
    toggleLever(state, &Point { state.player.x, state.player.y + 1 })
    return nil
}

func toggleLever(state *GameState, tile *Point) error {
    if tile.x < 0 || tile.y < 0 || tile.x >= BOARD_SIZE || tile.y >= BOARD_SIZE {
        return nil
    }
    gate, ok := state.leverMap[*tile]
    if !ok {
        return errors.New("Lever doesn't exist in map!")
    }
    lever := state.Grid.value(tile)
    if lever == "I" {
        state.Grid.set(tile, "i")
        state.Grid.set(&gate, "_")
    } else {
        state.Grid.set(tile, "I")
        state.Grid.set(&gate, "H")
    }
    return nil
}

