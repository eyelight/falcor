package main

import (
	"log"
	"os"
	"syscall"
	"time"

	"github.com/eyelight/falcor"
)

func init() {
	c := falcor.Config{
		// Verbose: true, // Your luck dragon can be talkative, or not
	}

	// to use a luck dragon, you'll need to call WithLuck
	atreyu := falcor.WithLuck(c)

	// now system signals can ride falcor carrying a bag of lucky handlers
	atreyu.Mount(syscall.SIGINT, "EXIT", func() {
		log.Println("Handling exit...")
		os.Exit(0)
	})

	// subsequent calls to Mount will add functions to the same
	// by defualt, they will be executed in order of calls to Mount, like a queue
	atreyu.Mount(syscall.SIGINT, "SIGINT-init", func() {
		log.Println("Handling SIGINT-init")
	})

	// You can see your Riders (handled signals)
	log.Printf("(init) Riders: %v", atreyu.Riders())
}

func main() {
	// you can re-aquire your luck dragon with Luck
	atreyu := falcor.Luck()

	// and more calls to Mount with the same signal will affect the same Rider
	atreyu.Mount(syscall.SIGINT, "SIGINT-main", func() {
		log.Println("Handling SIGINT-main")
	})

	// and new signals will ride the same luck dragon
	atreyu.Mount(syscall.SIGHUP, "SIGHUP-main", func() {
		log.Println("Handling SIGHUP")
	})
	log.Printf("(main) Riders: %v", atreyu.Riders())
	log.Printf("[Falcor] Rider %s: Funcs (%s) Sequence (%s)", atreyu.Rider(syscall.SIGINT).String(), atreyu.Rider(syscall.SIGINT).Funcs(), atreyu.Rider(syscall.SIGINT).Sequence())
	log.Printf("[Falcor] Rider %s: Funcs (%s) Sequence (%s)", atreyu.Rider(syscall.SIGHUP).String(), atreyu.Rider(syscall.SIGHUP).Funcs(), atreyu.Rider(syscall.SIGHUP).Sequence())

	// When you're ready to fly, you can
	atreyu.Fly()

	time.Sleep(1 * time.Second)

	// You can Dismount at any time, reducing the number of functions in a Rider
	atreyu.Dismount(syscall.SIGINT, "SIGINT-init")

	// Or removing a Rider entirely if you dismount all your handlers
	// atreyu.Dismount(syscall.SIGHUP, "SIGHUP-main")

	// And if you want your multi-handler Riders to execute like a stack, queue, or concurrent (default), you can do that
	atreyu.Rider(syscall.SIGINT).Execution(falcor.LIFO)

	time.Sleep(10 * time.Second)
	// If you want to stop all your system responders, you can land your luck dragon
	atreyu.Land()

	time.Sleep(3 * time.Second)

	// You can fly again when ready
	atreyu.Fly()

	log.Printf("(main) Riders: %v", atreyu.Riders())
	log.Printf("[Falcor] Rider %s: Funcs (%s) Sequence (%s)", atreyu.Rider(syscall.SIGINT).String(), atreyu.Rider(syscall.SIGINT).Funcs(), atreyu.Rider(syscall.SIGINT).Sequence())

	select {}
}
