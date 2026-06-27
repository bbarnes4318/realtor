package export

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"
)

type ExportOptions struct {
	JobID       string
	State       string
	City        string
	Zip         string
	DedupePhone bool
	DedupeURL   bool
}

// GenerateCSV writes agent details matching the criteria to the provided io.Writer.
func GenerateCSV(db *sql.DB, w io.Writer, opt ExportOptions) error {
	// 1. Query matching agents
	var rows *sql.Rows
	var err error

	queryBase := `
		SELECT a.id, a.person_name, a.first_name, a.last_name, a.title, a.slogan, a.email, a.profile_url, a.website,
		       addr.line, addr.line2, addr.city, addr.state_code, addr.state, addr.postal_code,
		       b.name, o.name, o.email, o.fulfillment_id,
		       oaddr.line, oaddr.line2, oaddr.city, oaddr.state_code, oaddr.state, oaddr.postal_code
		FROM agents a
		LEFT JOIN addresses addr ON a.address_id = addr.id
		LEFT JOIN brokers b ON a.broker_id = b.id
		LEFT JOIN offices o ON a.office_id = o.id
		LEFT JOIN addresses oaddr ON o.address_id = oaddr.id
	`

	if opt.JobID != "" {
		query := queryBase + `
			JOIN job_agents ja ON a.id = ja.agent_id
			WHERE ja.job_id = ?
		`
		rows, err = db.Query(query, opt.JobID)
	} else {
		query := queryBase + `
			WHERE (addr.state_code = ? OR ? = '')
			  AND (addr.city LIKE ? OR ? = '')
			  AND (addr.postal_code = ? OR ? = '')
		`
		cityParam := ""
		if opt.City != "" {
			cityParam = "%" + opt.City + "%"
		}
		rows, err = db.Query(query, opt.State, opt.State, cityParam, opt.City, opt.Zip, opt.Zip)
	}

	if err != nil {
		return fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write headers
	headers := []string{
		"Agent ID", "Full Name", "First Name", "Last Name", "Title", "Slogan", "Email",
		"Brokerage Name", "Office Name", "Office Email",
		"Office Address Line 1", "Office Address Line 2", "Office City", "Office State", "Office Zip",
		"Phones (All)", "Mobile Phone", "Office Phone",
		"Areas Served", "Languages", "License Numbers", "MLS Codes", "Profile URL", "Website", "Social Profiles",
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	seenPhones := make(map[string]bool)
	seenURLs := make(map[string]bool)

	for rows.Next() {
		var aID, aPersonName, aFirstName, aLastName, aTitle, aSlogan, aEmail, aProfileURL, aWebsite sql.NullString
		var addrLine, addrLine2, addrCity, addrStateCode, addrState, addrPostalCode sql.NullString
		var brokerName, officeName, officeEmail sql.NullString
		var officeFulfillmentID sql.NullInt64
		var oaddrLine, oaddrLine2, oaddrCity, oaddrStateCode, oaddrState, oaddrPostalCode sql.NullString

		err := rows.Scan(
			&aID, &aPersonName, &aFirstName, &aLastName, &aTitle, &aSlogan, &aEmail, &aProfileURL, &aWebsite,
			&addrLine, &addrLine2, &addrCity, &addrStateCode, &addrState, &addrPostalCode,
			&brokerName, &officeName, &officeEmail, &officeFulfillmentID,
			&oaddrLine, &oaddrLine2, &oaddrCity, &oaddrStateCode, &oaddrState, &oaddrPostalCode,
		)
		if err != nil {
			return fmt.Errorf("failed to scan agent row: %w", err)
		}

		agentID := aID.String

		// Deduplicate URL check
		if opt.DedupeURL && aProfileURL.Valid && aProfileURL.String != "" {
			if seenURLs[aProfileURL.String] {
				continue
			}
		}

		// Fetch phones
		allPhones, mobilePhone, officePhone, rawPhones, err := fetchAgentPhones(db, agentID, officeFulfillmentID.Int64)
		if err != nil {
			return fmt.Errorf("failed to fetch agent phones: %w", err)
		}

		// Deduplicate Phone check
		if opt.DedupePhone {
			hasDuplicate := false
			for _, pNum := range rawPhones {
				cleanPhone := cleanPhoneNumber(pNum)
				if cleanPhone != "" && seenPhones[cleanPhone] {
					hasDuplicate = true
					break
				}
			}
			if hasDuplicate {
				continue
			}
		}

		// Mark seen
		if aProfileURL.Valid && aProfileURL.String != "" {
			seenURLs[aProfileURL.String] = true
		}
		for _, pNum := range rawPhones {
			cleanPhone := cleanPhoneNumber(pNum)
			if cleanPhone != "" {
				seenPhones[cleanPhone] = true
			}
		}

		// Fetch other details
		languages, err := fetchAgentLanguages(db, agentID)
		if err != nil {
			return err
		}
		areas, err := fetchAgentAreas(db, agentID)
		if err != nil {
			return err
		}
		licenses, err := fetchAgentLicenses(db, agentID)
		if err != nil {
			return err
		}
		mlsCodes, err := fetchAgentMLS(db, agentID)
		if err != nil {
			return err
		}
		socials, err := fetchAgentSocials(db, agentID)
		if err != nil {
			return err
		}

		// office address fallback to agent address if empty
		oLine := oaddrLine.String
		oLine2 := oaddrLine2.String
		oCity := oaddrCity.String
		oState := oaddrStateCode.String
		oZip := oaddrPostalCode.String
		if oLine == "" {
			oLine = addrLine.String
			oLine2 = addrLine2.String
			oCity = addrCity.String
			oState = addrStateCode.String
			oZip = addrPostalCode.String
		}

		record := []string{
			agentID,
			aPersonName.String,
			aFirstName.String,
			aLastName.String,
			aTitle.String,
			aSlogan.String,
			aEmail.String,
			brokerName.String,
			officeName.String,
			officeEmail.String,
			oLine,
			oLine2,
			oCity,
			oState,
			oZip,
			strings.Join(allPhones, ", "),
			mobilePhone,
			officePhone,
			strings.Join(areas, ", "),
			strings.Join(languages, ", "),
			strings.Join(licenses, ", "),
			strings.Join(mlsCodes, ", "),
			aProfileURL.String,
			aWebsite.String,
			strings.Join(socials, ", "),
		}

		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("failed to write csv record: %w", err)
		}
	}

	// Save this export run in history
	exportID := uuid.New().String()
	filename := fmt.Sprintf("export_%s.csv", exportID)
	filtersMap := map[string]string{
		"state": opt.State,
		"city":  opt.City,
		"zip":   opt.Zip,
	}
	filtersJSON, _ := json.Marshal(filtersMap)

	_, _ = db.Exec(`
		INSERT INTO export_history (id, filename, filters, job_id, created_at, deduped)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
	`, exportID, filename, string(filtersJSON), opt.JobID, opt.DedupePhone || opt.DedupeURL)

	return nil
}

func cleanPhoneNumber(phone string) string {
	var sb strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func fetchAgentPhones(db *sql.DB, agentID string, officeFulfillmentID int64) (all []string, mobile string, office string, raw []string, err error) {
	rows, err := db.Query(`
		SELECT p.number, p.type 
		FROM phones p 
		JOIN agent_phones ap ON p.id = ap.phone_id 
		WHERE ap.agent_id = ?
	`, agentID)
	if err != nil {
		return nil, "", "", nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var number, pType sql.NullString
		if err := rows.Scan(&number, &pType); err == nil {
			numStr := number.String
			raw = append(raw, numStr)
			all = append(all, fmt.Sprintf("%s (%s)", numStr, pType.String))
			if strings.ToLower(pType.String) == "mobile" {
				mobile = numStr
			} else if strings.ToLower(pType.String) == "office" {
				office = numStr
			}
		}
	}

	// Fallback to office phones if office phone not found yet
	if office == "" && officeFulfillmentID > 0 {
		var officeID int64
		err = db.QueryRow("SELECT id FROM offices WHERE fulfillment_id = ?", officeFulfillmentID).Scan(&officeID)
		if err == nil {
			oRows, err := db.Query(`
				SELECT p.number 
				FROM phones p 
				JOIN office_phones op ON p.id = op.phone_id 
				WHERE op.office_id = ?
			`, officeID)
			if err == nil {
				for oRows.Next() {
					var number sql.NullString
					if err := oRows.Scan(&number); err == nil {
						office = number.String
						break
					}
				}
				oRows.Close()
			}
		}
	}

	return all, mobile, office, raw, nil
}

func fetchAgentLanguages(db *sql.DB, agentID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT l.name 
		FROM languages l 
		JOIN agent_languages al ON l.id = al.language_id 
		WHERE al.agent_id = ?
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var langs []string
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err == nil {
			langs = append(langs, name.String)
		}
	}
	return langs, nil
}

func fetchAgentAreas(db *sql.DB, agentID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT ar.name || ' (' || ar.state_code || ')'
		FROM areas ar 
		JOIN agent_served_areas asa ON ar.id = asa.area_id 
		WHERE asa.agent_id = ?
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var areas []string
	for rows.Next() {
		var area sql.NullString
		if err := rows.Scan(&area); err == nil {
			areas = append(areas, area.String)
		}
	}
	return areas, nil
}

func fetchAgentLicenses(db *sql.DB, agentID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT fl.license_number || ' (' || fl.state_code || ')'
		FROM feed_licenses fl 
		JOIN agent_feed_licenses afl ON fl.id = afl.feed_license_id 
		WHERE afl.agent_id = ?
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var licenses []string
	for rows.Next() {
		var license sql.NullString
		if err := rows.Scan(&license); err == nil {
			licenses = append(licenses, license.String)
		}
	}
	return licenses, nil
}

func fetchAgentMLS(db *sql.DB, agentID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT mls.abbreviation 
		FROM multiple_listing_services mls 
		JOIN agent_multiple_listing_services amls ON mls.id = amls.multiple_listing_service_id 
		WHERE amls.agent_id = ?
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code sql.NullString
		if err := rows.Scan(&code); err == nil {
			codes = append(codes, code.String)
		}
	}
	return codes, nil
}

func fetchAgentSocials(db *sql.DB, agentID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT href || ' (' || type || ')'
		FROM social_medias 
		WHERE agent_id = ?
	`, agentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var socials []string
	for rows.Next() {
		var social sql.NullString
		if err := rows.Scan(&social); err == nil {
			socials = append(socials, social.String)
		}
	}
	return socials, nil
}
