package main

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	"github.com/qmuntal/stateless"
)

type ctxKey int

const (
	ctxKeyPhoneID ctxKey = iota
)

func ctxWithPhoneID(ctx context.Context, phoneID PhoneID) context.Context {
	return context.WithValue(ctx, ctxKeyPhoneID, phoneID)
}

func phoneIDFromCtx(ctx context.Context) (PhoneID, bool) {
	phoneID, ok := ctx.Value(ctxKeyPhoneID).(PhoneID)
	return phoneID, ok
}

type trigger string

const (
	triggerCallDialed             trigger = "CallDialed"
	triggerCallConnected          trigger = "CallConnected"
	triggerLeftMessage            trigger = "LeftMessage"
	triggerPlacedOnHold           trigger = "PlacedOnHold"
	triggerTakenOffHold           trigger = "TakenOffHold"
	triggerPhoneHurledAgainstWall trigger = "PhoneHurledAgainstWall"
	triggerMuteMicrophone         trigger = "MuteMicrophone"
	triggerUnmuteMicrophone       trigger = "UnmuteMicrophone"
	triggerSetVolume              trigger = "SetVolume"
)

type state string

const (
	stateOffHook        state = "OffHook"
	stateRinging        state = "Ringing"
	stateConnected      state = "Connected"
	stateOnHold         state = "OnHold"
	statePhoneDestroyed state = "PhoneDestroyed"
)

type StateAccessor func(context.Context) (interface{}, error)
type StateMutator func(context.Context, interface{}) error

// PhoneStates holds the phone states
type PhoneStates struct {
	sm      *stateless.StateMachine
	logRepo *LogRepository
}

// NewPhoneStates is a factory for PhoneStates
func NewPhoneStates() *PhoneStates {
	logRepo := newLogRepo()
	stateAccessor := newStateAccessor(logRepo)
	stateMutator := newStateMutator(stateAccessor, logRepo)

	sm := stateless.NewStateMachineWithExternalStorage(
		stateAccessor, stateMutator, stateless.FiringQueued,
	)

	sm.SetTriggerParameters(triggerSetVolume, reflect.TypeOf(0))
	sm.SetTriggerParameters(triggerCallDialed, reflect.TypeOf(""))

	sm.Configure(stateOffHook).
		Permit(triggerCallDialed, stateRinging)

	sm.Configure(stateRinging).
		OnEntryFrom(triggerCallDialed, func(_ context.Context, args ...interface{}) error {
			onDialed(args[0].(string))
			return nil
		}).
		Permit(triggerCallConnected, stateConnected)

	sm.Configure(stateConnected).
		OnEntry(startCallTimer).
		OnExit(func(_ context.Context, _ ...interface{}) error {
			stopCallTimer()
			return nil
		}).
		InternalTransition(triggerMuteMicrophone, func(_ context.Context, _ ...interface{}) error {
			onMute()
			return nil
		}).
		InternalTransition(triggerUnmuteMicrophone, func(_ context.Context, _ ...interface{}) error {
			onUnmute()
			return nil
		}).
		InternalTransition(triggerSetVolume, func(_ context.Context, args ...interface{}) error {
			onSetVolume(args[0].(int))
			return nil
		}).
		Permit(triggerLeftMessage, stateOffHook).
		Permit(triggerPlacedOnHold, stateOnHold)

	sm.Configure(stateOnHold).
		SubstateOf(stateConnected).
		Permit(triggerTakenOffHold, stateConnected).
		Permit(triggerPhoneHurledAgainstWall, statePhoneDestroyed)

	sm.Configure(statePhoneDestroyed).
		OnEntry(func(_ context.Context, _ ...interface{}) error {
			fmt.Println("phone wrecked!")
			return nil
		})

	return &PhoneStates{sm: sm, logRepo: logRepo}
}

func newStateAccessor(logRepo *LogRepository) StateAccessor {
	return func(ctx context.Context) (interface{}, error) {
		phoneID, ok := phoneIDFromCtx(ctx)
		if !ok {
			return nil, errors.New("phone ID not in context")
		}

		last := logRepo.GetLastOrInsert(phoneID, stateOffHook)
		return last.To, nil
	}
}

func newStateMutator(stateAccessor StateAccessor, logrepo *LogRepository) StateMutator {
	return func(ctx context.Context, nextState interface{}) error {
		// new state
		next, ok := nextState.(state)
		if !ok {
			return fmt.Errorf("next state invalid type: %T", nextState)
		}

		currentState, err := stateAccessor(ctx)
		if err != nil {
			return err
		}

		// current state
		current, ok := currentState.(state)
		if !ok {
			return fmt.Errorf("current state invalid type %T", currentState)
		}

		// ignore if it's the same state
		if next == currentState {
			return nil
		}

		phoneID, ok := phoneIDFromCtx(ctx)
		if !ok {
			return errors.New("phone ID not in context")
		}

		err = logrepo.InsertPhoneLog(phoneID, current, next)
		if err != nil {
			return errors.Wrap(err, "error saving state")
		}

		return nil
	}
}

// GetPhoneLogs ...
func (ps *PhoneStates) GetPhoneLogs(phone Phone) []Log {
	return ps.logRepo.GetPhoneLogs(phone.ID)
}

// TriggerCallDialed ...
func (ps *PhoneStates) TriggerCallDialed(ctx context.Context, phoneID PhoneID, callee string) error {
	return ps.sm.FireCtx(ctxWithPhoneID(ctx, phoneID), triggerCallDialed, callee)
}

// TriggerCallConnected ...
func (ps *PhoneStates) TriggerCallConnected(ctx context.Context, phoneID PhoneID) error {
	return ps.sm.FireCtx(ctxWithPhoneID(ctx, phoneID), triggerCallConnected)
}

// TriggerSetVolume ...
func (ps *PhoneStates) TriggerSetVolume(ctx context.Context, phoneID PhoneID, volume int) error {
	return ps.sm.FireCtx(ctxWithPhoneID(ctx, phoneID), triggerSetVolume, volume)
}

// TriggerPlaceOnHold ...
func (ps *PhoneStates) TriggerPlaceOnHold(ctx context.Context, phoneID PhoneID) error {
	return ps.sm.FireCtx(ctxWithPhoneID(ctx, phoneID), triggerPlacedOnHold)
}

// TriggerMuteMicrophone ...
func (ps *PhoneStates) TriggerMuteMicrophone(ctx context.Context, phoneID PhoneID) error {
	return ps.sm.FireCtx(ctxWithPhoneID(ctx, phoneID), triggerMuteMicrophone)
}

// TriggerUnmuteMicrophone ...
func (ps *PhoneStates) TriggerUnmuteMicrophone(ctx context.Context, phoneID PhoneID) error {
	return ps.sm.FireCtx(ctxWithPhoneID(ctx, phoneID), triggerUnmuteMicrophone)
}

// TriggerTakenOffHold ...
func (ps *PhoneStates) TriggerTakenOffHold(ctx context.Context, phoneID PhoneID) error {
	return ps.sm.FireCtx(ctxWithPhoneID(ctx, phoneID), triggerTakenOffHold)
}

// TriggerPhoneHurledAgainstWall ...
func (ps *PhoneStates) TriggerPhoneHurledAgainstWall(ctx context.Context, phoneID PhoneID) error {
	return ps.sm.FireCtx(ctxWithPhoneID(ctx, phoneID), triggerPhoneHurledAgainstWall)
}

func onSetVolume(volume int) {
	fmt.Printf("Volume set to %d!\n", volume)
}

func onUnmute() {
	fmt.Println("Microphone unmuted!")
}

func onMute() {
	fmt.Println("Microphone muted!")
}

func onDialed(callee string) {
	fmt.Printf("[Phone Call] placed for : [%s]\n", callee)
}

func startCallTimer(_ context.Context, _ ...interface{}) error {
	fmt.Printf("[Timer:] Call started at %s hours\n", currentTime())
	return nil
}

func stopCallTimer() {
	fmt.Printf("[Timer:] Call ended at %s hours\n", currentTime())
}

func currentTime() string {
	hours, minutes, _ := time.Now().Clock()
	return fmt.Sprintf("%d:%02d", hours, minutes)
}
