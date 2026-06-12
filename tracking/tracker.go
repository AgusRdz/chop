package tracking

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"unicode/utf8"

	"github.com/AgusRdz/chop/config"
	"github.com/mattn/go-isatty"
	_ "modernc.org/sqlite"
)

// truncateRunes truncates s to at most n runes, appending "..." if truncated.
func truncateRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n]) + "..."
}

// ANSI color codes.
const (
	ansiReset  = "\033[0m"
	ansiDim    = "\033[2m"
	ansiBold   = "\033[1m"
	ansiYellow = "\033[33m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
)

// IsColorEnabled reports whether color output should be used:
// stdout must be a terminal and NO_COLOR must not be set.
func IsColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

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
	Project        string
}

// ProjectSummary holds per-project aggregate stats.
type ProjectSummary struct {
	Project     string
	Count       int
	RawTokens   int
	SavedTokens int
	SavingsPct  float64
}

var (
	db     *sql.DB
	dbOnce sync.Once
	dbErr  error

	gitRootOnce sync.Once
	gitRootVal  string
)

func dbPath() string {
	if p := os.Getenv("CHOP_DB_PATH"); p != "" {
		return p
	}
	return filepath.Join(config.DataDir(), "tracking.db")
}

// DBPath returns the path to the tracking database file.
func DBPath() string {
	return dbPath()
}

// Init opens (or creates) the tracking database and ensures the schema exists.
func Init() error {
	dbOnce.Do(func() {
		MigrateWindowsDataDir()
		path := dbPath()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			dbErr = err
			return
		}
		// Pre-create the DB file with secure permissions before SQLite opens it,
		// eliminating the TOCTOU window between file creation and chmod.
		if f, err := os.OpenFile(path, os.O_CREATE, 0o600); err == nil {
			f.Close()
		}
		db, dbErr = sql.Open("sqlite", path)
		if dbErr != nil {
			return
		}
		if err := os.Chmod(path, 0o600); err != nil {
			dbErr = err
			return
		}
		db.SetMaxOpenConns(4)
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
		// Migrate: add project column if not present (added in v1.x)
		var colExists int
		_ = db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('tracking') WHERE name='project'`).Scan(&colExists)
		if colExists == 0 {
			_, dbErr = db.Exec(`ALTER TABLE tracking ADD COLUMN project TEXT`)
			if dbErr != nil {
				return
			}
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
		if dbErr != nil {
			return
		}
		_, dbErr = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tracking_timestamp ON tracking (timestamp)`)
	})
	return dbErr
}

// initForTest resets the singleton so tests can re-init with a new DB path.
func initForTest() {
	dbOnce = sync.Once{}
	db = nil
	dbErr = nil
	gitRootOnce = sync.Once{}
	gitRootVal = ""
}

// Track records a command's token savings. Silent on error.
func Track(command string, rawTokens, filteredTokens int) error {
	if err := Init(); err != nil {
		return err
	}
	var skipped int
	row := db.QueryRow(`SELECT COUNT(*) FROM tracking_skip WHERE command = ? OR ? LIKE command || ' %' ESCAPE '\'`, command, command)
	if err := row.Scan(&skipped); err == nil && skipped > 0 {
		return nil
	}
	var savingsPct float64
	if rawTokens > 0 {
		savingsPct = 100.0 - (float64(filteredTokens) / float64(rawTokens) * 100.0)
	}
	now := time.Now().Local().Format("2006-01-02 15:04:05")
	project := gitRoot()
	_, err := db.Exec(
		`INSERT INTO tracking (timestamp, command, raw_tokens, filtered_tokens, savings_pct, project)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		now, command, rawTokens, filteredTokens, savingsPct, project,
	)
	return err
}

// GetStats returns aggregate statistics.
func GetStats() (Stats, error) {
	if err := Init(); err != nil {
		return Stats{}, err
	}

	now := time.Now().Local()
	today := now.Format("2006-01-02")
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day()-int(weekday-time.Monday), 0, 0, 0, 0, now.Location()).Format("2006-01-02 00:00:00")
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02 00:00:00")
	yearStart := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02 00:00:00")

	type bucket struct {
		cmds, raw, saved int
		err              error
	}
	results := make([]bucket, 5)

	type query struct {
		idx   int
		sql   string
		arg   any
	}
	queries := []query{
		{0, `SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking`, nil},
		{1, `SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking WHERE timestamp LIKE ? || '%' ESCAPE '\'`, escapeLike(today)},
		{2, `SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking WHERE timestamp >= ?`, weekStart},
		{3, `SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking WHERE timestamp >= ?`, monthStart},
		{4, `SELECT COUNT(*), COALESCE(SUM(raw_tokens),0), COALESCE(SUM(raw_tokens - filtered_tokens),0) FROM tracking WHERE timestamp >= ?`, yearStart},
	}

	var wg sync.WaitGroup
	for _, q := range queries {
		wg.Add(1)
		go func(q query) {
			defer wg.Done()
			var row *sql.Row
			if q.arg == nil {
				row = db.QueryRow(q.sql)
			} else {
				row = db.QueryRow(q.sql, q.arg)
			}
			results[q.idx].err = row.Scan(&results[q.idx].cmds, &results[q.idx].raw, &results[q.idx].saved)
		}(q)
	}
	wg.Wait()

	for _, r := range results {
		if r.err != nil {
			return Stats{}, r.err
		}
	}

	s := Stats{
		TotalCommands: results[0].cmds, TotalRawTokens: results[0].raw, TotalSavedTokens: results[0].saved,
		TodayCommands: results[1].cmds, TodayRawTokens: results[1].raw, TodaySavedTokens: results[1].saved,
		WeekCommands:  results[2].cmds, WeekRawTokens:  results[2].raw, WeekSavedTokens:  results[2].saved,
		MonthCommands: results[3].cmds, MonthRawTokens: results[3].raw, MonthSavedTokens: results[3].saved,
		YearCommands:  results[4].cmds, YearRawTokens:  results[4].raw, YearSavedTokens:  results[4].saved,
	}
	if s.TotalRawTokens > 0 {
		s.OverallSavingsPct = float64(s.TotalSavedTokens) / float64(s.TotalRawTokens) * 100.0
	}
	return s, nil
}

// GetHistory returns the last N tracking records in reverse chronological order.
func GetHistory(limit int) ([]Record, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT timestamp, command, raw_tokens, filtered_tokens, savings_pct, COALESCE(project, '')
		 FROM tracking ORDER BY id DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Timestamp, &r.Command, &r.RawTokens, &r.FilteredTokens, &r.SavingsPct, &r.Project); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// FilterCompressed returns only records where output was actually compressed (savings > 0).
func FilterCompressed(records []Record) []Record {
	out := records[:0:0]
	for _, r := range records {
		if r.FilteredTokens < r.RawTokens {
			out = append(out, r)
		}
	}
	return out
}

// GetHistoryByProject returns the last N tracking records for a specific project.
func GetHistoryByProject(project string, limit int) ([]Record, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT timestamp, command, raw_tokens, filtered_tokens, savings_pct, COALESCE(project, '')
		 FROM tracking WHERE project = ? ORDER BY id DESC LIMIT ?`, project, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Timestamp, &r.Command, &r.RawTokens, &r.FilteredTokens, &r.SavingsPct, &r.Project); err != nil {
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
		// Normalize full executable paths (e.g. C:/path/to/acli.exe) to just the binary name.
		s.BaseCommand = strings.TrimSuffix(filepath.Base(s.BaseCommand), ".exe")
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
	pattern := escapeLike(cmd) + " %"
	_, err := db.Exec(`DELETE FROM tracking WHERE command = ? OR command LIKE ? ESCAPE '\'`, cmd, pattern)
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
		if !verbose && utf8.RuneCountInString(cmd) > cmdWidth {
			cmd = truncateRunes(cmd, cmdWidth-3)
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
	color := IsColorEnabled()
	pct := func(saved, raw int) float64 {
		if raw == 0 {
			return 0
		}
		return float64(saved) / float64(raw) * 100.0
	}

	var b strings.Builder
	const headerText = "Chop \u2014 Token Savings"
	sep := strings.Repeat("\u2550", 46)
	divider := strings.Repeat("\u2500", 44)

	if color {
		b.WriteString(ansiBold + headerText + ansiReset + "\n")
		b.WriteString(ansiDim + sep + ansiReset + "\n")
	} else {
		b.WriteString(headerText + "\n")
		b.WriteString(sep + "\n")
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("  Commands:      %s\n", formatNum(s.TotalCommands)))
	b.WriteString(fmt.Sprintf("  Input tokens:  %s\n", formatCompact(s.TotalRawTokens)))
	b.WriteString(fmt.Sprintf("  Tokens saved:  %s\n", formatCompact(s.TotalSavedTokens)))
	bar := renderBar(s.OverallSavingsPct, 100.0, 20, color)
	b.WriteString(fmt.Sprintf("  Efficiency:    %s  %.1f%%\n", bar, s.OverallSavingsPct))
	b.WriteString("\n")

	colHeader := fmt.Sprintf("  %-12s  %8s  %10s  %6s", "Period", "Commands", "Saved", "Avg%")
	if color {
		b.WriteString(ansiDim + colHeader + ansiReset + "\n")
		b.WriteString(ansiDim + "  " + divider + ansiReset + "\n")
	} else {
		b.WriteString(colHeader + "\n")
		b.WriteString("  " + divider + "\n")
	}

	type periodRow struct {
		label    string
		commands int
		saved    int
		raw      int
	}
	periods := []periodRow{
		{"Today", s.TodayCommands, s.TodaySavedTokens, s.TodayRawTokens},
		{"This week", s.WeekCommands, s.WeekSavedTokens, s.WeekRawTokens},
		{"This month", s.MonthCommands, s.MonthSavedTokens, s.MonthRawTokens},
		{"This year", s.YearCommands, s.YearSavedTokens, s.YearRawTokens},
		{"All time", s.TotalCommands, s.TotalSavedTokens, s.TotalRawTokens},
	}
	for _, p := range periods {
		avg := pct(p.saved, p.raw)
		b.WriteString(fmt.Sprintf("  %-12s  %8s  %10s  %5.1f%%\n",
			p.label, formatNum(p.commands), formatNum(p.saved), avg))
	}

	b.WriteString("\n")
	hints := "  chop gain --summary   per-command breakdown\n" +
		"  chop gain --history   recent commands\n"
	if color {
		b.WriteString(ansiDim + hints + ansiReset)
	} else {
		b.WriteString(hints)
	}

	return b.String()
}

// FormatHistory formats history records for display.
// When verbose is false, long command strings are truncated to 50 characters.
// When color is true, ANSI colors are applied to markers and savings percentages.
// In verbose mode, records are grouped by project with a header line between groups.
func FormatHistory(records []Record, verbose bool, color bool) string {
	if len(records) == 0 {
		return "no commands tracked yet"
	}
	const maxCmd = 50

	var b strings.Builder
	const headerText = "Chop \u2014 Recent Commands"
	sep := strings.Repeat("\u2550", 46)

	if color {
		b.WriteString(ansiBold + headerText + ansiReset + "\n")
		b.WriteString(ansiDim + sep + ansiReset + "\n")
	} else {
		b.WriteString(headerText + "\n")
		b.WriteString(sep + "\n")
	}
	b.WriteString("\n")

	var lastProject string
	for _, r := range records {
		if verbose && r.Project != lastProject {
			if color {
				b.WriteString(ansiDim + "[project: " + r.Project + "]" + ansiReset + "\n")
			} else {
				b.WriteString("[project: " + r.Project + "]\n")
			}
			lastProject = r.Project
		}
		isZero := r.SavingsPct == 0 && r.RawTokens > 0
		marker := " "
		if isZero {
			marker = "!"
		}
		cmd := r.Command
		if !verbose && utf8.RuneCountInString(cmd) > maxCmd {
			cmd = truncateRunes(cmd, maxCmd-3)
		}
		if color {
			markerStr := "  " + marker + " "
			if isZero {
				markerStr = ansiYellow + "  " + marker + " " + ansiReset
			}
			ts := ansiDim + r.Timestamp + ansiReset
			var savingsStr string
			switch {
			case r.SavingsPct > 10:
				savingsStr = ansiGreen + fmt.Sprintf("%5.1f%%", r.SavingsPct) + ansiReset
			case r.SavingsPct > 0:
				savingsStr = ansiYellow + fmt.Sprintf("%5.1f%%", r.SavingsPct) + ansiReset
			default:
				savingsStr = ansiRed + fmt.Sprintf("%5.1f%%", r.SavingsPct) + ansiReset
			}
			b.WriteString(fmt.Sprintf("%s%s  %-50s %s  (%d \u2192 %d tokens)\n",
				markerStr, ts, cmd, savingsStr, r.RawTokens, r.FilteredTokens))
		} else {
			b.WriteString(fmt.Sprintf("  %s %s  %-50s %5.1f%%  (%d \u2192 %d tokens)\n",
				marker, r.Timestamp, cmd, r.SavingsPct, r.RawTokens, r.FilteredTokens))
		}
	}

	hasZero := false
	for _, r := range records {
		if r.SavingsPct == 0 && r.RawTokens > 0 {
			hasZero = true
			break
		}
	}
	if hasZero {
		legend := "\n  ! = 0% savings (filter may need improvement)\n"
		if color {
			legend = "\n" + ansiDim + "  ! = 0% savings (filter may need improvement)" + ansiReset + "\n"
		}
		b.WriteString(legend)
	}
	return b.String()
}

// FormatSummary formats per-command aggregates.
func FormatSummary(summaries []CommandSummary) string {
	if len(summaries) == 0 {
		return "no commands tracked yet"
	}

	color := IsColorEnabled()
	var b strings.Builder
	const headerText = "Chop \u2014 Token Savings by Command"
	sep := strings.Repeat("\u2550", 62)
	divider := strings.Repeat("\u2500", 60)

	if color {
		b.WriteString(ansiBold + headerText + ansiReset + "\n")
		b.WriteString(ansiDim + sep + ansiReset + "\n")
	} else {
		b.WriteString(headerText + "\n")
		b.WriteString(sep + "\n")
	}
	b.WriteString("\n")

	maxSaved := 0
	for _, s := range summaries {
		if s.SavedTokens > maxSaved {
			maxSaved = s.SavedTokens
		}
	}

	colHeader := fmt.Sprintf("  %-3s  %-20s  %5s  %8s  %5s  %s", "#", "Command", "Count", "Saved", "Avg%", "Impact")
	if color {
		b.WriteString(ansiDim + colHeader + ansiReset + "\n")
		b.WriteString(ansiDim + "  " + divider + ansiReset + "\n")
	} else {
		b.WriteString(colHeader + "\n")
		b.WriteString("  " + divider + "\n")
	}

	for i, s := range summaries {
		warn := ""
		if s.ZeroCount > 0 {
			warn = fmt.Sprintf("  (%d calls at 0%%)", s.ZeroCount)
		}
		impact := renderBar(float64(s.SavedTokens), float64(maxSaved), 20, color)
		b.WriteString(fmt.Sprintf("  %-3d  %-20s  %5d  %8s  %4.0f%%  %s%s\n",
			i+1, s.BaseCommand, s.Count, formatNum(s.SavedTokens), s.SavingsPct, impact, warn))
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %d command(s) tracked\n", len(summaries)))

	return b.String()
}

// FormatProjectSummary formats per-project aggregate savings.
func FormatProjectSummary(summaries []ProjectSummary) string {
	if len(summaries) == 0 {
		return "no projects tracked yet"
	}

	color := IsColorEnabled()
	var b strings.Builder
	const headerText = "Chop \u2014 Token Savings by Project"
	sep := strings.Repeat("\u2550", 66)
	divider := strings.Repeat("\u2500", 64)

	if color {
		b.WriteString(ansiBold + headerText + ansiReset + "\n")
		b.WriteString(ansiDim + sep + ansiReset + "\n")
	} else {
		b.WriteString(headerText + "\n")
		b.WriteString(sep + "\n")
	}
	b.WriteString("\n")

	colHeader := fmt.Sprintf("  %-40s  %8s  %8s  %6s", "Project", "Commands", "Saved", "Avg%")
	if color {
		b.WriteString(ansiDim + colHeader + ansiReset + "\n")
		b.WriteString(ansiDim + "  " + divider + ansiReset + "\n")
	} else {
		b.WriteString(colHeader + "\n")
		b.WriteString("  " + divider + "\n")
	}

	for _, s := range summaries {
		proj := s.Project
		if proj == "" {
			proj = "(unknown)"
		}
		if len(proj) > 40 {
			proj = "..." + proj[len(proj)-37:]
		}
		b.WriteString(fmt.Sprintf("  %-40s  %8s  %8s  %5.0f%%\n",
			proj, formatNum(s.Count), formatNum(s.SavedTokens), s.SavingsPct))
	}

	return b.String()
}

// formatCompact formats a large integer compactly: 1234 → "1.2K", 1234567 → "1.2M".
func formatCompact(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// renderBar renders a progress bar of the given width using Unicode block characters.
// pct/maxPct determines the filled fraction. When colored is true, ANSI codes are applied.
func renderBar(pct, maxPct float64, width int, colored bool) string {
	const full = "\u2588" // █
	const empty = "\u2591" // ░
	if maxPct <= 0 || width <= 0 {
		e := strings.Repeat(empty, width)
		if colored {
			return ansiDim + e + ansiReset
		}
		return e
	}
	filled := int(pct/maxPct*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}
	filledStr := strings.Repeat(full, filled)
	emptyStr := strings.Repeat(empty, width-filled)
	if colored {
		result := ""
		if filled > 0 {
			result += ansiGreen + filledStr + ansiReset
		}
		if filled < width {
			result += ansiDim + emptyStr + ansiReset
		}
		return result
	}
	return filledStr + emptyStr
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

// gitRoot returns the git repository root for the current working directory,
// or the working directory itself if not in a git repo.
// Result is cached for the lifetime of the process.
func gitRoot() string {
	gitRootOnce.Do(func() {
		cmd := exec.Command("git", "rev-parse", "--show-toplevel")
		out, err := cmd.Output()
		if err != nil {
			gitRootVal, _ = os.Getwd()
			return
		}
		gitRootVal = strings.TrimSpace(string(out))
	})
	return gitRootVal
}

// GetProjectSummary returns per-project aggregate stats, sorted by tokens saved descending.
func GetProjectSummary() ([]ProjectSummary, error) {
	if err := Init(); err != nil {
		return nil, err
	}
	rows, err := db.Query(`
		SELECT
			COALESCE(project, '') AS proj,
			COUNT(*) AS cnt,
			COALESCE(SUM(raw_tokens), 0) AS raw,
			COALESCE(SUM(raw_tokens - filtered_tokens), 0) AS saved
		FROM tracking
		GROUP BY proj
		ORDER BY saved DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []ProjectSummary
	for rows.Next() {
		var s ProjectSummary
		if err := rows.Scan(&s.Project, &s.Count, &s.RawTokens, &s.SavedTokens); err != nil {
			return nil, err
		}
		if s.RawTokens > 0 {
			s.SavingsPct = float64(s.SavedTokens) / float64(s.RawTokens) * 100.0
		}
		summaries = append(summaries, s)
	}
	return summaries, rows.Err()
}

func escapeLike(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
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
		`SELECT timestamp, command, raw_tokens, filtered_tokens, savings_pct, COALESCE(project, '')
         FROM tracking WHERE timestamp >= ? ORDER BY id DESC LIMIT ?`, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []Record
	for rows.Next() {
		var r Record
		if err := rows.Scan(&r.Timestamp, &r.Command, &r.RawTokens, &r.FilteredTokens, &r.SavingsPct, &r.Project); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// FormatGainSince formats a stats report for a --since time window.
func FormatGainSince(s Stats, sinceStr string) string {
	color := IsColorEnabled()
	var b strings.Builder

	headerText := fmt.Sprintf("Chop \u2014 Token Savings (last %s)", sinceStr)
	sep := strings.Repeat("\u2550", 46)

	if color {
		b.WriteString(ansiBold + headerText + ansiReset + "\n")
		b.WriteString(ansiDim + sep + ansiReset + "\n")
	} else {
		b.WriteString(headerText + "\n")
		b.WriteString(sep + "\n")
	}
	b.WriteString("\n")

	b.WriteString(fmt.Sprintf("  Commands:      %s\n", formatNum(s.TotalCommands)))
	b.WriteString(fmt.Sprintf("  Tokens saved:  %s\n", formatNum(s.TotalSavedTokens)))
	bar := renderBar(s.OverallSavingsPct, 100.0, 20, color)
	b.WriteString(fmt.Sprintf("  Efficiency:    %s  %.1f%%\n", bar, s.OverallSavingsPct))
	b.WriteString("\n")

	hint := fmt.Sprintf("  chop gain --since %s --history   recent commands\n", sinceStr)
	if color {
		b.WriteString(ansiDim + hint + ansiReset)
	} else {
		b.WriteString(hint)
	}

	return b.String()
}

// ParseSinceDuration parses duration strings like "7d", "2w", "24h", "30m", "1mo".
// Supports: m (minutes), h (hours), d (days), w (weeks), mo (months ≈ 30 days).
// Falls back to time.ParseDuration for standard Go duration strings.
func ParseSinceDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration %q", s)
	}
	// Check two-char suffix "mo" first.
	if strings.HasSuffix(s, "mo") {
		value := s[:len(s)-2]
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err == nil {
			return time.Duration(n) * 30 * 24 * time.Hour, nil
		}
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
