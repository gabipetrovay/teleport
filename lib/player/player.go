// Package player includes an API to play back recorded sessions.
package player

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/gravitational/trace"
	"github.com/jonboulle/clockwork"
	"github.com/sirupsen/logrus"

	"github.com/gravitational/teleport/api/types/events"
	"github.com/gravitational/teleport/lib/session"
)

type Player struct {
	// read only config fields
	clock     clockwork.Clock
	log       logrus.FieldLogger
	sessionID session.ID
	streamer  Streamer

	emit chan events.AuditEvent

	// playPause holds a channel to be closed when
	// the player transitions from paused to playing,
	// or nil if the player is already playing.
	//
	// This approach mimics a "select-able" condition variable
	// and is inspired by "Rethinking Classical Concurrency Patterns"
	// by Bryan C. Mills (GopherCon 2018): https://www.youtube.com/watch?v=5zXAHh5tJqQ
	playPause chan chan struct{}

	done chan struct{}
}

type Streamer interface {
	StreamSessionEvents(ctx context.Context, sessionID session.ID, startIndex int64) (chan events.AuditEvent, chan error)
}

type Config struct {
	Clock     clockwork.Clock
	Log       logrus.FieldLogger
	SessionID session.ID
	Streamer  Streamer
}

func New(cfg *Config) (*Player, error) {
	if cfg.Streamer == nil {
		return nil, trace.BadParameter("missing Streamer")
	}

	if cfg.SessionID == "" {
		return nil, trace.BadParameter("missing SessionID")
	}

	clk := cfg.Clock
	if clk == nil {
		clk = clockwork.NewRealClock()
	}

	var log logrus.FieldLogger = cfg.Log
	if log == nil {
		l := logrus.New().WithField(trace.Component, "player")
		l.Logger.SetOutput(os.Stdout) // TODO remove
		l.Logger.SetLevel(logrus.DebugLevel)
		log = l
	}

	p := &Player{
		clock:     clk,
		log:       log,
		sessionID: cfg.SessionID,
		streamer:  cfg.Streamer,
		emit:      make(chan events.AuditEvent, 64),
		playPause: make(chan chan struct{}, 1),
		done:      make(chan struct{}),
	}

	// start in a paused state
	p.playPause <- make(chan struct{})

	go p.stream()

	return p, nil
}

// errClosed is an internal error that is used to signal
// that the player has been closed
var errClosed = errors.New("player closed")

func (p *Player) stream() {
	eventsC, errC := p.streamer.StreamSessionEvents(context.TODO(), p.sessionID, 0)
	lastDelay := int64(0)
	for {
		select {
		case <-p.done:
			close(p.emit)
			return
		case err := <-errC:
			// TODO: figure out how to surface the error
			// (probably close the chan and expose a method)
			p.log.Warn(err)
			return
		case evt := <-eventsC:
			if evt == nil {
				p.log.Debug("reached end of playback")
				close(p.emit)
				return
			}

			if err := p.waitWhilePaused(); err != nil {
				p.log.Warn(err)
				close(p.emit)
				return
			}

			delay := getDelay(evt)
			if delay > 0 && delay > lastDelay {
				// TODO: scale delay based on playback speed
				if err := p.applyDelay(time.Duration(delay-lastDelay) * time.Millisecond); err != nil {
					close(p.emit)
					return
				}
				lastDelay = delay
			}

			p.log.Debugf("playing %v (%v)", evt.GetType(), evt.GetID())
			select {
			case p.emit <- evt:
				p.log.Debugf("event %v was played", evt.GetID())
			default:
				p.log.Warnf("dropped event %v, reader too slow", evt.GetID())
			}
		}
	}
}

// Close shuts down the player and cancels any streams that are
// in progress.
func (p *Player) Close() error {
	close(p.done)
	return nil
}

// C returns a read only channel of recorded session events.
// The player manages the timing of events and writes them to the channel
// when they should be rendered. The channel is closed when the player
// has reached the end of playback.
func (p *Player) C() <-chan events.AuditEvent {
	return p.emit
}

// TODO: add an Err() method to be checked after C is closed

// Pause temporarily stops the player from emitting events.
// It is a no-op if playback is currently paused.
func (p *Player) Pause() error {
	p.setPlaying(false)
	return nil
}

// Play starts emitting events. It is used to start playback
// for the first time and to resume playing after the player
// is paused.
func (p *Player) Play() error {
	p.setPlaying(true)
	return nil
}

// applyDelay "sleeps" for d in a manner that
// can be canceled
func (p *Player) applyDelay(d time.Duration) error {
	select {
	case <-p.done:
		return errClosed
	case <-p.clock.After(d):
		return nil
	}
}

func (p *Player) setPlaying(play bool) {
	ch := <-p.playPause
	alreadyPlaying := ch == nil

	if alreadyPlaying && !play {
		ch = make(chan struct{})
	} else if !alreadyPlaying && play {
		// signal waiters who are paused that it's time to resume playing
		close(ch)
		ch = nil
	}

	p.playPause <- ch
}

// waitWhilePaused blocks while the player is in a paused state.
// It returns immediately if the player is currently playing.
func (p *Player) waitWhilePaused() error {
	ch := <-p.playPause
	p.playPause <- ch

	if alreadyPlaying := ch == nil; !alreadyPlaying {
		select {
		case <-p.done:
			return errClosed
		case <-ch:
		}
	}
	return nil
}

// LastPlayed returns the time of the last played event,
// expressed as milliseconds since the start of the session.
func (p *Player) LastPlayed() int64 {
	return 0
}

func getDelay(e events.AuditEvent) int64 {
	switch x := e.(type) {
	case *events.DesktopRecording:
		return x.DelayMilliseconds
	case *events.SessionPrint:
		return x.DelayMilliseconds
	default:
		return int64(0)
	}
}

// TODO: set playback speed
// TODO: seek forward
// TODO: seek backward
