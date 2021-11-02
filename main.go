package main

import (
	"context"
	"fmt"
	"log"
)

func main() {
	var (
		err error
		ctx = context.Background()
	)

	phoneStates := NewPhoneStates()

	phones := []Phone{
		{ID: 1, Name: "Alice"},
		{ID: 2, Name: "John"},
		{ID: 3, Name: "Bob"},
	}

	for _, p := range phones {
		err = phoneStates.TriggerCallDialed(ctx, p.ID, p.Name)
		checkErr(err)
	}

	for _, p := range phones {
		err = phoneStates.TriggerCallConnected(ctx, p.ID)
		checkErr(err)
	}

	for _, p := range phones {
		err = phoneStates.TriggerSetVolume(ctx, p.ID, 2)
		checkErr(err)
	}

	for _, p := range phones {
		err = phoneStates.TriggerPlaceOnHold(ctx, p.ID)
		checkErr(err)
	}

	for _, p := range phones {
		err = phoneStates.TriggerMuteMicrophone(ctx, p.ID)
		checkErr(err)
	}

	for _, p := range phones {
		err = phoneStates.TriggerUnmuteMicrophone(ctx, p.ID)
		checkErr(err)
	}

	for _, p := range phones {
		err = phoneStates.TriggerTakenOffHold(ctx, p.ID)
		checkErr(err)
	}

	for _, p := range phones {
		err = phoneStates.TriggerSetVolume(ctx, p.ID, 11)
		checkErr(err)

	}

	for _, p := range phones {
		err = phoneStates.TriggerPlaceOnHold(ctx, p.ID)
		checkErr(err)
	}

	for _, p := range phones {
		err = phoneStates.TriggerPhoneHurledAgainstWall(ctx, p.ID)
		checkErr(err)
	}

	for _, p := range phones {
		logs := phoneStates.GetPhoneLogs(p)
		fmt.Printf("%s's phone logs\n", p.Name)
		for _, log := range logs {
			fmt.Printf("Log ID: %d, Remarks: %s\n", log.ID, log.Remarks)
		}
		fmt.Println()
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
