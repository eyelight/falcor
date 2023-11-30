package falcor

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
)

var falcor dragon

type rider struct {
	channel       *chan os.Signal   // the channel on which the rider will be notified of signals
	signal        os.Signal         // the signal this rider will respond to
	funcs         map[string]func() // the functions to be executed keyed by their nicknames
	order         []string          // a slice of func names in order of calls to Mount()
	executionMode mode              // Concurrent (default), FIFO, LIFO
}

type Rider interface {
	String() string
	Sequence() string
	Funcs() string
	Execution(m mode)
	Mode() mode
}

type dragon struct {
	sync.Mutex
	verbose bool
	flying  bool
	riders  []rider
}

type Dragon interface {
	Fly()
	Land()
	Riders() int
	Rider(os.Signal) Rider
	Mount(sig os.Signal, fn string, handler func())
	Dismount(sig os.Signal, fn string)
}

type Config struct {
	Verbose bool
}

type mode int

const (
	Concurrent mode = iota // spawn a goroutine for each signal responder func
	FIFO                   // execute sequentially in order of calls to Mount
	LIFO                   // execute sequentially in reverse order of calls to Mount
)

// WithLuck resets & configures the package level dragon & returns a pointer to it
func WithLuck(c Config) Dragon {
	falcor = dragon{
		verbose: c.Verbose,
		flying:  false,
		riders:  make([]rider, 0),
	}
	return &falcor
}

// Luck summons the package level dragon in case you lose your pointer
func Luck() Dragon {
	return &falcor
}

// Mount binds a function to an os.Signal as a rider on your luck dragon
func (f *dragon) Mount(sig os.Signal, fn string, handler func()) {
	f.Lock()
	defer f.Unlock()

	// check for prior existence of sig in riders
	if len(f.riders) > 0 {
		for i, r := range f.riders {
			if r.signal == sig {
				// add the handler to the existing rider's function map & order slice
				r.addFunc(fn, handler)
				f.riders[i] = r
				return
			}
		}
	}

	// if sig is a new signal, create a new rider for it & add to falcor
	r := rider{
		channel: nil,
		signal:  sig,
		order:   make([]string, 0),
		funcs:   make(map[string]func()),
	}
	// add to function map & order slice
	r.addFunc(fn, handler)

	// mount falcor
	f.riders = append(f.riders, r)
}

// Dismount removes a function from your luck dragon
func (f *dragon) Dismount(sig os.Signal, fn string) {
	f.Lock()
	defer f.Unlock()

	for i, r := range f.riders {
		if r.containsSignal(sig) {
			// remove the function with the specified key from the rider's map & order slice
			r.removeFunc(fn)
			f.riders[i] = r

			// if the function map is now empty, close its channel & remove the rider
			if len(r.funcs) == 0 {
				if r.channel != nil {
					close(*r.channel) // terminate the goroutine spawned in Fly()
				}
				f.riders = append(f.riders[:i], f.riders[i+1:]...)
			}
			return
		}
	}
}

// Fly starts the luck dragon's syscall responder
func (f *dragon) Fly() {
	f.Lock()
	defer f.Unlock()

	if f.flying {
		if f.verbose {
			log.Printf("[Falcor] INFO: Fly was called but luck dragon was already flying")
		}
		return
	}
	if len(f.riders) <= 0 {
		if f.verbose {
			log.Printf("[Falcor] INFO: Fly was called without any riders on the luck dragon")
		}
		return
	}
	for i := range f.riders {
		if f.verbose {
			log.Printf("[Falcor] Acquiring notification channel for signal %s", f.riders[i].String())
		}
		if f.riders[i].channel == nil {
			ch := make(chan os.Signal, 1)
			signal.Notify(ch, f.riders[i].signal)
			f.riders[i].channel = &ch
		}

		// spawn one goroutine per handled signal
		go f.fly(&f.riders[i])
	}
	f.flying = true
}

// fly listens for an os signal and executes the rider's functions
func (f *dragon) fly(r *rider) {
	if f.verbose {
		log.Printf("[Falcor] Starting goroutine for rider %s", r.String())
	}
	defer signal.Stop(*r.channel)

	for {
		sig, ok := <-*r.channel
		if !ok {
			if f.verbose {
				log.Printf("[Falcor] Stopping goroutine for Rider %s", r.String())
			}
			return // exit upon channel closure
		}

		if f.verbose {
			log.Printf("[Falcor] Received signal '%s' and will execute %v funcs", sig.String(), len(r.funcs))
		}

		switch r.executionMode {
		case LIFO: // stack-like according to calls to Mount
			for i := len(r.order) - 1; i >= 0; i-- {
				if f.verbose {
					log.Printf("[Falcor] Rider '%s' sequentially executing %s", r.signal.String(), r.order[i])
				}
				r.funcs[r.order[i]]()
				continue
			}
		case FIFO: // queue-like according to calls to Mount
			for _, fn := range r.order {
				if f.verbose {
					log.Printf("[Falcor] Rider '%s' sequentially executing %s", r.signal.String(), fn)
				}
				r.funcs[fn]()
			}
		default: // concurrently
			for fn, fnc := range r.funcs {
				if f.verbose {
					log.Printf("[Falcor] Rider '%s' concurrently executing %s", r.signal.String(), fn)
				}
				go fnc()
			}
		}
	}
}

// Land closes each rider's channel, stopping each associated goroutine, thus landing falcor
func (f *dragon) Land() {
	f.Lock()
	defer f.Unlock()

	if !f.flying {
		return
	}

	for i, r := range f.riders {
		if r.channel != nil {
			close(*r.channel)
		}
		f.riders[i].channel = nil
	}
	f.flying = false
}

func (f *dragon) Riders() int {
	return len(f.riders)
}

func (f *dragon) Rider(s os.Signal) Rider {
	for i, r := range f.riders {
		if r.signal == s {
			return &f.riders[i]
		}
	}
	return nil
}

// String returns the string value of the rider's os signal
func (r *rider) String() string {
	return r.signal.String()
}

// Funcs returns the names of all funcs the rider will run
func (r *rider) Funcs() string {
	funcs := make([]string, 0)
	for fn := range r.funcs {
		funcs = append(funcs, fn)
	}
	return strings.Join(funcs, ", ")
}

func (r *rider) Sequence() string {
	var sep string
	switch r.executionMode {
	case Concurrent:
		sep = " ∞ "
	case FIFO:
		sep = " → "
	case LIFO:
		sep = " ← "
	}
	return strings.Join(r.order, sep)
}

// Execution modifies the executionMode to either Concurrent, FIFO, or LIFO
func (r *rider) Execution(m mode) {
	r.executionMode = m
	if falcor.verbose {
		switch m {
		case Concurrent:
			log.Printf("[Falcor] Rider %s will execute concurrently", r.String())
		case FIFO:
			log.Printf("[Falcor] Rider %s will execute queue-like", r.String())
		case LIFO:
			log.Printf("[Falcor] Rider %s will execute stack-like", r.String())
		}
	}
}

// Mode returns the rider's current execution mode
func (r *rider) Mode() mode {
	return r.executionMode
}

// contains signal returns true if a rider's signal matches the passed-in signal
func (r *rider) containsSignal(sig os.Signal) bool {
	return r.signal == sig
}

// addFunc adds a function to the order slice & function map
func (r *rider) addFunc(fn string, handler func()) {
	r.order = append(r.order, fn)
	r.funcs[fn] = handler
}

// removeFunc removes the function associated with the passed-in function name
// from the order slice & function map
func (r *rider) removeFunc(fn string) {
	// remove from function map
	delete(r.funcs, fn)

	// remove from order slice
	for i, v := range r.order {
		if v == fn {
			r.order = append(r.order[:i], r.order[i+1:]...)
			break
		}
	}
}
