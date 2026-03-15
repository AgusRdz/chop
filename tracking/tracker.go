package tracking

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Stats holds aggregate token savings statistics.
type Stats struct {
	TotalCommands     int
	TotalRawTokens    int
	TotalSavedTokens  int
	OverallSavingsPct float64
	TodayCommands     int
	TodayRawTokens    int
	TodaySavedTokens  int
	WeekCommands      int
	WeekRawTokens     int
	WeekSavedTokens   int
	MonthCommands     int
	MonthRawTokens    int
	MonthSavedTokens  int
	YearCommands      int
	YearRawTokens     int
	YearSavedTokens   int
}

// Record holds a single tracking entry.
type Record struct {
	Timestamp      string
	Command        string
	RawTokens      int
	FilteredTokens int
	SavingsPct     float64
}

var (
	db     *sql.DB
	dbOnce sync.Once
	dbErr  error
)

func dbPath() string {
	if p := os.Getenv("CHOP_DB_PATH"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "chop", "tracking.db")
}

// Init opens (or creates) the tracking database and ensures the schema exists.
func Init() error {
	dbOnce.Do(func() {
		path := dbPath()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			dbErr = err
			return
		}
		db, dbErr = sql.Open("sqlite", path)
		if dbErr != nil {
			return
		}
		db.SetMaxOpenConns(1)
		_, dbErr = db.Exec("PRAGMA journal_mode=WAL")
		if dbErr != nil {
			return
		}
		_, dbErr = db.Exec("PRAGMA busy_timeout=5000")
		if dbErr != nil {
			return
		}
		_, dbErr = db.Exec(`CREATE TABLE IF NOT EXISTS tracking (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			command TEXT NOT NULL,
			raw_tokens INTEGER NOT NULL,
			filtered_tokens INTEGER NOT NULL,
			savings_pct REAL NOT NULL
		)`)
		if dbErr != nil {
			return
		}
		_, dbErr = db.Exec(`CREATE TABLE IF NOT EXISTS unchopped_skip (
			command TEXT PRIMARY KEY,
			added_at TEXT NOT NULL
		)`)
		if dbErr != nil {
			return
		}
		_, dbErr = db.Exec(`CREATE TABLE IF NOT EXISTS tracking_skip (
			command TEXT PRIMARY KEY,
			added_at TEXT NOT NULL
		)`)
	})
	return dbErr
}

// initForTest resets the singleton so tests can re-init with a new DB path.
func initForTest() {
	dbOnce = sync.Once{}
	db = nil
	dbErr = nil
}

// Track records a command's token savings. Silent on error.
func Track(command string, rawTokens, filteredTokens int) error {
	if err := Init(); err != nil {
		return err
	}
	var skipped int
	row := db.QueryRow(`SELECT COUNT(*) FROM tracking_skip WHERE command = ? OR ? LIKE command || ' %'`, command, command)
	if err := row.Scan(&skipped); err == nil && skipped > 0 {
		return nil
	}
	var savingsPct float64
	if rawTokens > 0 {
		savingsPct = 100.0 - (float64(filteredTokens) / float64(rawTokens) * 100.0)
	}
	now := time.Now().Local().Format("2006-01-02 15:04:05")
	_, err := db.Exec(
		`INSERT INTO tracking (timestamp, command, raw_tokens, filtered_tokens, savings_pct)
		 VALUES (?, ?, ?, ?, ?)`,
		now, command, rawTokens, filteredTokens, savingsPct,
	)
	return err
}

// GetStats returns aggregate statistics.
func GetStats() (Stats, error) {
	if err := Init(); err != nil {
		return Stats{}, err
	}
	var s Stats

	row := db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking`)
	if err := row.Scan(&s.TotalCommands, &s.TotalRawTokens, &s.TotalSavedTokens); err != nil {
		return Stats{}, err
	}
	if s.TotalRawTokens > 0 {
		s.OverallSavingsPct = float64(s.TotalSavedTokens) / float64(s.TotalRawTokens) * 100.0
	}

	today := time.Now().Local().Format("2006-01-02")
	row = db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking WHERE timestamp LIKE ?`,
		today+"%",
	)
	if err := row.Scan(&s.TodayCommands, &s.TodayRawTokens, &s.TodaySavedTokens); err != nil {
		return Stats{}, err
	}

	// Calendar week: Monday 00:00 through now
	now := time.Now().Local()
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-int(weekday-time.Monday), 0, 0, 0, 0, now.Location()).Format("2006-01-02 00:00:00")
	row = db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking WHERE timestamp >= ?`,
		weekStart,
	)
	if err := row.Scan(&s.WeekCommands, &s.WeekRawTokens, &s.WeekSavedTokens); err != nil {
		return Stats{}, err
	}

	// Calendar month: 1st of current month through now
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02 00:00:00")
	row = db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking WHERE timestamp >= ?`,
		monthStart,
	)
	if err := row.Scan(&s.MonthCommands, &s.MonthRawTokens, &s.MonthSavedTokens); err != nil {
		return Stats{}, err
	}

	// Calendar year: Jan 1 of current year through now
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02 00:00:00")
	row = db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking WHERE timestamp >= ?`,
		yearStart,
	)
	if err := row.Scan(&s.YearCommands, &s.YearRawTokens, &s.YearSavedTokens); err != nil {
		return Stats{}, err
	}

	return s, nil
}

// GetHistory returns the last N tracking records in reverse chronological order.
func GetHistory(limit int) ([]Record, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT timestamp, command, raw_tokens, filtered_tokens, savings_pct
		 FROM tracking ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Timestamp, &r.Command, &r.RawTokens, &r.FilteredTokens, &r.SavingsPct); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// CommandSummary holds per-command aggregate stats.
type CommandSummary struct {
	BaseCommand string
	Count       int
	RawTokens   int
	SavedTokens int
	SavingsPct  float64
	ZeroCount   int // times with 0% savings
}

// GetCommandSummary returns per-base-command aggregates, sorted by tokens saved descending.
func GetCommandSummary() ([]CommandSummary, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	rows, err := db.Query(`
		SELECT
			CASE
				WHEN INSTR(command, ' ') > 0 THEN SUBSTR(command, 1, INSTR(command, ' ') - 1)
				ELSE command
			END AS base_cmd,
			COUNT(*) AS cnt,
			COALESCE(SUM(raw_tokens), 0) AS raw,
			COALESCE(SUM(raw_tokens - filtered_tokens), 0) AS saved,
			SUM(CASE WHEN savings_pct = 0 AND raw_tokens > 0 THEN 1 ELSE 0 END) AS zero_cnt
		FROM tracking
		GROUP BY base_cmd
		ORDER BY saved DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []CommandSummary
	for rows.Next() {
		var s CommandSummary
		if err := rows.Scan(&s.BaseCommand, &s.Count, &s.RawTokens, &s.SavedTokens, &s.ZeroCount); err != nil {
			return nil, err
		}
		if s.RawTokens > 0 {
			s.SavingsPct = float64(s.SavedTokens) / float64(s.RawTokens) * 100.0
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

// UnchoppedSummary holds stats for commands that consistently get 0% savings.
type UnchoppedSummary struct {
	Command     string
	Count       int
	TotalTokens int // total raw tokens that could have been saved
}

// GetUnchopped returns commands that always got 0% savings, sorted by call count desc.
// These are the best candidates for writing new filters.
func GetUnchopped() ([]UnchoppedSummary, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	// Group by first two words (command + subcommand) to distinguish e.g. "git clone" vs "git status".
	// Only include commands where raw_tokens > 0 and savings_pct = 0.
	// Exclude commands that have ANY record with savings > 0 (those already have working filters).
	rows, err := db.Query(`
		WITH cmd_key AS (
			SELECT
				CASE
					WHEN INSTR(command, ' ') > 0 AND INSTR(SUBSTR(command, INSTR(command, ' ') + 1), ' ') > 0
					THEN SUBSTR(command, 1, INSTR(command, ' ') + INSTR(SUBSTR(command, INSTR(command, ' ') + 1), ' ') - 1)
					ELSE command
				END AS cmd,
				raw_tokens,
				savings_pct
			FROM tracking
			WHERE raw_tokens > 0
		)
		SELECT
			cmd,
			COUNT(*) AS cnt,
			COALESCE(SUM(raw_tokens), 0) AS total_raw
		FROM cmd_key
		WHERE cmd NOT IN (
			SELECT DISTINCT cmd FROM cmd_key WHERE savings_pct > 0
		)
		AND cmd NOT IN (
			SELECT command FROM unchopped_skip
		)
		GROUP BY cmd
		ORDER BY total_raw DESC, cnt DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []UnchoppedSummary
	for rows.Next() {
		var s UnchoppedSummary
		if err := rows.Scan(&s.Command, &s.Count, &s.TotalTokens); err != nil {
			return nil, err
		}
		results = append(results, s)
	}
	return results, rows.Err()
}

// SkipUnchopped marks a command as intentionally not needing a filter.
func SkipUnchopped(cmd string) error {
	if err := Init(); err != nil {
		return err
	}
	now := time.Now().Local().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT OR REPLACE INTO unchopped_skip (command, added_at) VALUES (?, ?)`, cmd, now)
	return err
}

// DeleteCommand removes all tracking records for a command key (first two words).
// This permanently erases the command from the history and unchopped report.
func DeleteCommand(cmd string) error {
	if err := Init(); err != nil {
		return err
	}
	// The tracking table stores full command strings; match on the key prefix.
	// Key is first two words, so match "cmd" or "cmd ..." (single-word keys too).
	_, err := db.Exec(`DELETE FROM tracking WHERE command = ? OR command LIKE ?`, cmd, cmd+" %")
	if err != nil {
		return err
	}
	// Also remove from skip list if present.
	_, err = db.Exec(`DELETE FROM unchopped_skip WHERE command = ?`, cmd)
	return err
}

// UnskipUnchopped removes a command from the skip list.
func UnskipUnchopped(cmd string) error {
	if err := Init(); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM unchopped_skip WHERE command = ?`, cmd)
	return err
}

// AddTrackingSkip adds a command to the no-track list so it is never recorded again.
func AddTrackingSkip(cmd string) error {
	if err := Init(); err != nil {
		return err
	}
	now := time.Now().Local().Format("2006-01-02 15:04:05")
	_, err := db.Exec(`INSERT OR REPLACE INTO tracking_skip (command, added_at) VALUES (?, ?)`, cmd, now)
	return err
}

// RemoveTrackingSkip removes a command from the no-track list, re-enabling tracking.
func RemoveTrackingSkip(cmd string) error {
	if err := Init(); err != nil {
		return err
	}
	_, err := db.Exec(`DELETE FROM tracking_skip WHERE command = ?`, cmd)
	return err
}

// GetSkippedCommands returns all commands in the skip list, ordered alphabetically.
func GetSkippedCommands() ([]string, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT command FROM unchopped_skip ORDER BY command`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cmds []string
	for rows.Next() {
		var cmd string
		if err := rows.Scan(&cmd); err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, rows.Err()
}

// FormatUnchopped formats the unchopped commands report.
// summaries are active candidates; skipped are manually skipped commands;
// filtered are commands auto-excluded because a registered filter exists for them.
// verbose disables command name truncation.
func FormatUnchopped(summaries []UnchoppedSummary, skipped []string, filtered []UnchoppedSummary, verbose bool) string {
	if len(summaries) == 0 && len(skipped) == 0 && len(filtered) == 0 {
		return "all commands are being chopped! 🎉\n"
	}
	var b strings.Builder
	if len(summaries) > 0 {
		b.WriteString("no filter registered - output passes through raw (write a filter to compress):\n\n")
		writeUnchoppedTable(&b, summaries, verbose)
		b.WriteString(fmt.Sprintf("\n  %d command(s) - focus on high AVG first\n", len(summaries)))
	} else {
		b.WriteString("no unfiltered candidates (all commands compress or are skipped)\n")
	}
	if len(filtered) > 0 {
		b.WriteString("\nfilter registered - 0% runs happen when output is already minimal (no action needed):\n\n")
		writeUnchoppedTable(&b, filtered, verbose)
		b.WriteString("\n")
	}
	if len(skipped) > 0 {
		b.WriteString("\nskipped (no filter needed):\n")
		b.WriteString(fmt.Sprintf("  %s\n", strings.Join(skipped, ", ")))
	}
	return b.String()
}

func writeUnchoppedTable(b *strings.Builder, rows []UnchoppedSummary, verbose bool) {
	const cmdWidth = 25
	b.WriteString(fmt.Sprintf("  %-25s %5s %10s %6s\n", "COMMAND", "CALLS", "TOKENS", "AVG"))
	b.WriteString(fmt.Sprintf("  %-25s %5s %10s %6s\n", strings.Repeat("─", cmdWidth), strings.Repeat("─", 5), strings.Repeat("─", 10), strings.Repeat("─", 6)))
	for _, s := range rows {
		cmd := s.Command
		if !verbose && len(cmd) > cmdWidth {
			cmd = cmd[:cmdWidth-3] + "..."
		}
		avg := 0
		if s.Count > 0 {
			avg = s.TotalTokens / s.Count
		}
		b.WriteString(fmt.Sprintf("  %-25s %5d %10s %6d\n", cmd, s.Count, formatNum(s.TotalTokens), avg))
	}
}

// Cleanup removes records older than the given number of days.
func Cleanup(days int) error {
	if err := Init(); err != nil {
		return err
	}
	cutoff := time.Now().Local().AddDate(0, 0, -days).Format("2006-01-02 15:04:05")
	_, err := db.Exec(`DELETE FROM tracking WHERE timestamp < ?`, cutoff)
	return err
}

// CountTokens returns the word count of a string (whitespace-split).
func CountTokens(s string) int {
	return len(strings.Fields(s))
}

// FormatGain prints the gain summary report.
func FormatGain(s Stats) string {
	pct := func(saved, raw int) float64 {
		if raw == 0 {
			return 0
		}
		return float64(saved) / float64(raw) * 100.0
	}
	return fmt.Sprintf(`chop - token savings report

  today: %d commands   %s tokens saved   (%.1f%% compression)
  week:  %d commands   %s tokens saved   (%.1f%% compression)
  month: %d commands   %s tokens saved   (%.1f%% compression)
  year:  %d commands   %s tokens saved   (%.1f%% compression)
  total: %d commands   %s tokens saved   (%.1f%% compression)

run 'chop gain --history' for command history`,
		s.TodayCommands, formatNum(s.TodaySavedTokens), pct(s.TodaySavedTokens, s.TodayRawTokens),
		s.WeekCommands, formatNum(s.WeekSavedTokens), pct(s.WeekSavedTokens, s.WeekRawTokens),
		s.MonthCommands, formatNum(s.MonthSavedTokens), pct(s.MonthSavedTokens, s.MonthRawTokens),
		s.YearCommands, formatNum(s.YearSavedTokens), pct(s.YearSavedTokens, s.YearRawTokens),
		s.TotalCommands, formatNum(s.TotalSavedTokens), s.OverallSavingsPct,
	)
}

// FormatHistory formats history records for display.
// When verbose is false, long command strings are truncated to 50 characters.
func FormatHistory(records []Record, verbose bool) string {
	if len(records) == 0 {
		return "no commands tracked yet"
	}
	const maxCmd = 50
	var b strings.Builder
	b.WriteString("recent commands:\n")
	for _, r := range records {
		marker := " "
		if r.SavingsPct == 0 && r.RawTokens > 0 {
			marker = "!"
		}
		cmd := r.Command
		if !verbose && len(cmd) > maxCmd {
			cmd = cmd[:maxCmd-3] + "..."
		}
		b.WriteString(fmt.Sprintf(" %s %s  %-50s %5.1f%%  (%d -> %d tokens)\n",
			marker, r.Timestamp, cmd, r.SavingsPct, r.RawTokens, r.FilteredTokens))
	}
	b.WriteString("\n ! = 0% savings (filter may need improvement)\n")
	return b.String()
}

// FormatSummary formats per-command aggregates.
func FormatSummary(summaries []CommandSummary) string {
	if len(summaries) == 0 {
		return "no commands tracked yet"
	}
	var b strings.Builder
	b.WriteString("per-command savings:\n")
	b.WriteString(fmt.Sprintf("  %-12s %5s %8s %7s %s\n", "COMMAND", "CALLS", "SAVED", "AVG", ""))
	for _, s := range summaries {
		warn := ""
		if s.ZeroCount > 0 {
			warn = fmt.Sprintf("(%d calls at 0%%)", s.ZeroCount)
		}
		b.WriteString(fmt.Sprintf("  %-12s %5d %8s %6.0f%%  %s\n",
			s.BaseCommand, s.Count, formatNum(s.SavedTokens), s.SavingsPct, warn))
	}
	return b.String()
}

func formatNum(n int) string {
	if n < 0 {
		return "-" + formatNum(-n)
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return formatNum(n/1000) + fmt.Sprintf(",%03d", n%1000)
}

// GetStatsSince returns aggregate stats for records within the last d duration.
func GetStatsSince(d time.Duration) (Stats, error) {
	if err := Init(); err != nil {
		return Stats{}, err
	}
	since := time.Now().Local().Add(-d).Format("2006-01-02 15:04:05")
	var s Stats

	row := db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0)
         FROM tracking WHERE timestamp >= ?`, since)
	if err := row.Scan(&s.TotalCommands, &s.TotalRawTokens, &s.TotalSavedTokens); err != nil {
		return Stats{}, err
	}
	if s.TotalRawTokens > 0 {
		s.OverallSavingsPct = float64(s.TotalSavedTokens) / float64(s.TotalRawTokens) * 100.0
	}
	return s, nil
}

// GetHistorySince returns up to limit records newer than the given duration.
func GetHistorySince(limit int, d time.Duration) ([]Record, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	since := time.Now().Local().Add(-d).Format("2006-01-02 15:04:05")
	rows, err := db.Query(
		`SELECT timestamp, command, raw_tokens, filtered_tokens, savings_pct
         FROM tracking WHERE timestamp >= ? ORDER BY id DESC LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Timestamp, &r.Command, &r.RawTokens, &r.FilteredTokens, &r.SavingsPct); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// FormatGainSince formats a stats report for a --since time window.
func FormatGainSince(s Stats, sinceStr string) string {
	return fmt.Sprintf(`chop - token savings report (last %s)

  commands: %d
  saved:    %s tokens
  avg:      %.1f%%

run 'chop gain --since %s --history' for command history`,
		sinceStr,
		s.TotalCommands, formatNum(s.TotalSavedTokens), s.OverallSavingsPct,
		sinceStr,
	)
}

// ParseSinceDuration parses duration strings like "7d", "2w", "24h", "30m".
// Supports: m (minutes), h (hours), d (days), w (weeks).
// Falls back to time.ParseDuration for standard Go duration strings.
func ParseSinceDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	unit := s[len(s)-1]
	value := s[:len(s)-1]
	var n int
	if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
		return time.ParseDuration(s)
	}
	switch unit {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return time.ParseDuration(s)
	}
}

// ExportJSON writes tracking history and summary stats as JSON to w.
func ExportJSON(w io.Writer, records []Record, s Stats) error {
	type jsonRecord struct {
		Ts         string  `json:"ts"`
		Cmd        string  `json:"cmd"`
		Raw        int     `json:"raw"`
		Compressed int     `json:"compressed"`
		Saved      int     `json:"saved"`
		SavingsPct float64 `json:"savings_pct"`
	}
	type jsonSummary struct {
		TotalCommands int     `json:"total_commands"`
		TokensSaved   int     `json:"tokens_saved"`
		AvgSavingsPct float64 `json:"avg_savings_pct"`
	}
	type jsonExport struct {
		GeneratedAt string       `json:"generated_at"`
		Summary     jsonSummary  `json:"summary"`
		History     []jsonRecord `json:"history"`
	}

	history := make([]jsonRecord, len(records))
	for i, r := range records {
		history[i] = jsonRecord{
			Ts:         r.Timestamp,
			Cmd:        r.Command,
			Raw:        r.RawTokens,
			Compressed: r.FilteredTokens,
			Saved:      r.RawTokens - r.FilteredTokens,
			SavingsPct: r.SavingsPct,
		}
	}

	export := jsonExport{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Summary: jsonSummary{
			TotalCommands: s.TotalCommands,
			TokensSaved:   s.TotalSavedTokens,
			AvgSavingsPct: s.OverallSavingsPct,
		},
		History: history,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(export)
}

// ExportCSV writes tracking history as CSV to w.
func ExportCSV(w io.Writer, records []Record) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"timestamp", "command", "raw_tokens", "compressed_tokens", "saved_tokens", "savings_pct"}); err != nil {
		return err
	}
	for _, r := range records {
		row := []string{
			r.Timestamp,
			r.Command,
			fmt.Sprintf("%d", r.RawTokens),
			fmt.Sprintf("%d", r.FilteredTokens),
			fmt.Sprintf("%d", r.RawTokens-r.FilteredTokens),
			fmt.Sprintf("%.1f", r.SavingsPct),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
