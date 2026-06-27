package scraper

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/suffer-sami/realtor-scraper/internal/database"
)

func (s *Scraper) executeTransaction(ctx context.Context, txFunc func(context.Context, *sql.Tx, *database.Queries) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	qtx := s.dbQueries.WithTx(tx)

	err = txFunc(ctx, tx, qtx)
	if err != nil {
		return fmt.Errorf("transaction failed: %v", err)
	}
	return tx.Commit()
}

func (s *Scraper) storeAgent(agent Agent) error {
	return s.executeTransaction(context.Background(), func(ctx context.Context, tx *sql.Tx, qtx *database.Queries) error {
		s.logger.Infof("Agent: %s", agent.PersonName)

		dbAgent, err := qtx.GetAgent(ctx, agent.ID)
		if err != nil {
			if err != sql.ErrNoRows {
				return err
			}

			dbAgent, err = qtx.CreateAgent(ctx, database.CreateAgentParams{
				ID:                   agent.ID,
				FirstName:            stringToNullString(agent.FirstName),
				LastName:             stringToNullString(agent.LastName),
				NickName:             stringToNullString(agent.NickName),
				PersonName:           stringToNullString(agent.PersonName),
				Title:                stringToNullString(agent.Title),
				Slogan:               stringToNullString(agent.Slogan),
				Email:                stringToNullString(agent.Email),
				AgentRating:          intToNullInt64(agent.AgentRating),
				Description:          stringToNullString(agent.Description),
				RecommendationsCount: intToNullInt64(agent.RecommendationsCount),
				ReviewCount:          intToNullInt64(agent.ReviewCount),
				LastUpdated:          strToNullTime(agent.LastUpdated, time.RFC1123),
				FirstMonth:           numericToNullInt(agent.FirstMonth),
				FirstYear:            intToNullInt64(agent.AgentRating),
				Photo:                stringToNullString(agent.Photo.Href),
				Video:                stringToNullString(agent.Video),
				ProfileUrl:           stringToNullString(agent.WebURL),
				Website:              stringToNullString(agent.Href),
			})

			if err != nil {
				return err
			}

			if s.saveRawAgents {
				s.logger.Debugf("- raw agent:")
				if jsonStrAgent, err := anyToJsonString(agent); err == nil {
					s.logger.Debugf("	* %s", jsonStrAgent)
					if err := qtx.CreateRawAgent(ctx, database.CreateRawAgentParams{
						AgentID: stringToNullString(dbAgent.ID),
						Data:    stringToNullString(jsonStrAgent),
					}); err != nil {
						s.logger.Errorf("error creating raw agent: %v", err)
					}
				}
			}
		}
		agentId := stringToNullString(dbAgent.ID)

		// Link to job_agents if jobID is specified
		if s.jobID != "" {
			_, err = tx.ExecContext(ctx, `
				INSERT OR IGNORE INTO job_agents (job_id, agent_id, scraped_at) 
				VALUES (?, ?, CURRENT_TIMESTAMP)
			`, s.jobID, dbAgent.ID)
			if err != nil {
				s.logger.Errorf("error linking agent %s to job %s: %v", dbAgent.ID, s.jobID, err)
			}
		}

		s.logger.Debugf("- sales data: %s", agent.RecentlySold.LastSoldDate)
		if err := qtx.CreateSalesData(ctx, database.CreateSalesDataParams{
			Count:        intToNullInt64(agent.RecentlySold.Count),
			Min:          intToNullInt64(agent.RecentlySold.Min),
			Max:          intToNullInt64(agent.RecentlySold.Max),
			LastSoldDate: strToNullTime(agent.RecentlySold.LastSoldDate, time.DateOnly),
			AgentID:      agentId,
		}); err != nil {
			s.logger.Errorf("error creating sales data: %v", err)
		}

		s.logger.Debugf("- listing data: %s", agent.ForSalePrice.LastListingDate)
		if err := qtx.CreateListingsData(ctx, database.CreateListingsDataParams{
			Count:           intToNullInt64(agent.ForSalePrice.Count),
			Min:             intToNullInt64(agent.ForSalePrice.Min),
			Max:             intToNullInt64(agent.ForSalePrice.Max),
			LastListingDate: timeToNullTime(agent.ForSalePrice.LastListingDate),
			AgentID:         agentId,
		}); err != nil {
			s.logger.Errorf("error creating listing data: %v", err)
		}

		s.logger.Debugf("- social medias:")
		for _, socialMedia := range agent.SocialMedias {
			s.logger.Debugf("	* %s", socialMedia.Type)
			if err := qtx.CreateSocialMedia(ctx, database.CreateSocialMediaParams{
				Type:    stringToNullString(socialMedia.Type),
				Href:    stringToNullString(socialMedia.Href),
				AgentID: agentId,
			}); err != nil {
				s.logger.Errorf("error creating social media: %v", err)
			}
		}

		s.logger.Debugf("- feed licences:")
		for _, feedLicense := range agent.FeedLicenses {
			if feedLicense.IsZero() {
				continue
			}
			s.logger.Debugf("	* (%s, %s)", feedLicense.StateCode, feedLicense.Country)
			dbFeedLicenseID, err := qtx.CreateFeedLicense(ctx, database.CreateFeedLicenseParams{
				Country:       stringToNullString(feedLicense.Country),
				LicenseNumber: stringToNullString(feedLicense.LicenseNumber),
				StateCode:     stringToNullString(feedLicense.StateCode),
			})

			if err != nil {
				return err
			}

			if err := qtx.CreateAgentFeedLicense(ctx, database.CreateAgentFeedLicenseParams{
				FeedLicenseID: int64ToNullInt64(dbFeedLicenseID),
				AgentID:       agentId,
			}); err != nil {
				s.logger.Errorf("error creating feed license: %v", err)
			}
		}

		s.logger.Debugf("- mls:")
		for _, mls := range agent.Mls {
			s.logger.Debugf("	* %s", mls.Abbreviation)
			dbMls, err := qtx.GetMultipleListingService(ctx, database.GetMultipleListingServiceParams{
				Abbreviation:  stringToNullString(mls.Abbreviation),
				Type:          stringToNullString(mls.Type),
				MemberID:      stringToNullString(mls.MemberID),
				LicenseNumber: stringToNullString(mls.LicenseNumber),
			})

			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
				dbMls, err = qtx.CreateMultipleListingService(ctx, database.CreateMultipleListingServiceParams{
					Abbreviation:  stringToNullString(mls.Abbreviation),
					LicenseNumber: stringToNullString(mls.LicenseNumber),
					Type:          stringToNullString(mls.Type),
					MemberID:      stringToNullString(mls.MemberID),
					IsPrimary:     boolToNullBool(mls.Primary),
				})

				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentMultipleListingService(ctx, database.CreateAgentMultipleListingServiceParams{
				AgentID:                  agentId,
				MultipleListingServiceID: int64ToNullInt64(dbMls.ID),
			}); err != nil {
				s.logger.Errorf("error creating agent mls: %v", err)
			}
		}
		s.logger.Debugf("- mls history:")
		for _, mls := range agent.MlsHistory {
			s.logger.Debugf("	* %s", mls.Abbreviation)
			dbMls, err := qtx.GetMultipleListingService(ctx, database.GetMultipleListingServiceParams{
				Abbreviation:  stringToNullString(mls.Abbreviation),
				Type:          stringToNullString(mls.Type),
				MemberID:      stringToNullString(mls.Member.ID),
				LicenseNumber: stringToNullString(mls.LicenseNumber),
			})

			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}
				dbMls, err = qtx.CreateMultipleListingService(ctx, database.CreateMultipleListingServiceParams{
					Abbreviation:     stringToNullString(mls.Abbreviation),
					InactivationDate: timeToNullTime(mls.InactivationDate),
					LicenseNumber:    stringToNullString(mls.LicenseNumber),
					IsPrimary:        boolToNullBool(mls.Primary),
					Type:             stringToNullString(mls.Type),
					MemberID:         stringToNullString(mls.Member.ID),
				})

				if err != nil {
					return err
				}
			}

			if dbMls.InactivationDate.Time != mls.InactivationDate {
				s.logger.Debugf("	* %s (update inactivation_date: %v)", mls.Abbreviation, mls.InactivationDate)
				if err := qtx.UpdateMultipleListingServiceInactivationDate(ctx, database.UpdateMultipleListingServiceInactivationDateParams{
					InactivationDate: dbMls.InactivationDate,
					ID:               dbMls.ID,
				}); err != nil {
					s.logger.Errorf("error (update mls inactivation_date: %v)", err)
				}
			}

			if err = qtx.CreateAgentMultipleListingService(ctx, database.CreateAgentMultipleListingServiceParams{
				AgentID:                  agentId,
				MultipleListingServiceID: int64ToNullInt64(dbMls.ID),
			}); err != nil {
				s.logger.Errorf("error creating agent mls: %v", err)
			}
		}

		s.logger.Debugf("- languages:")
		for _, lang := range agent.Languages {
			s.logger.Debugf("	* %s", lang)
			dbLangID, err := qtx.GetLanguageID(ctx, stringToNullString(lang))
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbLangID, err = qtx.CreateLanguage(ctx, stringToNullString(lang))
				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentLanguage(ctx, database.CreateAgentLanguageParams{
				LanguageID: int64ToNullInt64(dbLangID),
				AgentID:    agentId,
			}); err != nil {
				s.logger.Errorf("error creating agent language: %v", err)
			}
		}
		s.logger.Debugf("- user languages:")
		for _, lang := range agent.UserLanguages {
			s.logger.Debugf("	* %s", lang)
			dbLangID, err := qtx.GetLanguageID(ctx, stringToNullString(lang))
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbLangID, err = qtx.CreateLanguage(ctx, stringToNullString(lang))
				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentUserLanguage(ctx, database.CreateAgentUserLanguageParams{
				LanguageID: int64ToNullInt64(dbLangID),
				AgentID:    agentId,
			}); err != nil {
				s.logger.Errorf("error creating agent language: %v", err)
			}
		}

		s.logger.Debugf("- zips:")
		for _, zip := range agent.Zips {
			s.logger.Debugf("	* %s", zip)
			dbZipID, err := qtx.GetZipID(ctx, stringToNullString(zip))
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbZipID, err = qtx.CreateZip(ctx, stringToNullString(zip))
				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentZip(ctx, database.CreateAgentZipParams{
				ZipID:   int64ToNullInt64(dbZipID),
				AgentID: agentId,
			}); err != nil {
				s.logger.Errorf("error creating agent zip: %v", err)
			}
		}

		s.logger.Debugf("- areas:")
		for _, area := range agent.ServedAreas {
			s.logger.Debugf("	* (%s, %s)", area.Name, area.StateCode)
			dbAreaID, err := qtx.GetAreaID(ctx, database.GetAreaIDParams{
				Name:      stringToNullString(area.Name),
				StateCode: stringToNullString(area.StateCode),
			})

			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbAreaID, err = qtx.CreateArea(ctx, database.CreateAreaParams{
					Name:      stringToNullString(area.Name),
					StateCode: stringToNullString(area.StateCode),
				})
				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentServedArea(ctx, database.CreateAgentServedAreaParams{
				AreaID:  int64ToNullInt64(dbAreaID),
				AgentID: agentId,
			}); err != nil {
				s.logger.Errorf("error creating agent served area: %v", err)
			}
		}
		s.logger.Debugf("- marketing areas:")
		for _, area := range agent.MarketingAreaCities {
			s.logger.Debugf("	* (%s, %s)", area.Name, area.StateCode)
			dbAreaID, err := qtx.GetAreaID(ctx, database.GetAreaIDParams{
				Name:      stringToNullString(area.Name),
				StateCode: stringToNullString(area.StateCode),
			})

			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbAreaID, err = qtx.CreateArea(ctx, database.CreateAreaParams{
					Name:      stringToNullString(area.Name),
					StateCode: stringToNullString(area.StateCode),
				})
				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentMarketingArea(ctx, database.CreateAgentMarketingAreaParams{
				AreaID:  int64ToNullInt64(dbAreaID),
				AgentID: agentId,
			}); err != nil {
				s.logger.Errorf("error creating agent marketing area: %v", err)
			}
		}

		s.logger.Debugf("- designations:")
		for _, designation := range agent.Designations {
			s.logger.Debugf("	* %s", designation.Name)
			dbDesignationID, err := qtx.GetDesignationID(ctx, stringToNullString(designation.Name))
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbDesignationID, err = qtx.CreateDesignation(ctx, stringToNullString(designation.Name))
				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentDesignation(ctx, database.CreateAgentDesignationParams{
				DesignationID: int64ToNullInt64(dbDesignationID),
				AgentID:       agentId,
			}); err != nil {
				s.logger.Errorf("error creating agent designation: %v", err)
			}
		}
		s.logger.Debugf("- specializations:")
		for _, specialization := range agent.Specializations {
			s.logger.Debugf("	* %s", specialization.Name)
			dbSpecializationID, err := qtx.GetSpecializationID(ctx, stringToNullString(specialization.Name))
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbSpecializationID, err = qtx.CreateSpecialization(ctx, stringToNullString(specialization.Name))
				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentSpecialization(ctx, database.CreateAgentSpecializationParams{
				SpecializationID: int64ToNullInt64(dbSpecializationID),
				AgentID:          agentId,
			}); err != nil {
				s.logger.Errorf("error creating agent specialization: %v", err)
			}
		}

		s.logger.Debugf("- address:")
		if !agent.Address.IsZero() {
			s.logger.Debugf("	* %+v", agent.Address)
			dbAddressID, err := qtx.GetAddressID(ctx, database.GetAddressIDParams{
				Line:       stringToNullString(agent.Address.Line),
				Line2:      stringToNullString(agent.Address.Line2),
				City:       stringToNullString(agent.Address.City),
				StateCode:  stringToNullString(agent.Address.StateCode),
				PostalCode: stringToNullString(agent.Address.PostalCode),
			})
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbAddressID, err = qtx.CreateAddress(ctx, database.CreateAddressParams{
					Line:       stringToNullString(agent.Address.Line),
					Line2:      stringToNullString(agent.Address.Line2),
					City:       stringToNullString(agent.Address.City),
					StateCode:  stringToNullString(agent.Address.StateCode),
					State:      stringToNullString(agent.Address.State),
					PostalCode: stringToNullString(agent.Address.PostalCode),
					Country:    stringToNullString(agent.Address.Country),
				})

				if err != nil {
					return err
				}
			}

			if err := qtx.UpdateAgentAddressID(ctx, database.UpdateAgentAddressIDParams{
				AddressID: int64ToNullInt64(dbAddressID),
				ID:        agent.ID,
			}); err != nil {
				s.logger.Errorf("error updating agent address_id: %v", err)
			}
		}

		s.logger.Debugf("- phones:")
		for _, phone := range agent.Phones {
			if phone.IsZero() {
				continue
			}
			s.logger.Debugf("	* %s", phone.Number)

			dbPhoneID, err := qtx.GetPhoneID(ctx, database.GetPhoneIDParams{
				Ext:    stringToNullString(phone.Ext),
				Number: stringToNullString(phone.Number),
				Type:   stringToNullString(phone.Type),
			})

			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbPhoneID, err = qtx.CreatePhone(ctx, database.CreatePhoneParams{
					Ext:     stringToNullString(phone.Ext),
					Number:  stringToNullString(phone.Number),
					Type:    stringToNullString(phone.Type),
					IsValid: boolToNullBool(phone.IsValid),
				})

				if err != nil {
					return err
				}
			}

			if err := qtx.CreateAgentPhone(ctx, database.CreateAgentPhoneParams{
				PhoneID: int64ToNullInt64(dbPhoneID),
				AgentID: agentId,
			}); err != nil {
				s.logger.Errorf("error creating agent phone: %v", err)
			}
		}

		s.logger.Debugf("- broker:")
		if !agent.Broker.IsZero() {
			s.logger.Debugf("	* %s", agent.Broker.Name)
			dbBrokerID, err := qtx.GetBrokerID(ctx, intToNullInt64(agent.Broker.FulfillmentID))
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbBrokerID, err = qtx.CreateBroker(ctx, database.CreateBrokerParams{
					FulfillmentID: intToNullInt64(agent.Broker.FulfillmentID),
					Name:          stringToNullString(agent.Broker.Name),
					Photo:         stringToNullString(agent.Broker.Photo.Href),
					Video:         stringToNullString(agent.Broker.Video),
				})

				if err != nil {
					return err
				}
			}
			if err := qtx.UpdateAgentBrokerID(ctx, database.UpdateAgentBrokerIDParams{
				BrokerID: int64ToNullInt64(dbBrokerID),
				ID:       agent.ID,
			}); err != nil {
				s.logger.Errorf("error updating agent broker_id: %v", err)
			}
		}

		if !agent.Office.IsZero() {
			var dbOfficeAddressID int64
			s.logger.Debugf("- office address:")
			if !agent.Office.Address.IsZero() {
				s.logger.Debugf("	* %+v", agent.Office.Address)
				dbOfficeAddressID, err = qtx.GetAddressID(ctx, database.GetAddressIDParams{
					Line:       stringToNullString(agent.Address.Line),
					Line2:      stringToNullString(agent.Address.Line2),
					City:       stringToNullString(agent.Address.City),
					StateCode:  stringToNullString(agent.Address.StateCode),
					PostalCode: stringToNullString(agent.Address.PostalCode),
				})
				if err != nil {
					if err != sql.ErrNoRows {
						return err
					}

					dbOfficeAddressID, err = qtx.CreateAddress(ctx, database.CreateAddressParams{
						Line:       stringToNullString(agent.Address.Line),
						Line2:      stringToNullString(agent.Address.Line2),
						City:       stringToNullString(agent.Address.City),
						StateCode:  stringToNullString(agent.Address.StateCode),
						State:      stringToNullString(agent.Address.State),
						PostalCode: stringToNullString(agent.Address.PostalCode),
						Country:    stringToNullString(agent.Address.Country),
					})

					if err != nil {
						return err
					}
				}
			}

			s.logger.Debugf("- office:")
			s.logger.Debugf("	* %v", agent.Office.Name)
			dbOfficeID, err := qtx.GetOfficeID(ctx, intToNullInt64(agent.Office.FulfillmentID))
			if err != nil {
				if err != sql.ErrNoRows {
					return err
				}

				dbOfficeID, err = qtx.CreateOffice(ctx, database.CreateOfficeParams{
					Name:          stringToNullString(agent.Office.Name),
					Email:         stringToNullString(agent.Office.Email),
					Photo:         stringToNullString(agent.Office.Photo.Href),
					Website:       stringToNullString(agent.Office.Website),
					Slogan:        stringToNullString(agent.Office.Slogan),
					Video:         stringToNullString(agent.Office.Video),
					FulfillmentID: intToNullInt64(agent.Office.FulfillmentID),
					AddressID:     int64ToNullInt64(dbOfficeAddressID),
				})

				if err != nil {
					return err
				}
			}

			if err := qtx.UpdateAgentOfficeID(ctx, database.UpdateAgentOfficeIDParams{
				OfficeID: int64ToNullInt64(dbOfficeID),
				ID:       agent.ID,
			}); err != nil {
				s.logger.Errorf("error updating agent office_id: %v", err)
			}

			officePhones := make([]Phone, 0, len(agent.Office.Phones)+len(agent.Office.PhoneList))
			officePhones = append(officePhones, agent.Phones...)
			for _, officePh := range agent.Office.PhoneList {
				officePhones = append(officePhones, officePh)
			}

			s.logger.Debugf("- office phones:")
			for _, phone := range officePhones {
				if phone.IsZero() {
					continue
				}
				s.logger.Debugf("	* %+v", phone)

				dbPhoneID, err := qtx.GetPhoneID(ctx, database.GetPhoneIDParams{
					Ext:    stringToNullString(phone.Ext),
					Number: stringToNullString(phone.Number),
					Type:   stringToNullString(phone.Type),
				})

				if err != nil {
					if err != sql.ErrNoRows {
						return err
					}

					dbPhoneID, err = qtx.CreatePhone(ctx, database.CreatePhoneParams{
						Ext:     stringToNullString(phone.Ext),
						Number:  stringToNullString(phone.Number),
						Type:    stringToNullString(phone.Type),
						IsValid: boolToNullBool(phone.IsValid),
					})

					if err != nil {
						return err
					}
				}

				if err := qtx.CreateOfficePhone(ctx, database.CreateOfficePhoneParams{
					PhoneID:  int64ToNullInt64(dbPhoneID),
					OfficeID: int64ToNullInt64(dbOfficeID),
				}); err != nil {
					s.logger.Errorf("error creating office phone: %v", err)
				}
			}

			officeFeedLicenses := make([]FeedLicense, 0, len(agent.Office.FeedLicenses)+len(agent.Office.Licenses))
			officeFeedLicenses = append(officeFeedLicenses, agent.Office.FeedLicenses...)
			officeFeedLicenses = append(officeFeedLicenses, agent.Office.Licenses...)

			s.logger.Debugf("- office feed licences:")
			for _, feedLicense := range officeFeedLicenses {
				if feedLicense.IsZero() {
					continue
				}
				s.logger.Debugf("	* (%s, %s)", feedLicense.StateCode, feedLicense.Country)
				dbFeedLicenseID, err := qtx.CreateFeedLicense(ctx, database.CreateFeedLicenseParams{
					Country:       stringToNullString(feedLicense.Country),
					LicenseNumber: stringToNullString(feedLicense.LicenseNumber),
					StateCode:     stringToNullString(feedLicense.StateCode),
				})

				if err != nil {
					return err
				}

				if err := qtx.CreateOfficeFeedLicense(ctx, database.CreateOfficeFeedLicenseParams{
					FeedLicenseID: int64ToNullInt64(dbFeedLicenseID),
					OfficeID:      int64ToNullInt64(dbOfficeID),
				}); err != nil {
					s.logger.Errorf("error creating office feed license: %v", err)
				}
			}
		}
		return nil
	})
}
