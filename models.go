package main

import "time"

// Represents user data form
type Form struct {
	ID                            int
	DeparturePoint                string
	ArrivalPoint                  string
	DepartureDate                 time.Time
	CarriageType                  string
	NumberOfPassengers            int    // invariant: 1..6
	CompartmentNumber             int    // invariant: 1..9
	ShelfType                     string // invariant: one of "any", "top", "bottom"
	NumberOfPassengersTopShefl    int    // invariant: <= NumberOfPassengers
	NumberOfPassengersBottomShefl int    // invariant: <= NumberOfPassengers
	TrackPriceChange              bool
	SuggestSimilarSeats           bool
}

type FormUpdate struct {
	DeparturePoint                *string
	ArrivalPoint                  *string
	DepartureDate                 *time.Time
	CarriageType                  *string
	NumberOfPassengers            *int
	CompartmentNumber             *int
	ShelfType                     *string
	NumberOfPassengersTopShefl    *int
	NumberOfPassengersBottomShefl *int
	TrackPriceChange              *bool
	SuggestSimilarSeats           *bool
}

type Session struct {
	Step    int
	Command string // invariant: one of "none", and other
	Forms   []Form
}

type SessionUpdate struct {
	Step    *int
	Command *string
}
