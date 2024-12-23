package asciitimer

import (
    "fmt"
    "time"
    "os"
    "os/signal"
    "syscall"
    "strings"
    "unsafe"
)

type Termios struct {
    Iflag  uint32
    Oflag  uint32
    Cflag  uint32
    Lflag  uint32
    Line   uint8
    Cc     [19]uint8
    Ispeed uint32
    Ospeed uint32
}

var digits = [][]string{
    { // 0
        "█████",
        "█   █",
        "█   █",
        "█   █",
        "█████",
    },
    { // 1
        "  █  ",
        " ██  ",
        "  █  ",
        "  █  ",
        "█████",
    },
    { // 2
        "█████",
        "    █",
        "█████",
        "█    ",
        "█████",
    },
    { // 3
        "█████",
        "    █",
        "█████",
        "    █",
        "█████",
    },
    { // 4
        "█   █",
        "█   █",
        "█████",
        "    █",
        "    █",
    },
    { // 5
        "█████",
        "█    ",
        "█████",
        "    █",
        "█████",
    },
    { // 6
        "█████",
        "█    ",
        "█████",
        "█   █",
        "█████",
    },
    { // 7
        "█████",
        "    █",
        "   █ ",
        "  █  ",
        " █   ",
    },
    { // 8
        "█████",
        "█   █",
        "█████",
        "█   █",
        "█████",
    },
    { // 9
        "█████",
        "█   █",
        "█████",
        "    █",
        "█████",
    },
}

var colon = []string{
    " ",
    "█",
    " ",
    "█",
    " ",
}

var pausedText = []string{
    "█████  █████  █   █  █████  █████  ████ ",
    "█   █  █   █  █   █  █      █      █   █",
    "█████  █████  █   █  █████  █████  █   █",
    "█      █   █  █   █      █  █      █   █",
    "█      █   █  █████  █████  █████  ████ ",
}

const (
    TCGETS = 0x5401
    TCSETS = 0x5402
    ECHO   = 0x00000008
    ICANON = 0x00000002
    VMIN   = 0x6
    VTIME  = 0x5
)

func clearScreen() {
    fmt.Print("\033[H\033[2J")
}

func getASCIITime(h, m, s int) string {
    timeStr := fmt.Sprintf("%02d:%02d:%02d", h, m, s)
    
    var result []string
    for row := 0; row < 5; row++ {
        line := ""
        for _, char := range timeStr {
            if char == ':' {
                line += colon[row] + " "
            } else {
                digit := int(char - '0')
                line += digits[digit][row] + " "
            }
        }
        result = append(result, line)
    }
    
    return strings.Join(result, "\n")
}

func makeRaw(fd int) (*Termios, error) {
    termios := &Termios{}
    
    _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
        uintptr(fd),
        uintptr(TCGETS),
        uintptr(unsafe.Pointer(termios)))
        
    if errno != 0 {
        return nil, errno
    }
    
    oldTermios := *termios
    
    // Отключаем канонический режим и эхо
    termios.Lflag &^= uint32(ICANON | ECHO)
    termios.Cc[VMIN] = 1
    termios.Cc[VTIME] = 0
    
    _, _, errno = syscall.Syscall(syscall.SYS_IOCTL,
        uintptr(fd),
        uintptr(TCSETS),
        uintptr(unsafe.Pointer(termios)))
        
    if errno != 0 {
        return nil, errno
    }
    
    return &oldTermios, nil
}

func restoreTerminal(fd int, oldState *Termios) error {
    _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
        uintptr(fd),
        uintptr(TCSETS),
        uintptr(unsafe.Pointer(oldState)))
    
    if errno != 0 {
        return errno
    }
    
    return nil
}

func RunTimer() {
    fd := int(os.Stdin.Fd())
    oldState, err := makeRaw(fd)
    if err != nil {
        fmt.Printf("Ошибка при настройке терминала: %v\n", err)
        return
    }
    defer restoreTerminal(fd, oldState)

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    isPaused := false
    startTime := time.Now()
    pausedDuration := time.Duration(0)
    pauseStart := time.Time{}

    go func() {
        buf := make([]byte, 1)
        for {
            os.Stdin.Read(buf)
            if buf[0] == 'p' || buf[0] == 'P' {
                isPaused = !isPaused
                if isPaused {
                    pauseStart = time.Now()
                } else if !pauseStart.IsZero() {
                    pausedDuration += time.Since(pauseStart)
                }
            }
            if buf[0] == 3 || buf[0] == 'q' {
                restoreTerminal(fd, oldState)
                os.Exit(0)
            }
        }
    }()

    clearScreen()
    fmt.Print("\033[?25l")
    defer fmt.Print("\033[?25h")

    ticker := time.NewTicker(1 * time.Second)
    
    for {
        select {
        case <-sigChan:
            clearScreen()
            return
        case <-ticker.C:
            clearScreen()
            elapsed := time.Since(startTime) - pausedDuration
            hours := int(elapsed.Hours())
            minutes := int(elapsed.Minutes()) % 60
            seconds := int(elapsed.Seconds()) % 60
            
            fmt.Print("\033[1;1H")
            fmt.Println(getASCIITime(hours, minutes, seconds))
            
            if isPaused {
                fmt.Print("\033[1;1H") 
                fmt.Println(strings.Join(pausedText, "\n"))
            }
        }
    }
}

