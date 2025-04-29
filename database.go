package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/dgraph-io/badger/v4"
)

func getDBKey(chatID int64) []byte {
	return []byte(fmt.Sprintf("%d", chatID))
}

// ---- sessions ----
// checks if user has forms
func userHasSession(chatID int64) (bool, error) {
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})

	if err == badger.ErrKeyNotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func createSession(chatID int64) error {
	key := getDBKey(chatID)
	session := Session{
		Step:    0,
		Command: "none",
		Forms:   []Form{},
	}

	jsn, err := json.Marshal(session)
	if err != nil {
		log.Println("Error: marshaling new session: ", err)
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, jsn)
	})
	if err != nil {
		log.Println("Error: could not store new session in DB: ", err)
		return err
	}

	return nil
}

func getSession(chatID int64) (Session, error) {
	var session Session
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})

	if err != nil {
		log.Println("Error: could not read session from DB while getting session: ", err)
		return Session{}, err
	}

	return session, nil
}

func updateSession(chatID int64, update SessionUpdate) error {
	key := getDBKey(chatID)
	var session Session

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})
	if err != nil {
		log.Println("Error: could not read session from DB while updating session: ", err)
		return err
	}

	if update.Step != nil {
		session.Step = *update.Step
	}
	if update.Command != nil {
		session.Command = *update.Command
	}
	if update.FormsStatus != nil {
		session.FormsStatus = *update.FormsStatus
	}

	data, err := json.Marshal(session)
	if err != nil {
		log.Println("Error: could not marshal updated session:", err)
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, data)
	})
	if err != nil {
		log.Println("Error: updating session in DB: ", err)
		return err
	}

	return nil
}

// ---- forms ----

// inserts empty form in user session. must have a session, or will cause error
func insertEmptyForm(chatID int64) error {
	var session Session
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				log.Println("Error(db): user does not have a session while crearting empty form: ", err)
			}

			log.Println("Error(db): could not get item by key while crearting empty form: ", err)
			return err
		}

		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})
	if err != nil && err != badger.ErrKeyNotFound {
		log.Println("Error: could not view user database: ", err)
		return err
	}

	session.Forms = append(session.Forms, Form{ID: len(session.Forms)})

	jsn, err := json.Marshal(session)
	if err != nil {
		log.Println("Error: could not unmarshall session with empty form: ", err)
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, jsn)
	})
	if err != nil {
		log.Println("Error: updating db while creating empty form: ", err)
		return err
	}

	return nil
}

func updateLastForm(chatID int64, update FormUpdate) error {
	var session Session
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})
	if err != nil {
		log.Println("Error: could not get current session: ", err)
		return err
	}

	if len(session.Forms) == 0 {
		log.Println("Error: no forms in session.")
		return fmt.Errorf("no forms in session")
	}

	form := &session.Forms[len(session.Forms)-1]

	if update.DeparturePoint != nil {
		form.DeparturePoint = *update.DeparturePoint
	}
	if update.ArrivalPoint != nil {
		form.ArrivalPoint = *update.ArrivalPoint
	}
	if update.DepartureDate != nil {
		form.DepartureDate = *update.DepartureDate
	}
	if update.CarriageType != nil {
		form.CarriageType = *update.CarriageType
	}
	if update.NumberOfPassengers != nil {
		form.NumberOfPassengers = *update.NumberOfPassengers
	}
	if update.CompartmentNumber != nil {
		form.CompartmentNumber = *update.CompartmentNumber
	}
	if update.ShelfType != nil {
		form.ShelfType = *update.ShelfType
	}
	if update.NumberOfPassengersTopShefl != nil {
		form.NumberOfPassengersTopShefl = *update.NumberOfPassengersTopShefl
	}
	if update.NumberOfPassengersBottomShefl != nil {
		form.NumberOfPassengersBottomShefl = *update.NumberOfPassengersBottomShefl
	}
	if update.TrackPriceChange != nil {
		form.TrackPriceChange = *update.TrackPriceChange
	}
	if update.SuggestSimilarSeats != nil {
		form.SuggestSimilarSeats = *update.SuggestSimilarSeats
	}

	jsn, err := json.Marshal(session)
	if err != nil {
		log.Println("Error: failed to marshal updated session: ", err)
		return err
	}

	err = db.Update(func(txn *badger.Txn) error {
		return txn.Set(key, jsn)
	})
	if err != nil {
		log.Println("Error: failed to update session in db: ", err)
		return err
	}

	return nil
}

func getLastForm(chatID int64) (Form, error) {
	var session Session
	key := getDBKey(chatID)

	err := db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &session)
		})
	})

	if err != nil {
		log.Println("Error: could not read session from DB while getting last form: ", err)
		return Form{}, err
	}

	if len(session.Forms) == 0 {
		log.Println("Error: session has no forms")
		return Form{}, fmt.Errorf("no forms in session")
	}

	return session.Forms[len(session.Forms)-1], nil
}
