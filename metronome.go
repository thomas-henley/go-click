package main

import (
	"bufio"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"
)

const (
	clickHiWav = "click_hi.wav"
	clickLoWav = "click_lo.wav"
)

//go:embed click_hi.wav
var clickHiSound embed.FS

//go:embed click_lo.wav
var clickLoSound embed.FS

var clickHiTmp string
var clickLoTmp string

// initSound creates the temporary sound file once
func initSound(embedded embed.FS, wav string, tmp *string) error {
	// Read the embedded sound file
	data, err := embedded.ReadFile(wav)
	if err != nil {
		return fmt.Errorf("reading embedded sound: %v", err)
	}

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "click*.wav")
	if err != nil {
		return fmt.Errorf("creating temp file: %v", err)
	}
	defer tmpFile.Close()

	// Write the sound data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		os.Remove(tmpFile.Name())
		return fmt.Errorf("writing temp file: %v", err)
	}

	*tmp = tmpFile.Name()
	return nil
}

// playClick plays the pre-created sound file
func playClick(tmp_file string) {
	if tmp_file == "" {
		return
	}

	var cmd *exec.Cmd
	switch {
	case commandExists("paplay"):
		cmd = exec.Command("paplay", tmp_file)
	case commandExists("aplay"):
		cmd = exec.Command("aplay", tmp_file)
	case commandExists("afplay"):
		cmd = exec.Command("afplay", tmp_file)
	case commandExists("powershell"):
		cmd = exec.Command("powershell", "-c", fmt.Sprintf("(New-Object Media.SoundPlayer '%s').PlaySync()", tmp_file))
	default:
		return
	}

	cmd.Start() // Don't wait for it to finish
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: metronome <beats_per_measure> <bpm>")
		fmt.Println("Example: metronome 4 120")
		os.Exit(1)
	}

	beatsPerMeasure, err := strconv.Atoi(os.Args[1])
	if err != nil || beatsPerMeasure <= 0 {
		fmt.Println("Error: beats_per_measure must be a positive integer")
		os.Exit(1)
	}

	bpm, err := strconv.Atoi(os.Args[2])
	if err != nil || bpm <= 0 {
		fmt.Println("Error: bpm must be a positive integer")
		os.Exit(1)
	}

	// Calculate interval between beats in milliseconds
	interval := time.Duration(60000/bpm) * time.Millisecond

	// Initialize sound file once
	if err := initSound(clickHiSound, clickHiWav, &clickHiTmp); err != nil {
		fmt.Printf("Warning: Could not initialize sound (%v)\n", err)
	}
	defer func() {
		if clickHiTmp != "" {
			os.Remove(clickHiTmp)
		}
	}()

	// Initialize sound file once
	if err := initSound(clickLoSound, clickLoWav, &clickLoTmp); err != nil {
		fmt.Printf("Warning: Could not initialize sound (%v)\n", err)
	}
	defer func() {
		if clickLoTmp != "" {
			os.Remove(clickLoTmp)
		}
	}()

	fmt.Printf("Metronome: %d beats per measure at %d BPM\n", beatsPerMeasure, bpm)
	fmt.Println("Press Enter to stop...")
	fmt.Println()

	// Hide cursor blinking
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Show cursor again when done

	// Channel to signal when to stop
	stop := make(chan bool)

	// Goroutine to listen for key press
	go func() {
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')
		stop <- true
	}()

	// Metronome loop
	beat := 1
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			fmt.Print("\033[?25h") // Show cursor again
			fmt.Println("\nMetronome stopped.")
			return
		case <-ticker.C:
			if beat == 1 {
				// Start of new measure - clear line and start fresh
				fmt.Print("\r\033[K")
				playClick(clickHiTmp)
			} else {
				playClick(clickLoTmp) // Play sound (no goroutine needed since we use Start())
			}
			fmt.Printf("%d ", beat)
			beat++
			if beat > beatsPerMeasure {
				beat = 1
			}
		}
	}
}

