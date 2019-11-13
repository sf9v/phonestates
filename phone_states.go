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

const (
	triggerCallDialed             = "CallDialed"
	triggerCallConnected          = "CallConnected"
	triggerLeftMessage            = "LeftMessage"
	triggerPlacedOnHold           = "PlacedOnHold"
	triggerTakenOffHold           = "TakenOffHold"
	triggerPhoneHurledAgainstWall = "PhoneHurledAgainstWall"
	triggerMuteMicrophone         = "MuteMicrophone"
	triggerUnmuteMicrophone       = "UnmuteMicrophone"
	triggerSetVolume              = "SetVolume"
)

const (
	stateOffHook        = "OffHook"
	stateRinging        = "Ringing"
	stateConnected      = "Connected"
	stateOnHold         = "OnHold"
	statePhoneDestroyed = "PhoneDestroyed"
)

// Phone is a phone
type Phone struct {
	ID int
}

// PhoneLog contains the information about the state transition of a phone
type PhoneLog struct {
	ID      int
	PhoneID int
	From    string
	To      string
	Remarks string
}

// PhoneLogs are phone logs
type PhoneLogs map[int][]PhoneLog

// GetLastOrInsert gets the last log for the phone or inserts if no log is found
func (pl PhoneLogs) GetLastOrInsert(phoneID int, initialState string) PhoneLog {
	logs, ok := pl[phoneID]
	if !ok {
		// insert
		pl[phoneID] = []PhoneLog{
			{
				ID:      1,
				PhoneID: phoneID,
				From:    "",
				To:      initialState,
				Remarks: "Initial state",
			},
		}

		return pl[phoneID][0]
	}

	return logs[len(logs)-1]
}

// GetPhoneLogs gives you all the logs for the phone
func (pl PhoneLogs) GetPhoneLogs(phoneID int) []PhoneLog {
	return pl[phoneID]
}

// InsertPhoneLog inserts a log for the phone
func (pl PhoneLogs) InsertPhoneLog(phoneID int, from, to string) error {
	logs, ok := pl[phoneID]
	if !ok {
		pl[phoneID] = []PhoneLog{}
	}

	log := PhoneLog{
		ID:      len(logs) + 1,
		PhoneID: phoneID,
		From:    from,
		To:      to,
		Remarks: fmt.Sprintf("from %q to %q state", from, to),
	}

	pl[phoneID] = append(logs, log)

	return nil
}

// PhoneStates holds the phone states
type PhoneStates struct {
	stateMachine *stateless.StateMachine
	phoneLogs    PhoneLogs
}

// NewPhoneStates is a factory for PhoneStates
func NewPhoneStates() *PhoneStates {
	phoneStates := &PhoneStates{
		phoneLogs: map[int][]PhoneLog{},
	}

	stateMachine := stateless.NewStateMachineWithExternalStorage(
		func(ctx context.Context) (interface{}, error) {
			return stateAccessor(ctx, phoneStates)
		},
		func(ctx context.Context, newState interface{}) error {
			return stateMutator(ctx, phoneStates, newState)
		},
		stateless.FiringQueued,
	)

	stateMachine.SetTriggerParameters(triggerSetVolume, reflect.TypeOf(0))
	stateMachine.SetTriggerParameters(triggerCallDialed, reflect.TypeOf(""))

	stateMachine.Configure(stateOffHook).
		Permit(triggerCallDialed, stateRinging)

	stateMachine.Configure(stateRinging).
		OnEntryFrom(triggerCallDialed, func(_ context.Context, args ...interface{}) error {
			onDialed(args[0].(string))
			return nil
		}).
		Permit(triggerCallConnected, stateConnected)

	stateMachine.Configure(stateConnected).
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

	stateMachine.Configure(stateOnHold).
		SubstateOf(stateConnected).
		Permit(triggerTakenOffHold, stateConnected).
		Permit(triggerPhoneHurledAgainstWall, statePhoneDestroyed)

	stateMachine.Configure(statePhoneDestroyed).
		OnEntry(func(_ context.Context, _ ...interface{}) error {
			fmt.Println("phone wrecked!")
			return nil
		})

	phoneStates.stateMachine = stateMachine

	return phoneStates
}

func stateAccessor(ctx context.Context, phoneStates *PhoneStates) (currentState interface{}, err error) {
	phoneID, ok := ctx.Value(ctxKeyPhoneID).(int)
	if !ok {
		return nil, errors.New("invalid primary keys, perhaps you have forgotten to pass phone ID on context?")
	}

	last := phoneStates.phoneLogs.GetLastOrInsert(phoneID, stateOffHook)

	return last.To, nil
}

func stateMutator(ctx context.Context, phoneStates *PhoneStates, ns interface{}) error {
	// new state
	newState, ok := ns.(string)
	if !ok {
		return fmt.Errorf("new state is invalid: %v", newState)
	}

	cs, err := phoneStates.stateMachine.State(ctx)
	if err != nil {
		return err
	}

	// current state
	currentState, ok := cs.(string)
	if !ok {
		return fmt.Errorf("current state is invalid %v", cs)
	}

	// ignore if it's the same state
	if newState == currentState {
		return nil
	}

	phoneID, ok := ctx.Value(ctxKeyPhoneID).(int)
	if !ok {
		return errors.New("invalid primary keys, perhaps you have forgotten to pass phone ID on context?")
	}

	err = phoneStates.phoneLogs.InsertPhoneLog(phoneID, currentState, newState)
	if err != nil {
		return errors.Wrap(err, "error saving state")
	}

	return nil
}

// GetPhoneLogs ...
func (ps *PhoneStates) GetPhoneLogs(phone Phone) []PhoneLog {
	return ps.phoneLogs.GetPhoneLogs(phone.ID)
}

// TriggerCallDialed ...
func (ps *PhoneStates) TriggerCallDialed(ctx context.Context, phone Phone, callee string) error {
	ctx = context.WithValue(ctx, ctxKeyPhoneID, phone.ID)
	return ps.stateMachine.FireCtx(ctx, triggerCallDialed, callee)
}

// TriggerCallConnected ...
func (ps *PhoneStates) TriggerCallConnected(ctx context.Context, phone Phone) error {
	ctx = context.WithValue(ctx, ctxKeyPhoneID, phone.ID)
	return ps.stateMachine.FireCtx(ctx, triggerCallConnected)
}

// TriggerSetVolume ...
func (ps *PhoneStates) TriggerSetVolume(ctx context.Context, phone Phone, volume int) error {
	ctx = context.WithValue(ctx, ctxKeyPhoneID, phone.ID)
	return ps.stateMachine.FireCtx(ctx, triggerSetVolume, volume)
}

// TriggerPlaceOnHold ...
func (ps *PhoneStates) TriggerPlaceOnHold(ctx context.Context, phone Phone) error {
	ctx = context.WithValue(ctx, ctxKeyPhoneID, phone.ID)
	return ps.stateMachine.FireCtx(ctx, triggerPlacedOnHold)
}

// TriggerMuteMicrophone ...
func (ps *PhoneStates) TriggerMuteMicrophone(ctx context.Context, phone Phone) error {
	ctx = context.WithValue(ctx, ctxKeyPhoneID, phone.ID)
	return ps.stateMachine.FireCtx(ctx, triggerMuteMicrophone)
}

// TriggerUnmuteMicrophone ...
func (ps *PhoneStates) TriggerUnmuteMicrophone(ctx context.Context, phone Phone) error {
	ctx = context.WithValue(ctx, ctxKeyPhoneID, phone.ID)
	return ps.stateMachine.FireCtx(ctx, triggerUnmuteMicrophone)
}

// TriggerTakenOffHold ...
func (ps *PhoneStates) TriggerTakenOffHold(ctx context.Context, phone Phone) error {
	ctx = context.WithValue(ctx, ctxKeyPhoneID, phone.ID)
	return ps.stateMachine.FireCtx(ctx, triggerTakenOffHold)
}

// TriggerPhoneHurledAgainstWall ...
func (ps *PhoneStates) TriggerPhoneHurledAgainstWall(ctx context.Context, phone Phone) error {
	ctx = context.WithValue(ctx, ctxKeyPhoneID, phone.ID)
	return ps.stateMachine.FireCtx(ctx, triggerPhoneHurledAgainstWall)
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
