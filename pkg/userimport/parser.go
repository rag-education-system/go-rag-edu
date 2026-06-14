package userimport

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

const MaxRows = 500

type Row struct {
	RowNum   int
	Name     string
	Email    string
	Major    string
	Role     string
	Password string
}

var headerAliases = map[string]string{
	"nama":       "name",
	"name":       "name",
	"email":      "email",
	"jurusan":    "major",
	"major":      "major",
	"prodi":      "major",
	"program":    "major",
	"peran":      "role",
	"role":       "role",
	"password":   "password",
	"kata_sandi": "password",
	"sandi":      "password",
}

func ParseFile(filename string, data []byte) ([]Row, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return parseCSV(data)
	case ".xlsx", ".xlsm", ".xltx", ".xltm":
		return parseXLSX(data)
	default:
		return nil, fmt.Errorf("format file tidak didukung, gunakan CSV atau XLSX")
	}
}

func parseCSV(data []byte) ([]Row, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file CSV: %w", err)
	}

	return rowsFromRecords(records)
}

func parseXLSX(data []byte) ([]Row, error) {
	file, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("gagal membaca file Excel: %w", err)
	}
	defer file.Close()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("file Excel tidak memiliki sheet")
	}

	records, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("gagal membaca sheet Excel: %w", err)
	}

	return rowsFromRecords(records)
}

func rowsFromRecords(records [][]string) ([]Row, error) {
	if len(records) < 2 {
		return nil, errors.New("file harus memiliki header dan minimal 1 baris data")
	}

	headerMap, err := mapHeaders(records[0])
	if err != nil {
		return nil, err
	}

	var rows []Row
	for i := 1; i < len(records); i++ {
		record := records[i]
		if isEmptyRecord(record) {
			continue
		}

		row := Row{RowNum: i + 1}
		for idx, value := range record {
			field, ok := headerMap[idx]
			if !ok {
				continue
			}
			value = strings.TrimSpace(value)
			switch field {
			case "name":
				row.Name = value
			case "email":
				row.Email = strings.ToLower(value)
			case "major":
				row.Major = value
			case "role":
				row.Role = strings.ToUpper(value)
			case "password":
				row.Password = value
			}
		}

		if row.Name == "" && row.Email == "" && row.Major == "" && row.Role == "" && row.Password == "" {
			continue
		}

		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, errors.New("tidak ada data pengguna yang valid ditemukan")
	}
	if len(rows) > MaxRows {
		return nil, fmt.Errorf("maksimal %d baris per file", MaxRows)
	}

	return rows, nil
}

func mapHeaders(headers []string) (map[int]string, error) {
	mapped := make(map[int]string)
	for idx, header := range headers {
		key := normalizeHeader(header)
		if key == "" {
			continue
		}
		field, ok := headerAliases[key]
		if !ok {
			continue
		}
		mapped[idx] = field
	}

	required := []string{"name", "email", "major", "role"}
	found := make(map[string]bool)
	for _, field := range mapped {
		found[field] = true
	}

	var missing []string
	for _, req := range required {
		if !found[req] {
			missing = append(missing, req)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("kolom wajib tidak ditemukan: %s", strings.Join(missing, ", "))
	}

	return mapped, nil
}

func normalizeHeader(header string) string {
	header = strings.TrimSpace(strings.ToLower(header))
	header = strings.ReplaceAll(header, " ", "_")
	return header
}

func isEmptyRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func TemplateCSV() string {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"nama", "email", "jurusan", "peran", "password"})
	_ = writer.Write([]string{"Budi Santoso", "budi@univ.ac.id", "PTIK", "STUDENT", "budi123"})
	_ = writer.Write([]string{"Siti Dosen", "siti@univ.ac.id", "PTIK", "TEACHER", ""})
	writer.Flush()
	return buf.String()
}

func ReadLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	limited := io.LimitReader(r, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("ukuran file maksimal %d MB", maxBytes/(1024*1024))
	}
	return data, nil
}
