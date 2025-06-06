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
	CompartmentNumber             []int  // invariant: non-empty list of 1..9
	ShelfType                     string // invariant: one of "Любое", "Указать нижние", "Указать верхние"
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
	CompartmentNumber             *[]int
	ShelfType                     *string
	NumberOfPassengersTopShefl    *int
	NumberOfPassengersBottomShefl *int
	TrackPriceChange              *bool
	SuggestSimilarSeats           *bool
}

type FormState struct {
	Price string
	Date  time.Time
}

type Session struct {
	Step        int
	Command     string // invariant: one of "none", and other
	Forms       []Form
	FormsStatus []FormState
}

type SessionUpdate struct {
	Step        *int
	Command     *string
	FormsStatus *[]FormState
}
