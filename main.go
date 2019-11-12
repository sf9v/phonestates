package main

import (
	"context"
	"fmt"
	"log"
)

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	var (
		err error
		ctx = context.Background()
	)

	phoneStates := NewPhoneStates()

	phones := []Phone{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	names := []string{"Stamp", "Ricka", "Marie"}

	for index, phone := range phones {
		err = phoneStates.TriggerCallDialed(ctx, phone, names[index])
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerCallConnected(ctx, phone)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerSetVolume(ctx, phone, 2)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerPlaceOnHold(ctx, phone)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerMuteMicrophone(ctx, phone)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerUnmuteMicrophone(ctx, phone)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerTakenOffHold(ctx, phone)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerSetVolume(ctx, phone, 11)
		checkErr(err)

	}

	for _, phone := range phones {
		err = phoneStates.TriggerPlaceOnHold(ctx, phone)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerPhoneHurledAgainstWall(ctx, phone)
		checkErr(err)
	}

	for _, phone := range phones {
		logs := phoneStates.GetPhoneLogs(phone)
		for _, log := range logs {
			fmt.Printf("PhoneID: %d, Log ID: %d, Remarks: %s\n", log.PhoneID, log.ID, log.Remarks)
		}
		fmt.Println()
	}
}
