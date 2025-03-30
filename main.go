package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const UserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:98.0) Gecko/20100101 Firefox/98.0"
const AvailabilityURL = "https://cme-dlr.wdprapps.disney.com/availability/api/v2/availabilities/?sku=66282&sku=66283"
const TokenURL = "https://disneyland.disney.go.com/com-shared/api/get-token/"
const OutputFile = "availability.json"

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   string `json:"expires_in"`
}

type AccessToken struct {
	AccessToken string
	ValidUntil  time.Time
}

type CalenderAvailabilityResponse struct {
	Availabilities []SingleDay `json:"calendar-availabilities"`
}

type SingleDay struct {
	Date       string     `json:"date"`
	Facilities []Facility `json:"facilities"`
}

type Facility struct {
	FacilityName string `json:"facilityName"`
	Available    bool   `json:"available"`
}

func main() {
	log.Println("Daemon started, checking availability every hour...")
	var accessToken *AccessToken

	for {
		if accessToken == nil || time.Now().After(accessToken.ValidUntil) {
			log.Println("Requesting new token...")
			var err error
			accessToken, err = GetAccessToken()
			if err != nil {
				log.Println("Failed to get access token:", err)
				continue
			}
		}

		availability, err := QueryAvailability(accessToken.AccessToken)
		if err != nil {
			log.Println("Failed to query availability:", err)
		} else {
			saveAvailability(availability)

			if len(availability.Availabilities) > 0 {
				firstDate := availability.Availabilities[0].Date
				lastDate := availability.Availabilities[len(availability.Availabilities)-1].Date
				log.Println("Availability from", firstDate, "to", lastDate)
			}
		}

		log.Println("Sleeping for 1 hour...")
		time.Sleep(1 * time.Hour)
	}
}

func GetAccessToken() (*AccessToken, error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", TokenURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResponse TokenResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&tokenResponse); err != nil {
		return nil, err
	}

	expiresIn, err := strconv.Atoi(tokenResponse.ExpiresIn)
	if err != nil {
		return nil, err
	}

	return &AccessToken{
		AccessToken: tokenResponse.AccessToken,
		ValidUntil:  time.Now().Add(time.Duration(expiresIn-5) * time.Second),
	}, nil
}

func QueryAvailability(accessTokenString string) (*CalenderAvailabilityResponse, error) {
	req, err := http.NewRequest("GET", AvailabilityURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Authorization", "Bearer "+accessTokenString)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var availability CalenderAvailabilityResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&availability); err != nil {
		return nil, err
	}

	return &availability, nil
}

func saveAvailability(data *CalenderAvailabilityResponse) {
	file, err := os.Create(OutputFile)
	if err != nil {
		log.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(data); err != nil {
		log.Println("Error writing to file:", err)
	}
}
