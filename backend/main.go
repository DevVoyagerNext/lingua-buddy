package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

type Entry struct {
	Word         string
	Phonetic     sql.NullString
	Definition   sql.NullString
	Translation  sql.NullString
	PartOfSpeech sql.NullString
	Exchange     sql.NullString
}

func main() {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("读取 .env 失败: %v", err)
	}

	if len(os.Args) != 2 {
		log.Fatalf("用法: go run . <单词>，例如: go run . apple")
	}

	db, err := openDatabase()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry, err := findEntry(ctx, db, os.Args[1])
	if errors.Is(err, sql.ErrNoRows) {
		log.Fatalf("没有找到单词 %q", os.Args[1])
	}
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}

	printEntry(entry)
}

func openDatabase() (*sql.DB, error) {
	user := os.Getenv("DB_USER")
	if user == "" {
		return nil, errors.New("请先设置环境变量 DB_USER")
	}

	cfg := mysql.Config{
		User:                 user,
		Passwd:               os.Getenv("DB_PASSWORD"),
		Net:                  "tcp",
		Addr:                 envOrDefault("DB_ADDR", "127.0.0.1:3306"),
		DBName:               envOrDefault("DB_NAME", "lingua"),
		ParseTime:            true,
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}

	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("创建数据库连接失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	return db, nil
}

func findEntry(ctx context.Context, db *sql.DB, word string) (Entry, error) {
	const query = `
		SELECT word, phonetic, definition, translation, pos, exchange
		FROM ecdict
		WHERE word = ?
		LIMIT 1`

	var entry Entry
	err := db.QueryRowContext(ctx, query, strings.TrimSpace(word)).Scan(
		&entry.Word,
		&entry.Phonetic,
		&entry.Definition,
		&entry.Translation,
		&entry.PartOfSpeech,
		&entry.Exchange,
	)
	return entry, err
}

func printEntry(entry Entry) {
	fmt.Printf("单词: %s\n", entry.Word)
	printField("音标", entry.Phonetic)
	printField("词性", entry.PartOfSpeech)
	printField("英文释义", entry.Definition)
	printField("中文释义", entry.Translation)
	printField("词形变化", entry.Exchange)
}

func printField(label string, value sql.NullString) {
	if !value.Valid || value.String == "" {
		return
	}
	fmt.Printf("%s:\n%s\n", label, strings.ReplaceAll(value.String, `\n`, "\n"))
}

func envOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
