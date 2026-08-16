// Command displayimage shows a PNG on a Waveshare 7.5 inch e-Paper V2 panel.
//
// It draws the image, waits for SIGINT or SIGTERM, then clears the screen to
// white and puts the panel to sleep before exiting.
//
//	displayimage --image beach.png
package main

import (
	"errors"
	"flag"
	"fmt"
	"image/png"
	"os"
	"os/signal"
	"syscall"

	"github.com/ditrytus/epd7in5v2"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run() (err error) {
	imageFile := flag.String("image", "", "input image to display on the screen")
	flag.Parse()

	if *imageFile == "" {
		return fmt.Errorf("missing required flag --image")
	}

	file, err := os.Open(*imageFile)
	if err != nil {
		return err
	}
	img, err := png.Decode(file)
	if err != nil {
		if closeErr := file.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}

	screen := epd7in5v2.NewEPD()
	defer func() {
		if closeErr := screen.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	if err := screen.GPIOInit(); err != nil {
		return err
	}
	if err := screen.Init(); err != nil {
		return err
	}
	if err := screen.DisplayImage(img); err != nil {
		return err
	}
	if err := screen.Sleep(); err != nil {
		return err
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	if err := screen.Init(); err != nil {
		return err
	}
	if err := screen.ClearToWhite(); err != nil {
		return err
	}
	return screen.Sleep()
}
