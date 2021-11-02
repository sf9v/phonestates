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

	phones := []Phone{{ID: 1}, {ID: 2}, {ID: 3}}

	names := []string{"Alice", "John", "Bo"}

	for index, phone := range phones {
		err = phoneStates.TriggerCallDialed(ctx, phone.ID, names[index])
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerCallConnected(ctx, phone.ID)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerSetVolume(ctx, phone.ID, 2)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerPlaceOnHold(ctx, phone.ID)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerMuteMicrophone(ctx, phone.ID)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerUnmuteMicrophone(ctx, phone.ID)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerTakenOffHold(ctx, phone.ID)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerSetVolume(ctx, phone.ID, 11)
		checkErr(err)

	}

	for _, phone := range phones {
		err = phoneStates.TriggerPlaceOnHold(ctx, phone.ID)
		checkErr(err)
	}

	for _, phone := range phones {
		err = phoneStates.TriggerPhoneHurledAgainstWall(ctx, phone.ID)
		checkErr(err)
	}

	for _, phone := range phones {
		logs := phoneStates.GetPhoneLogs(phone)
		for _, log := range logs {
			fmt.Printf("PhoneID: %d, Log ID: %d, Remarks: %s\n", log.PhoneID, log.ID, log.Remarks)
		}
		fmt.Println()
	}

	fmt.Println(phoneStates.sm.ToGraph())
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
