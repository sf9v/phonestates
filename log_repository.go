package main

import (
	"fmt"
	"sync"
)

type (
	PhoneID int
	LogID   int
)

// Phone is a phone
type Phone struct {
	ID   PhoneID
	Name string
}

// Log contains the information about the state transition of a phone
type Log struct {
	ID      LogID
	PhoneID PhoneID
	From    state
	To      state
	Remarks string
}

// LogRepository is the phone log repository
type LogRepository struct {
	sync.RWMutex
	phoneLogs map[PhoneID][]Log
}

func newLogRepo() *LogRepository {
	return &LogRepository{
		phoneLogs: make(map[PhoneID][]Log),
	}
}

// GetLastOrInsert gets the last log for the phone or inserts if no log is found
func (repo *LogRepository) GetLastOrInsert(phoneID PhoneID, initial state) Log {
	repo.Lock()
	defer repo.Unlock()

	logs, ok := repo.phoneLogs[phoneID]
	if !ok {
		// insert
		repo.phoneLogs[phoneID] = []Log{{
			ID:      1,
			PhoneID: phoneID,
			From:    "",
			To:      initial,
			Remarks: fmt.Sprintf("Initial %q", initial),
		}}

		return repo.phoneLogs[phoneID][0]
	}

	return logs[len(logs)-1]
}

// GetPhoneLogs gives you all the logs for the phone
func (repo *LogRepository) GetPhoneLogs(phoneID PhoneID) []Log {
	repo.RLock()
	defer repo.RUnlock()

	phoneLogs := repo.phoneLogs[phoneID]
	logs := make([]Log, len(phoneLogs))
	copy(logs, phoneLogs)

	return logs
}

// InsertPhoneLog inserts a log for the phone
func (repo *LogRepository) InsertPhoneLog(phoneID PhoneID, from, to state) error {
	repo.Lock()
	defer repo.Unlock()

	logs, ok := repo.phoneLogs[phoneID]
	if !ok {
		repo.phoneLogs[phoneID] = []Log{}
	}

	log := Log{
		ID:      LogID(len(logs) + 1),
		PhoneID: phoneID,
		From:    from,
		To:      to,
		Remarks: fmt.Sprintf("From %q to %q", from, to),
	}

	repo.phoneLogs[phoneID] = append(logs, log)

	return nil
}
