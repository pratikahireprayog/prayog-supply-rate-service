package naqel

import (
	"encoding/xml"
)

// NaqelGetRateRequest represents the SOAP request structure for Naqel GetRate API
type NaqelGetRateRequest struct {
	XMLName xml.Name `xml:"soap:Envelope"`
	SoapNS  string   `xml:"xmlns:soap,attr"`
	XsiNS   string   `xml:"xmlns:xsi,attr"`
	XsdNS   string   `xml:"xmlns:xsd,attr"`
	Body    NaqelGetRateBody
}

type NaqelGetRateBody struct {
	XMLName xml.Name         `xml:"soap:Body"`
	GetRate NaqelGetRateBodyContent
}

type NaqelGetRateBodyContent struct {
	XMLName    xml.Name        `xml:"GetRate"`
	Namespace  string          `xml:"xmlns,attr"`
	ClientInfo NaqelClientInfo `xml:"ClientInfo"`
	Weight     float64         `xml:"Weight"`
	LoadTypeID int             `xml:"LoadTypeID"`
	FromDate   string          `xml:"FromDate"`
	Origin     string          `xml:"Origin"`
	Destination string         `xml:"Destination"`
}

type NaqelClientInfo struct {
	XMLName       xml.Name           `xml:"ClientInfo"`
	ClientAddress NaqelClientAddress `xml:"ClientAddress"`
	ClientContact NaqelClientContact `xml:"ClientContact"`
	ClientID      string             `xml:"ClientID"`
	Password      string             `xml:"Password"`
	Version       string             `xml:"Version"`
}

type NaqelClientAddress struct {
	XMLName        xml.Name `xml:"ClientAddress"`
	PhoneNumber    string   `xml:"PhoneNumber"`
	NationalAddress string  `xml:"NationalAddress"`
	POBox          string   `xml:"POBox"`
	ZipCode        string   `xml:"ZipCode"`
	Fax            string   `xml:"Fax"`
	Latitude       string   `xml:"Latitude"`
	Longitude      string   `xml:"Longitude"`
	ShipperName    string   `xml:"ShipperName"`
	FirstAddress   string   `xml:"FirstAddress"`
	Location       string   `xml:"Location"`
	CountryCode    string   `xml:"CountryCode"`
	CityCode       string   `xml:"CityCode"`
}

type NaqelClientContact struct {
	XMLName    xml.Name `xml:"ClientContact"`
	Name       string   `xml:"Name"`
	Email      string   `xml:"Email"`
	PhoneNumber string  `xml:"PhoneNumber"`
	MobileNo   string   `xml:"MobileNo"`
}

// NaqelGetRateResponse represents the SOAP response structure from Naqel GetRate API
type NaqelGetRateResponse struct {
	XMLName xml.Name                `xml:"Envelope"`
	Body    NaqelGetRateResponseBody
}

type NaqelGetRateResponseBody struct {
	XMLName            xml.Name                       `xml:"Body"`
	GetRateResponse    NaqelGetRateResponseContent    `xml:"GetRateResponse"`
}

type NaqelGetRateResponseContent struct {
	XMLName         xml.Name               `xml:"GetRateResponse"`
	GetRateResult   NaqelGetRateResult     `xml:"GetRateResult"`
}

type NaqelGetRateResult struct {
	XMLName xml.Name `xml:"GetRateResult"`
	Price   float64  `xml:"Price"`    // API returns Price, not Rate
	Rate    float64  `xml:"Rate"`     // Fallback field
	Currency string  `xml:"Currency"`
	Message  string  `xml:"Message,omitempty"`
	// Add other fields as needed based on actual API response
}

// NaqelLoadType represents a LoadType from the GetLoadTypeList response
type NaqelLoadType struct {
	ID            int    `xml:"ID"`
	Name          string `xml:"Name"`
	ServiceTypeID int    `xml:"ServiceTypeID"`
	ClientID      int    `xml:"ClientID"`
	ServiceType   string `xml:"ServiceType"`
}

