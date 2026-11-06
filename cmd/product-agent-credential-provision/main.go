package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	credential "task-processor/internal/aicapability/channelcredential"
	"time"

	"task-processor/internal/app/runtime/currentapplication"
	platformdatabase "task-processor/internal/platform/database"
)

func main() {
	manifestPath := flag.String("config", "", "absolute private current-application manifest path")
	inputPath := flag.String("input", "", "absolute private channel credential input path (declared organization connection only)")
	flag.Parse()
	if *manifestPath == "" || *inputPath == "" {
		fmt.Fprintln(os.Stderr, "-config and -input are required")
		os.Exit(2)
	}
	if err := run(*manifestPath, *inputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(manifestPath, inputPath string) error {
	cfg, err := currentapplication.LoadConfig(manifestPath)
	if err != nil {
		return fmt.Errorf("private current-application manifest is invalid")
	}
	input, err := currentapplication.LoadChannelCredentialProvision(inputPath)
	if err != nil || currentapplication.ValidateChannelCredentialProvision(cfg, input) != nil {
		return fmt.Errorf("private credential input or target is invalid")
	}
	writer := input.WriterDatabase
	dbConfig := &platformdatabase.Config{Host: writer.Host, Port: writer.Port, User: writer.User, Password: writer.Password, Database: writer.Database, MaxConnections: writer.MaxConnections, MaxIdleConnections: writer.MaxConnections, ConnectionMaxLifetime: time.Hour}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := platformdatabase.OpenExistingWritableContext(ctx, dbConfig)
	if err != nil {
		return fmt.Errorf("credential writer database is unavailable")
	}
	defer platformdatabase.Close(db)
	result, err := currentapplication.SaveChannelCredential(ctx, cfg, input, db)
	if errors.Is(err, credential.ErrOutcomeUnknown) {
		return fmt.Errorf("credential commit outcome UNKNOWN; retry only the identical requestKey and payload")
	}
	if err != nil {
		return fmt.Errorf("credential provision failed")
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		return fmt.Errorf("write non-secret route result failed")
	}
	return nil
}
