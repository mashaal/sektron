package sequencer

// Track contains a track state.
type Track interface {
	Steps() []*step
	CurrentStep() int
	IsActive() bool
	IsCurrentStepActive() bool
}

type track struct {
	steps []*step

	// The pulse defines the current position of the playhead in the track.
	// Each time the clock ticks, we increment the pulse.
	// pulse ranges from 0 to len(steps) * pulsesPerStep (check clock.go).
	// Because each track can have a different number of steps, track pulses
	// are not always synchronized.
	pulse int

	// A track can be assigned to a specific instrument's device and channel.
	// For a midi instrument, they are related to a specific midi device and
	// a specific midi channel.
	device  int
	channel uint8

	// Each track starts a goroutine to handle its pulse progression and step
	// triggering, by using the trig chan at each clock tick.
	// On track removal, we use the done chan to terminate the goroutine.
	trig chan struct{}
	done chan struct{}

	// An inactive will progression like an active track, but will not trigger
	// any steps.
	active bool

	// The next attributes defines the note parameters for the instrument and
	// can be overriden per step (check step.go).
	//  - length defines for how long (pulse value) the note should be played
	//  - chord holds all the notes that should be played
	//  - velocity defines how loud a note should be played
	//  - probability defines the chances that the note will be played
	length      int
	chord       []uint8
	velocity    uint8
	probability int
}

// Steps returns all the track steps.
func (t track) Steps() []*step {
	return t.steps
}

// CurrentStep returns the step where the pulse is right now.
func (t track) CurrentStep() int {
	return t.pulse / pulsesPerStep
}

// IsActive returns true if the track is active.
func (t track) IsActive() bool {
	return t.active
}

// IsCurrentStepActive returns true if the current step is active.
func (t track) IsCurrentStepActive() bool {
	if !t.active || len(t.steps) < t.CurrentStep() {
		return false
	}
	return t.steps[t.CurrentStep()].IsActive()
}

func (t *track) start() {
	t.trig = make(chan struct{})
	t.done = make(chan struct{})
	go func(track *track) {
		for {
			select {
			case <-track.trig:
				track.trigger()
			case <-track.done:
				return
			}
		}
	}(t)
}

// tick will trigger the track at each clock tick.
func (t *track) tick() {
	t.trig <- struct{}{}
}

func (t *track) close() {
	defer close(t.done)
	defer close(t.trig)
	t.done <- struct{}{}
}

// trigger goes over each steps and trigger them or stop them if we're at their
// starting or ending pulse. They are calculated relative to the pulse, using
// the length and offset parameters (check step.go)
func (t *track) trigger() {
	for i, step := range t.steps {
		if t.active && step.isStartingPulse() {
			step.trigger()
		}

		// To avoid 2 steps of the same track being triggered at the same time,
		// we always check the next step. If it's active, we stop the current
		// step, even if it's supposed to play longer.
		if step.isEndingPulse() || (i != t.CurrentStep() && t.isStepForNextPulseActive()) {
			step.reset()
		}
	}

	t.pulse++

	// Go back to the beginning if we reach the end of the track.
	if t.pulse == pulsesPerStep*len(t.steps) {
		t.pulse = 0
	}
}

func (t track) stepForNextPulse() int {
	return (t.pulse + 1) % (pulsesPerStep * len(t.steps)) / pulsesPerStep
}

func (t track) isStepForNextPulseActive() bool {
	return t.steps[t.stepForNextPulse()].active
}

// reset move back the pulse to the beginning, and stops all the already
// triggered steps.
func (t *track) reset() {
	t.pulse = 0
	for _, step := range t.steps {
		step.reset()
	}
}
