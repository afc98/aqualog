package cmd

import (
	"aqualog/db"
	"aqualog/parsers"
	"database/sql"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var loggerImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a logger file to a site",
	Run: func(cmd *cobra.Command, args []string) {
		// Read flags
		fileType, _ := cmd.Flags().GetString("type")
		filePath, _ := cmd.Flags().GetString("file")
		siteIdent, _ := cmd.Flags().GetString("site")
		siteID, err := db.ResolveSiteIdentifier(siteIdent)
		if err != nil {
			fmt.Println("Site not found:", err)
			return
		}		
		loggerIdent, _ := cmd.Flags().GetString("logger")
		loggerID, err := db.ResolveLoggerIdentifier(loggerIdent)
		if err != nil {
			fmt.Println("Logger not found:", err)
			return
		}
		if _, err := os.Stat(filePath); err != nil {
			fmt.Printf("Could not open file: %w", err)
			return
		}
		var meta parsers.Metadata
		var recs []parsers.Record
		switch fileType {
		case "aquaread":
			fmt.Printf("Importing Aquaread file: %s (site %d)\n", filePath, siteID)
			parsedData, err := parsers.ParseAquaread(filePath)
			if err != nil {
				fmt.Println("Failed to parse aquaread data")
				return
			}
			meta = parsedData.Metadata
			recs = parsedData.Records
		case "solinst":
			fmt.Printf("Importing Aquaread file: %s (site %d)\n", filePath, siteID)
			parsedData, err := parsers.ParseSolinst(filePath)
			if err != nil {
				fmt.Println("Failed to parse aquaread data")
				return
			}
			meta = parsedData.Metadata
			recs = parsedData.Records
		case "in-situ":
			fmt.Println("In-Situ logger filetype support is not yet implemented")
			return
		default:
			fmt.Println("Logger file type not recognised")
			return
		}

		// Open database
		database, err := db.GetDB()
		if err != nil {
			fmt.Println("Database error:", err)
			return
		}
		defer database.Close()

		// Check if logger already exists using its serial number if a logger ID has not been provided
		if loggerID == -1 {
			err = database.QueryRow(`
			SELECT id FROM loggers
			WHERE serial_number=? AND site_id=?
			`, meta.SerialNumber, siteID).Scan(&loggerID)
			if err != nil && err != sql.ErrNoRows {
				fmt.Println("No logger with matching metadata found in database.")
				return
			}
			if err == nil {
				fmt.Println("Found existing logger, ID:", loggerID)
			}

			// Add a new logger if it doesn't already exist
			if err == sql.ErrNoRows {
				fmt.Println("Adding new logger based on file metadata")
				res, err := database.Exec(`
					INSERT INTO loggers (site_id, name, serial_number)
					VALUES (?, ?, ?)
				`, siteID, meta.SiteIdent, meta.SerialNumber)
				if err != nil {
					fmt.Println("Unable to create a new logger entry from file metadata")
					return
				}
				loggerID64, _ := res.LastInsertId()
				loggerID = int(loggerID64)
			}
		}
		// Insert records into logger_data
		fmt.Printf("Inserting %d records for logger %d\n", len(recs), loggerID)
		skipped := 0
		for _, rec := range recs {
			// Check if record already exists
			var existingID int
			err := database.QueryRow(`
			SELECT id FROM logger_data
			WHERE logger_id=? AND timestamp=?
			`, loggerID, rec.Timestamp).Scan(&existingID)

			if err == nil {
				// Record already exists, skip it
				skipped++
				continue
			} else if err != sql.ErrNoRows {
				// Database error
				fmt.Printf("Error checking for existing record at %s: %v\n", rec.Timestamp, err)
				return
			}

			// Record doesn't exist, insert it
			_, err = database.Exec(`
				INSERT INTO logger_data (logger_id, timestamp, level_m, temp_c, sal_psu, ec_us)
				VALUES (?, ?, ?, ?, ?, ?)
			`, loggerID, rec.Timestamp.Format("2006-01-02 15:04:05Z"), rec.LevelM, rec.TempC, rec.SalPSU, rec.EC)
			if err != nil {
				fmt.Printf("Failed to insert record at %s: %v\n", rec.Timestamp, err)
				return
			}
		}
		fmt.Printf("Successfully imported %d records (%d duplicate rows skipped)\n", len(recs)-skipped, skipped)
	},
}

func init() {
	loggerImportCmd.Flags().StringP("file", "f", "", "Logger file")
	loggerImportCmd.Flags().StringP("type", "t", "", "File type")
	loggerImportCmd.Flags().StringP("site", "s", 0, "Site ID")
	loggerImportCmd.Flags().StringP("logger", "l", -1, "Site ID")

	loggerImportCmd.MarkFlagRequired(("file"))
	loggerImportCmd.MarkFlagRequired(("type"))
	loggerImportCmd.MarkFlagRequired(("site"))

	rootCmd.AddCommand(loggerImportCmd)
}
